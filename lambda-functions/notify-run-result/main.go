package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
	"github.com/google/uuid"
	"github.com/hashicorp/go-tfe"
	"log"
	"os"
)

type NotifyRunResultRequest struct {
	TerraformRunId          string                  `json:"terraformRunId"`
	WorkflowToken           string                  `json:"workflowToken"`
	RecordId                string                  `json:"recordId"`
	TracerTag               TracerTag               `json:"tracerTag"`
	ServiceCatalogOperation ServiceCatalogOperation `json:"serviceCatalogOperation"`
	AwsAccountId            string                  `json:"awsAccountId"`
	TerraformOrganization   string                  `json:"terraformOrganization"`
	ProvisionedProductId    string                  `json:"provisionedProductId"`
	Error                   string                  `json:"error"`
	ErrorMessage            string                  `json:"errorMessage"`
}

type ServiceCatalogOperation string

// Enum values for ServiceCatalogOperation
const (
	Terminating  ServiceCatalogOperation = "TERMINATING"
	Provisioning ServiceCatalogOperation = "PROVISIONING"
)

type TracerTag struct {
	TracerTagKey   string `json:"key"`
	TracerTagValue string `json:"value"`
}

type NotifyRunResultResponse struct {
	Name string `json:"terraformRunId"`
}

func HandleRequest(ctx context.Context, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	serviceCatalogClient := servicecatalog.NewFromConfig(sdkConfig)

	tfeClient, err := getTFEClient(ctx, sdkConfig)
	if err != nil {
		return nil, err
	}

	switch {
	case request.ServiceCatalogOperation == Terminating:
		return NotifyTerminateResult(ctx, serviceCatalogClient, tfeClient, request)
	case request.ServiceCatalogOperation == Provisioning:
		return NotifyProvisioningResult(ctx, serviceCatalogClient, tfeClient, request)
	default:
		log.Printf("Unknown serviceCatalogOperation: %s\n", request.ServiceCatalogOperation)
		return nil, errors.New("unknown serviceCatalogOperation")
	}
}

func NotifyTerminateResult(ctx context.Context, scClient *servicecatalog.Client, tfeClient *tfe.Client, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	// Delete the workspace
	err := DeleteWorkspace(ctx, tfeClient, request)
	if err != nil {
		request.ErrorMessage = err.Error()
	}

	var status = types.EngineWorkflowStatusSucceeded
	var failureReason *string = nil
	if request.ErrorMessage != "" {
		failureReason = aws.String(request.ErrorMessage)
		status = types.EngineWorkflowStatusFailed
	}

	log.Printf("Notifying terminate result %s\n", status)
	_, err = scClient.NotifyTerminateProvisionedProductEngineWorkflowResult(
		ctx,
		&servicecatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput{
			WorkflowToken:    &request.WorkflowToken,
			RecordId:         &request.RecordId,
			Status:           status,
			FailureReason:    failureReason,
			IdempotencyToken: tfe.String(uuid.New().String()),
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	return nil, err
}

func NotifyProvisioningResult(ctx context.Context, scClient *servicecatalog.Client, tfeClient *tfe.Client, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	outputs, err := FetchRunOutputs(ctx, tfeClient, request.TerraformRunId)
	if err != nil {
		return nil, err
	}

	var status = types.EngineWorkflowStatusSucceeded
	var failureReason *string = nil
	if request.ErrorMessage != "" {
		failureReason = FormatError(request.Error, request.ErrorMessage)
		status = types.EngineWorkflowStatusFailed
	}

	log.Printf("Notifying provision result %s\n", status)
	_, err = scClient.NotifyProvisionProductEngineWorkflowResult(
		ctx,
		&servicecatalog.NotifyProvisionProductEngineWorkflowResultInput{
			WorkflowToken:    &request.WorkflowToken,
			RecordId:         &request.RecordId,
			Status:           status,
			FailureReason:    failureReason,
			IdempotencyToken: tfe.String(uuid.New().String()),
			Outputs:          outputs,
			ResourceIdentifier: &types.EngineWorkflowResourceIdentifier{
				UniqueTag: &types.UniqueTagResourceIdentifier{
					Key:   tfe.String(request.TracerTag.TracerTagKey),
					Value: tfe.String(request.TracerTag.TracerTagValue),
				},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	return nil, err
}

func FormatError(err string, errorMessage string) *string {
	// Check if error was due to lambda timeout
	if err == "States.Timeout" {
		return aws.String("A lambda function invoked by the state machine has timed out")
	}

	// Max error message length is 2048
	if len(errorMessage) <= (2048) {
		return aws.String(errorMessage)
	}

	// Truncate error message to fit maximum failure reason length allowed by Service Catalog.
	// We use 2045 to make room for the ellipsis.
	return aws.String(errorMessage[:2045] + "...")
}

type TFECredentialsSecret struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token"`
}

func getTFEClient(ctx context.Context, sdkConfig aws.Config) (*tfe.Client, error) {
	// Create secrets client SDK to fetch TFE credentials
	secretsManagerClient := secretsmanager.NewFromConfig(sdkConfig)

	// Fetch the TFE credentials/config from AWS Secrets Manager
	secretId := os.Getenv("TFE_CREDENTIALS_SECRET_ID")
	versionId := os.Getenv("TFE_CREDENTIALS_SECRET_VERSION_ID")

	tfeCredentialsSecretJson, err := secretsManagerClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:  aws.String(secretId),
		VersionId: aws.String(versionId),
	})
	if err != nil {
		return nil, err
	}

	// Decode the response from AWS Secrets Manager
	var tfeCredentialsSecret TFECredentialsSecret
	if err = json.Unmarshal([]byte(*tfeCredentialsSecretJson.SecretString), &tfeCredentialsSecret); err != nil {
		return nil, err
	}

	// Use the credentials to create a TFE client
	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", tfeCredentialsSecret.Hostname),
		Token:   tfeCredentialsSecret.Token,
	})

	return client, err
}

func main() {
	lambda.Start(HandleRequest)
}

func DeleteWorkspace(ctx context.Context, client *tfe.Client, request NotifyRunResultRequest) error {
	// Get workspace name
	workspaceName := getWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)

	// Make a call to delete workspace
	err := client.Workspaces.Delete(ctx, request.TerraformOrganization, workspaceName)

	return err
}

// Get the workspace name, which is `${accountId} - ${provisionedProductId}`
func getWorkspaceName(awsAccountId string, provisionedProductId string) string {
	return fmt.Sprintf("%s-%s", awsAccountId, provisionedProductId)
}
