package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/awsconfig"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/fileutils"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/identifiers"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
	"github.com/hashicorp/go-tfe"
	"log"
	"strings"
	"time"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
)

type SendApplyRequest struct {
	AwsAccountId          string              `json:"awsAccountId"`
	TerraformOrganization string              `json:"terraformOrganization"`
	ProvisionedProductId  string              `json:"provisionedProductId"`
	Artifact              Artifact            `json:"artifact"`
	LaunchRoleArn         string              `json:"launchRoleArn"`
	ProductId             string              `json:"productId"`
	Tags                  []AWSTag            `json:"tags"`
	TracerTag             tracertag.TracerTag `json:"tracerTag"`
}

type AWSTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Artifact struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type SendApplyResponse struct {
	TerraformRunId string `json:"terraformRunId"`
}

func HandleRequest(ctx context.Context, request SendApplyRequest) (*SendApplyResponse, error) {
	sdkConfig := awsconfig.GetSdkConfig(ctx)

	s3Client := s3.NewFromConfig(sdkConfig)

	client, err := tfc.GetTFEClient(ctx, sdkConfig)
	if err != nil {
		return nil, err
	}

	// Find or create the Project
	projectName := request.ProductId
	p, err := FindOrCreateProject(ctx, client, request.TerraformOrganization, projectName)
	if err != nil {
		return nil, err
	}

	// Create or find the Workspace
	workspaceName := identifiers.GetWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)
	w, err := FindOrCreateWorkspace(ctx, client, request.TerraformOrganization, p, workspaceName)
	if err != nil {
		return nil, err
	}

	// Configure ENV variables for OIDC
	err = UpdateWorkspaceVariables(ctx, client, w, request.LaunchRoleArn)
	if err != nil {
		return nil, err
	}

	cv, err := client.ConfigurationVersions.Create(ctx,
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
	sourceProductConfig, err := fileutils.DownloadS3File(ctx, key, bucket, s3Client)
	if err != nil {
		return nil, err
	}

	// Create override files for injecting AWS default tags
	providerOverrides, _ := CreateAWSProviderOverrides(sdkConfig.Region, request.Tags, request.TracerTag)

	// Inject AWS default tags, via the override file, into the tar file
	modifiedProductConfig, err := InjectOverrides(sourceProductConfig, []ConfigurationOverride{*providerOverrides})
	if err != nil {
		return nil, err
	}

	// Upload newly modified configuration to TFE
	err = client.ConfigurationVersions.UploadTarGzip(ctx, cv.UploadURL, modifiedProductConfig)
	if err != nil {
		return nil, err
	}

	uploadTimeoutInSeconds := 120
	for i := 0; ; i++ {
		refreshed, err := client.ConfigurationVersions.Read(ctx, cv.ID)

		if refreshed.Status == tfe.ConfigurationUploaded {
			break
		}

		if i > uploadTimeoutInSeconds {
			return nil, err
		}

		time.Sleep(1 * time.Second)
	}

	run, err := client.Runs.Create(ctx, tfe.RunCreateOptions{
		Workspace:            w,
		ConfigurationVersion: cv,
		AutoApply:            tfe.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	return &SendApplyResponse{TerraformRunId: run.ID}, err
}

func FindOrCreateProject(ctx context.Context, client *tfe.Client, organizationName string, name string) (*tfe.Project, error) {
	// Check if the Project already exists...
	projects, err := client.Projects.List(ctx, organizationName, &tfe.ProjectListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 0,
			PageSize:   100,
		},
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	for _, project := range projects.Items {
		// Check for exact name match, because the search we made is a partial search
		if project.Name == name {
			return project, nil
		}
	}

	// Otherwise, create the Project
	return client.Projects.Create(ctx, organizationName, tfe.ProjectCreateOptions{
		Name: name,
	})
}

func main() {
	lambda.Start(HandleRequest)
}

// Resolves artifactPath to bucket and key
func resolveArtifactPath(artifactPath string) (string, string) {
	bucket := strings.Split(artifactPath, "/")[2]
	key := strings.SplitN(artifactPath, "/", 4)[3]
	return bucket, key
}

func FindOrCreateWorkspace(ctx context.Context, client *tfe.Client, organizationName string, project *tfe.Project, workspaceName string) (*tfe.Workspace, error) {
	// Check if the workspace already exists...
	workspaces, err := client.Workspaces.List(ctx, organizationName, &tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 0,
			PageSize:   100,
		},
		Search: workspaceName,
	})
	if err != nil {
		return nil, err
	}

	for _, workspace := range workspaces.Items {
		// Check for exact name match, because the search we made is a partial search
		if workspace.Name == workspaceName {
			return workspace, nil
		}
	}

	// Otherwise, create the Workspace
	return client.Workspaces.Create(ctx, organizationName, tfe.WorkspaceCreateOptions{
		Name:    tfe.String(workspaceName),
		Project: project,
	})
}

func UpdateWorkspaceVariables(ctx context.Context, client *tfe.Client, w *tfe.Workspace, launchRoleArn string) error {
	log.Default().Print("Updating variable TFC_AWS_PROVIDER_AUTH")
	err := FindOrCreateVariable(ctx, client, w, "TFC_AWS_PROVIDER_AUTH", "true", "Enable the Workload Identity integration for AWS.")
	if err != nil {
		return err
	}

	log.Default().Print("Updating variable TFC_AWS_RUN_ROLE_ARN")
	return FindOrCreateVariable(ctx, client, w, "TFC_AWS_RUN_ROLE_ARN", launchRoleArn, "The AWS role arn runs will use to authenticate.")
}

func FindOrCreateVariable(ctx context.Context, client *tfe.Client, w *tfe.Workspace, key string, value string, description string) error {
	// TODO: Update to support workspaces that contain more than 100 variables
	variables, err := client.Variables.List(ctx, w.ID, &tfe.VariableListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 0,
			PageSize:   100,
		},
	})
	if err != nil {
		return err
	}

	var variableToUpdate *tfe.Variable
	for _, v := range variables.Items {
		if v.Key == key {
			variableToUpdate = v
			break
		}
	}

	if variableToUpdate != nil {
		// Update the variables
		log.Default().Printf("Updating variable with ID: %s", variableToUpdate.ID)
		_, err = client.Variables.Update(ctx, w.ID, variableToUpdate.ID, tfe.VariableUpdateOptions{
			Key:      tfe.String(key),
			Value:    tfe.String(value),
			Category: tfe.Category(tfe.CategoryEnv),
			HCL:      tfe.Bool(false),
		})
		return err
	}

	// Create the variable as it does not currently exist
	_, err = client.Variables.Create(ctx, w.ID, tfe.VariableCreateOptions{
		Key:         tfe.String(key),
		Value:       tfe.String(value),
		Description: tfe.String(description),
		Category:    tfe.Category(tfe.CategoryEnv),
		HCL:         tfe.Bool(false),
		Sensitive:   tfe.Bool(false),
	})
	return err
}
