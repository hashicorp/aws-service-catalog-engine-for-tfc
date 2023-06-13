package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/awsconfig"
	"context"
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
	Token                  string `json:"token"`
	Operation              string `json:"operation"`
	ProductId              string `json:"productId"`
	ProvisionedProductId   string `json:"provisionedProductId"`
	ProvisionedProductName string `json:"provisionedProductName"`
	RecordId               string `json:"recordId"`
	LaunchRoleArn          string `json:"launchRoleArn"`
	TerraformOrganization  string `json:"terraformOrganization"`
	Identity               struct {
		Principal      string `json:"principal"`
		AwsAccountId   string `json:"awsAccountId"`
		OrganizationId string `json:"organizationId"`
	} `json:"identity"`
	TracerTag struct {
		Key   string `json:"token"`
		Value string `json:"operation"`
	} `json:"tracerTag"`
	Artifact struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"artifact"`
	Tags []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"tags"`
}

type ProvisioningOperationsHandlerResponse struct {
	BatchItemFailures []BatchItemFailure `json:"batchItemFailures"`
}

type BatchItemFailure struct {
	ItemIdentifier string `json:"itemIdentifier"`
}

func main() {
	// Create temporary context to initialize the handler with
	initContext := context.TODO()

	// Create step functions client
	sdkConfig := awsconfig.GetSdkConfig(initContext)
	sfnClient := sfn.NewFromConfig(sdkConfig)

	// Get Terraform Organization
	terraformOrganization := os.Getenv("TERRAFORM_ORGANIZATION")

	// Get state machine arn
	stateMachineArn := os.Getenv("STATE_MACHINE_ARN")

	handler := ProvisioningOperationsHandler{
		terraformOrganization: terraformOrganization,
		stepFunctions:         SF{Client: sfnClient},
		stateMachineArn:       stateMachineArn,
	}

	lambda.Start(handler.HandleRequest)
}
