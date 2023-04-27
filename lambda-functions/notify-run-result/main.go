package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
	"github.com/google/uuid"
	"github.com/hashicorp/go-tfe"
	"log"
)

type NotifyRunResultRequest struct {
	WorkflowToken string    `json:"workflowToken"`
	RecordId      string    `json:"recordId"`
	TracerTag     TracerTag `json:"tracerTag"`
}

type TracerTag struct {
	TracerTagKey   string `json:"key"`
	TracerTagValue string `json:"value"`
}

type NotifyRunResultResponse struct {
	Name string `json:"terraformRunId"`
}

func HandleRequest(ctx context.Context, request NotifyRunResultRequest) (NotifyRunResultResponse, error) {

	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
		return NotifyRunResultResponse{}, err
	}
	serviceCatalogClient := servicecatalog.NewFromConfig(sdkConfig)

	_, err = serviceCatalogClient.NotifyProvisionProductEngineWorkflowResult(
		ctx,
		&servicecatalog.NotifyProvisionProductEngineWorkflowResultInput{
			WorkflowToken:    &request.WorkflowToken,
			RecordId:         &request.RecordId,
			Status:           types.EngineWorkflowStatusSucceeded,
			FailureReason:    nil,
			IdempotencyToken: tfe.String(uuid.New().String()),
			Outputs:          []types.RecordOutput{},
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

	return NotifyRunResultResponse{}, err
}

func main() {
	lambda.Start(HandleRequest)
}
