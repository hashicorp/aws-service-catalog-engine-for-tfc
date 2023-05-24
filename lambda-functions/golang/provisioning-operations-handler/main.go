package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/awsconfig"
	"log"
	"os"
)

type ProvisioningOperationsHandlerRequest struct {
	Records []Record `json:"Records"`
}

type Record struct {
	MessageId string `json:"messageId"`
	Body      string `json:"body"`
}

type StateMachinePayload struct {
	Token                string `json:"token"`
	ProvisionedProductId string `json:"provisionedProductId"`
	RecordId             string `json:"recordId"`
}

type ProvisioningOperationsHandlerResponse struct {
	BatchItemFailures []BatchItemFailure `json:"batchItemFailures"`
}

type BatchItemFailure struct {
	ItemIdentifier string `json:"itemIdentifier"`
}

func HandleRequest(ctx context.Context, request ProvisioningOperationsHandlerRequest) (*ProvisioningOperationsHandlerResponse, error) {
	sdkConfig := awsconfig.GetSdkConfig(ctx)
	sfnClient := sfn.NewFromConfig(sdkConfig)

	response := ProvisioningOperationsHandlerResponse{}
	for _, record := range request.Records {
		err := StartStateMachineExecution(ctx, sfnClient, record)
		if err != nil {
			log.Default().Printf("Failed to start state machine execution for record, cause: %s", err.Error())

			// Add the failure to the dead letter queue
			response.BatchItemFailures = append(response.BatchItemFailures, BatchItemFailure{ItemIdentifier: record.MessageId})
		}
	}

	return &ProvisioningOperationsHandlerResponse{}, nil
}

func StartStateMachineExecution(ctx context.Context, client *sfn.Client, record Record) error {
	var stateMachinePayload StateMachinePayload
	if err := json.Unmarshal([]byte(record.Body), &stateMachinePayload); err != nil {
		return err
	}

	stateMachineArn := os.Getenv("STATE_MACHINE_ARN")
	executionName := fmt.Sprintf("%s-%s", stateMachinePayload.ProvisionedProductId, stateMachinePayload.RecordId)
	execution, err := client.StartExecution(ctx, &sfn.StartExecutionInput{
		StateMachineArn: &stateMachineArn,
		Input:           &record.Body,
		Name:            &executionName,
	})
	if err != nil {
		return err
	}

	log.Default().Printf("Started state machine execution with arn: %s for request Id: %s", execution.ExecutionArn, execution.ResultMetadata.Get("RequestId"))

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
