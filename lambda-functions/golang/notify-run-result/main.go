package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
	"github.com/google/uuid"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfeauth"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
	"github.com/hashicorp/go-tfe"
	"log"
)

type NotifyRunResultRequest struct {
	TerraformRunId          string                  `json:"terraformRunId"`
	WorkflowToken           string                  `json:"workflowToken"`
	RecordId                string                  `json:"recordId"`
	TracerTag               tracertag.TracerTag     `json:"tracerTag"`
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
	Updating     ServiceCatalogOperation = "UPDATING"
)

type NotifyRunResultResponse struct{}

func HandleRequest(ctx context.Context, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	serviceCatalogClient := servicecatalog.NewFromConfig(sdkConfig)

	tfeClient, err := tfeauth.GetTFEClient(ctx, sdkConfig)
	if err != nil {
		return nil, err
	}

	switch {
	case request.ServiceCatalogOperation == Terminating:
		return NotifyTerminateResult(ctx, serviceCatalogClient, tfeClient, request)
	case request.ServiceCatalogOperation == Provisioning:
		return NotifyProvisioningResult(ctx, serviceCatalogClient, tfeClient, request)
	case request.ServiceCatalogOperation == Updating:
		return NotifyUpdatingResult(ctx, serviceCatalogClient, tfeClient, request)
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
	var outputs []types.RecordOutput
	var err error

	var status = types.EngineWorkflowStatusSucceeded
	var failureReason *string = nil
	if request.ErrorMessage != "" {
		failureReason = FormatError(request.Error, request.ErrorMessage)
		status = types.EngineWorkflowStatusFailed
	} else {
		outputs, err = FetchRunOutputs(ctx, tfeClient, request)
		if err != nil {
			return nil, err
		}
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

func NotifyUpdatingResult(ctx context.Context, scClient *servicecatalog.Client, tfeClient *tfe.Client, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	var outputs []types.RecordOutput
	var err error

	var status = types.EngineWorkflowStatusSucceeded
	var failureReason *string = nil
	if request.ErrorMessage != "" {
		failureReason = FormatError(request.Error, request.ErrorMessage)
		status = types.EngineWorkflowStatusFailed
	} else {
		outputs, err = FetchRunOutputs(ctx, tfeClient, request)
		if err != nil {
			return nil, err
		}
	}

	log.Printf("Notifying update result %s\n", status)
	_, err = scClient.NotifyUpdateProvisionedProductEngineWorkflowResult(
		ctx,
		&servicecatalog.NotifyUpdateProvisionedProductEngineWorkflowResultInput{
			IdempotencyToken: tfe.String(uuid.New().String()),
			RecordId:         &request.RecordId,
			Status:           status,
			WorkflowToken:    &request.WorkflowToken,
			FailureReason:    failureReason,
			Outputs:          outputs,
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
