package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
)

type ProvisioningOperationsHandlerRequest struct {
	AwsRegion            string `json:"awsRegion"`
	StateMachineArn      string `json:"awsStateMachineArn"`
	Records              string `json:"records"`
	Body                 string `json:"body"`
	MessageId            string `json:"MessageId"`
	Token                string `json:"token"`
	ProvisionedProductId string `json:"provisionedProductId"`
	RecordId             string `json:"recordId"`
}

type ProvisioningOperationsHandlerResponse struct{}

func HandleRequest(ctx context.Context, request ProvisioningOperationsHandlerRequest) (*ProvisioningOperationsHandlerResponse, error) {
	return nil, nil
}

func main() {
	lambda.Start(HandleRequest)
}
