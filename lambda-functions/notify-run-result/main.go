package main

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
	"github.com/google/uuid"
	"github.com/hashicorp/go-tfe"
	"log"
)

type NotifyRunResultRequest struct {
	WorkflowToken           string                  `json:"workflowToken"`
	RecordId                string                  `json:"recordId"`
	TracerTag               TracerTag               `json:"tracerTag"`
	ServiceCatalogOperation ServiceCatalogOperation `json:"serviceCatalogOperation"`
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
		log.Fatal(err)
		return nil, err
	}
	serviceCatalogClient := servicecatalog.NewFromConfig(sdkConfig)

	switch {
	case request.ServiceCatalogOperation == Terminating:
		return NotifyTerminateResult(ctx, serviceCatalogClient, request)
	case request.ServiceCatalogOperation == Provisioning:
		return NotifyProvisioningResult(ctx, serviceCatalogClient, request)
	default:
		return nil, errors.New("unknown serviceCatalogOperation")
	}
}

func NotifyTerminateResult(ctx context.Context, client *servicecatalog.Client, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	_, err := client.NotifyTerminateProvisionedProductEngineWorkflowResult(
		ctx,
		&servicecatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput{
			WorkflowToken:    &request.WorkflowToken,
			RecordId:         &request.RecordId,
			Status:           types.EngineWorkflowStatusSucceeded,
			FailureReason:    nil,
			IdempotencyToken: tfe.String(uuid.New().String()),
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	return nil, err
}

func NotifyProvisioningResult(ctx context.Context, client *servicecatalog.Client, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	_, err := client.NotifyProvisionProductEngineWorkflowResult(
		ctx,
		&servicecatalog.NotifyProvisionProductEngineWorkflowResultInput{
			WorkflowToken:    &request.WorkflowToken,
			RecordId:         &request.RecordId,
			Status:           types.EngineWorkflowStatusSucceeded,
			FailureReason:    nil,
			IdempotencyToken: tfe.String(uuid.New().String()),
			// TODO: Parse outputs here
			Outputs: []types.RecordOutput{},
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

func main() {
	lambda.Start(HandleRequest)
}
