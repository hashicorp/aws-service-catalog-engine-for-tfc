package main

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/go-tfe"
	"log"
	"net/http"
)

type TFCApplier struct {
	tfeClient *tfe.Client
}

func (h *SendApplyHandler) NewTFCApplier(ctx context.Context, request SendApplyRequest) (*TFCApplier, error) {
	headers := http.Header{}

	headers.Set("Tfp-Aws-Service-Catalog-Product-Id", request.ProductId)
	headers.Set("Tfp-Aws-Service-Catalog-Prv-Product-Id", request.ProvisionedProductId)
	//headers.Set("Tfp-Aws-Service-Catalog-Portfolio-Id", request)
	//headers.Set("Tfp-Aws-Service-Catalog-Product-Ver", request)

	tfeClient, err := tfc.GetTFEClientWithHeaders(ctx, h.secretsManager, headers)
	return &TFCApplier{tfeClient: tfeClient}, err
}

func (applier *TFCApplier) FindOrCreateProject(ctx context.Context, organizationName string, name string) (*tfe.Project, error) {
	// Check if the project already exists...
	project, err := applier.FindProjectByName(ctx, organizationName, name, 0)
	if project != nil || err != nil {
		return project, err
	}

	// Otherwise, create the project
	return applier.tfeClient.Projects.Create(ctx, organizationName, tfe.ProjectCreateOptions{
		Name: name,
	})
}

func (applier *TFCApplier) FindProjectByName(ctx context.Context, organizationName string, projectName string, pageNumber int) (*tfe.Project, error) {
	// Check if the project already exists...
	projects, err := applier.tfeClient.Projects.List(ctx, organizationName, &tfe.ProjectListOptions{
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
		return applier.FindProjectByName(ctx, organizationName, projectName, pageNumber+1)
	}

	return nil, nil
}

func (applier *TFCApplier) FindOrCreateWorkspace(ctx context.Context, organizationName string, project *tfe.Project, workspaceName string) (*tfe.Workspace, error) {
	// Check if the workspace already exists...
	workspace, err := applier.FindWorkspaceByName(ctx, organizationName, workspaceName, 0)
	if workspace != nil || err != nil {
		return workspace, err
	}

	// Otherwise, create the Workspace
	return applier.tfeClient.Workspaces.Create(ctx, organizationName, tfe.WorkspaceCreateOptions{
		Name:    tfe.String(workspaceName),
		Project: project,
	})
}

func (applier *TFCApplier) FindWorkspaceByName(ctx context.Context, organizationName string, workspaceName string, pageNumber int) (*tfe.Workspace, error) {
	// Check if the workspace already exists...
	workspaces, err := applier.tfeClient.Workspaces.List(ctx, organizationName, &tfe.WorkspaceListOptions{
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
		return applier.FindWorkspaceByName(ctx, organizationName, workspaceName, pageNumber+1)
	}

	return nil, nil
}

func (applier *TFCApplier) UpdateWorkspaceVariables(ctx context.Context, w *tfe.Workspace, launchRoleArn string) error {
	log.Default().Print("Updating variable TFC_AWS_PROVIDER_AUTH")
	err := applier.FindOrCreateVariable(ctx, w, "TFC_AWS_PROVIDER_AUTH", "true", "Enable the Workload Identity integration for AWS.")
	if err != nil {
		return err
	}

	log.Default().Print("Updating variable TFC_AWS_RUN_ROLE_ARN")
	return applier.FindOrCreateVariable(ctx, w, "TFC_AWS_RUN_ROLE_ARN", launchRoleArn, "The AWS role ARN runs will use to authenticate.")
}

func (applier *TFCApplier) CreateConfigurationVersion(ctx context.Context, workspaceId string) (*tfe.ConfigurationVersion, error) {
	return applier.tfeClient.ConfigurationVersions.Create(ctx,
		workspaceId,
		tfe.ConfigurationVersionCreateOptions{
			// Disable auto queue runs, so we can create the run ourselves to get the runId
			AutoQueueRuns: tfe.Bool(false),
		},
	)
}

func (applier *TFCApplier) FindOrCreateVariable(ctx context.Context, w *tfe.Workspace, key string, value string, description string) error {
	variableToUpdate, err := applier.FindVariableByKey(ctx, w, key, 0)
	if err != nil {
		return err
	}

	if variableToUpdate != nil {
		// Update the variables
		log.Default().Printf("Updating variable with ID: %s", variableToUpdate.ID)
		_, err = applier.tfeClient.Variables.Update(ctx, w.ID, variableToUpdate.ID, tfe.VariableUpdateOptions{
			Key:      tfe.String(key),
			Value:    tfe.String(value),
			Category: tfe.Category(tfe.CategoryEnv),
			HCL:      tfe.Bool(false),
		})
		return err
	}

	// Create the variable as it does not currently exist
	_, err = applier.tfeClient.Variables.Create(ctx, w.ID, tfe.VariableCreateOptions{
		Key:         tfe.String(key),
		Value:       tfe.String(value),
		Description: tfe.String(description),
		Category:    tfe.Category(tfe.CategoryEnv),
		HCL:         tfe.Bool(false),
		Sensitive:   tfe.Bool(false),
	})
	return err
}

func (applier *TFCApplier) FindVariableByKey(ctx context.Context, w *tfe.Workspace, key string, pageNumber int) (*tfe.Variable, error) {
	variables, err := applier.tfeClient.Variables.List(ctx, w.ID, &tfe.VariableListOptions{
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
		return applier.FindVariableByKey(ctx, w, key, pageNumber+1)
	}

	return nil, nil
}
