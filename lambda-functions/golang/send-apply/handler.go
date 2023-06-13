package main

import (
	"github.com/hashicorp/go-tfe"
	"time"
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/identifiers"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/fileutils"
	"log"
	"strings"
)

type SendApplyHandler struct {
	tfeClient    *tfe.Client
	s3Downloader fileutils.S3Downloader
	region       string
}

func (h *SendApplyHandler) HandleRequest(ctx context.Context, request SendApplyRequest) (*SendApplyResponse, error) {

	// Find or create the Project
	projectName := request.ProductId
	p, err := h.FindOrCreateProject(ctx, request.TerraformOrganization, projectName)
	if err != nil {
		return nil, err
	}

	// Create or find the Workspace
	workspaceName := identifiers.GetWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)
	w, err := h.FindOrCreateWorkspace(ctx, request.TerraformOrganization, p, workspaceName)
	if err != nil {
		return nil, err
	}

	// Configure ENV variables for OIDC
	err = h.UpdateWorkspaceVariables(ctx, w, request.LaunchRoleArn)
	if err != nil {
		return nil, err
	}

	cv, err := h.tfeClient.ConfigurationVersions.Create(ctx,
		w.ID,
		tfe.ConfigurationVersionCreateOptions{
			// Disable auto queue runs, so we can create the run ourselves to get the runId
			AutoQueueRuns: tfe.Bool(false),
		},
	)
	if err != nil {
		return nil, err
	}

	// Download product configuration files
	bucket, key := resolveArtifactPath(request.Artifact.Path)
	sourceProductConfig, err := fileutils.DownloadS3File(ctx, h.s3Downloader, key, bucket)
	if err != nil {
		return nil, err
	}

	// Create override files for injecting AWS default tags
	providerOverrides, _ := CreateAWSProviderOverrides(h.region, request.Tags, request.TracerTag)

	// Inject AWS default tags, via the override file, into the tar file
	modifiedProductConfig, err := InjectOverrides(sourceProductConfig, []ConfigurationOverride{*providerOverrides})
	if err != nil {
		return nil, err
	}

	// Upload newly modified configuration to TFE
	err = h.tfeClient.ConfigurationVersions.UploadTarGzip(ctx, cv.UploadURL, modifiedProductConfig)
	if err != nil {
		return nil, err
	}

	uploadTimeoutInSeconds := 120
	for i := 0; ; i++ {
		refreshed, err := h.tfeClient.ConfigurationVersions.Read(ctx, cv.ID)
		if err != nil {
			return nil, err
		}

		if refreshed.Status == tfe.ConfigurationUploaded {
			break
		}

		if i > uploadTimeoutInSeconds {
			return nil, err
		}

		time.Sleep(1 * time.Second)
	}

	run, err := h.tfeClient.Runs.Create(ctx, tfe.RunCreateOptions{
		Workspace:            w,
		ConfigurationVersion: cv,
		AutoApply:            tfe.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	return &SendApplyResponse{TerraformRunId: run.ID}, err
}

func (h *SendApplyHandler) FindOrCreateProject(ctx context.Context, organizationName string, name string) (*tfe.Project, error) {
	// Check if the project already exists...
	project, err := h.FindProjectByName(ctx, organizationName, name, 0)
	if project != nil || err != nil {
		return project, err
	}

	// Otherwise, create the project
	return h.tfeClient.Projects.Create(ctx, organizationName, tfe.ProjectCreateOptions{
		Name: name,
	})
}

func (h *SendApplyHandler) FindProjectByName(ctx context.Context, organizationName string, projectName string, pageNumber int) (*tfe.Project, error) {
	// Check if the project already exists...
	projects, err := h.tfeClient.Projects.List(ctx, organizationName, &tfe.ProjectListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: pageNumber,
			PageSize:   100,
		},
		Name: projectName,
	})
	if err != nil {
		return nil, err
	}

	for _, project := range projects.Items {
		// Check for exact name match, because the search we made is a "contains" search
		if project.Name == projectName {
			return project, nil
		}
	}

	// If more projects exists, fetch them and check them as well
	if projects.TotalCount > ((pageNumber + 1) * 100) {
		return h.FindProjectByName(ctx, organizationName, projectName, pageNumber+1)
	}

	return nil, nil
}

func (h *SendApplyHandler) FindOrCreateWorkspace(ctx context.Context, organizationName string, project *tfe.Project, workspaceName string) (*tfe.Workspace, error) {
	// Check if the workspace already exists...
	workspace, err := h.FindWorkspaceByName(ctx, organizationName, workspaceName, 0)
	if workspace != nil || err != nil {
		return workspace, err
	}

	// Otherwise, create the Workspace
	return h.tfeClient.Workspaces.Create(ctx, organizationName, tfe.WorkspaceCreateOptions{
		Name:    tfe.String(workspaceName),
		Project: project,
	})
}

func (h *SendApplyHandler) FindWorkspaceByName(ctx context.Context, organizationName string, workspaceName string, pageNumber int) (*tfe.Workspace, error) {
	// Check if the workspace already exists...
	workspaces, err := h.tfeClient.Workspaces.List(ctx, organizationName, &tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: pageNumber,
			PageSize:   100,
		},
		Search: workspaceName,
	})
	if err != nil {
		return nil, err
	}

	for _, workspace := range workspaces.Items {
		// Check for exact name match, because the search we made is a "contains" search
		if workspace.Name == workspaceName {
			return workspace, nil
		}
	}

	// If more workspaces exists, fetch them and check them as well
	if workspaces.TotalCount > ((pageNumber + 1) * 100) {
		return h.FindWorkspaceByName(ctx, organizationName, workspaceName, pageNumber+1)
	}

	return nil, nil
}

func (h *SendApplyHandler) UpdateWorkspaceVariables(ctx context.Context, w *tfe.Workspace, launchRoleArn string) error {
	log.Default().Print("Updating variable TFC_AWS_PROVIDER_AUTH")
	err := h.FindOrCreateVariable(ctx, w, "TFC_AWS_PROVIDER_AUTH", "true", "Enable the Workload Identity integration for AWS.")
	if err != nil {
		return err
	}

	log.Default().Print("Updating variable TFC_AWS_RUN_ROLE_ARN")
	return h.FindOrCreateVariable(ctx, w, "TFC_AWS_RUN_ROLE_ARN", launchRoleArn, "The AWS role arn runs will use to authenticate.")
}

func (h *SendApplyHandler) FindOrCreateVariable(ctx context.Context, w *tfe.Workspace, key string, value string, description string) error {
	variableToUpdate, err := h.FindVariableByKey(ctx, w, key, 0)
	if err != nil {
		return err
	}

	if variableToUpdate != nil {
		// Update the variables
		log.Default().Printf("Updating variable with ID: %s", variableToUpdate.ID)
		_, err = h.tfeClient.Variables.Update(ctx, w.ID, variableToUpdate.ID, tfe.VariableUpdateOptions{
			Key:      tfe.String(key),
			Value:    tfe.String(value),
			Category: tfe.Category(tfe.CategoryEnv),
			HCL:      tfe.Bool(false),
		})
		return err
	}

	// Create the variable as it does not currently exist
	_, err = h.tfeClient.Variables.Create(ctx, w.ID, tfe.VariableCreateOptions{
		Key:         tfe.String(key),
		Value:       tfe.String(value),
		Description: tfe.String(description),
		Category:    tfe.Category(tfe.CategoryEnv),
		HCL:         tfe.Bool(false),
		Sensitive:   tfe.Bool(false),
	})
	return err
}

func (h *SendApplyHandler) FindVariableByKey(ctx context.Context, w *tfe.Workspace, key string, pageNumber int) (*tfe.Variable, error) {
	variables, err := h.tfeClient.Variables.List(ctx, w.ID, &tfe.VariableListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: pageNumber,
			PageSize:   100,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, variable := range variables.Items {
		if variable.Key == key {
			return variable, nil
		}
	}

	// If more variables exists, fetch them and check them as well
	if variables.TotalCount > ((pageNumber + 1) * 100) {
		return h.FindVariableByKey(ctx, w, key, pageNumber+1)
	}

	return nil, nil
}

// Resolves artifactPath to bucket and key
func resolveArtifactPath(artifactPath string) (string, string) {
	bucket := strings.Split(artifactPath, "/")[2]
	key := strings.SplitN(artifactPath, "/", 4)[3]
	return bucket, key
}
