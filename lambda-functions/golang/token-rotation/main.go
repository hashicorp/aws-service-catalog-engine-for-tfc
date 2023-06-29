package main

import (
	"context"
	lambdacore "github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/awsconfig"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/stepfunctions"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"log"
	"os"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/token-rotation/lambda"
)

type RotateTeamTokensRequest struct {
	Token       string    `json:"token"`
	TeamTokenID string    `json:"teamTokenID"`
	Operation   Operation `json:"operation"`
}

type Operation string

// Enum values for Operation
const (
	Pausing  Operation = "PAUSING"
	Polling  Operation = "POLLING"
	Rotating Operation = "ROTATING"
	Erroring Operation = "ERRORING"
)

type RotateTeamTokensResponse struct {
	StateMachineExecutionCount int `json:"stateMachineExecutionCount"`
}

func main() {
	// Create temporary context to initialize the handler with
	initContext := context.TODO()

	// Initialize the TFE client
	sdkConfig := awsconfig.GetSdkConfig(initContext)
	tfeClient, err := tfc.GetTFEClient(initContext, sdkConfig)
	if err != nil {
		log.Fatalf("failed to initialize TFE client: %s", err)
	}

	// Get provisioning state machine ARN
	provisioningStateMachineArn := os.Getenv("PROVISIONING_STATE_MACHINE_ARN")

	// Get updating state machine ARN
	updatingStateMachineArn := os.Getenv("UPDATING_STATE_MACHINE_ARN")

	// Get terminating state machine ARN
	terminatingStateMachineArn := os.Getenv("TERMINATING_STATE_MACHINE_ARN")

	// Get provisioning function name
	provisioningFunctionName := os.Getenv("PROVISIONING_FUNCTION_NAME")

	// Get updating function name
	updatingFunctionName := os.Getenv("UPDATING_FUNCTION_NAME")

	// Get terminating function name
	terminatingFunctionName := os.Getenv("TERMINATING_FUNCTION_NAME")

	handler := RotateTeamTokensHandler{
		tfeClient:                   tfeClient,
		stepFunctions:               stepfunctions.NewFromConfig(sdkConfig),
		lambda:                      lambda.NewFromConfig(sdkConfig),
		provisioningStateMachineArn: provisioningStateMachineArn,
		updatingStateMachineArn:     updatingStateMachineArn,
		terminatingStateMachineArn:  terminatingStateMachineArn,
		provisioningFunctionName:    provisioningFunctionName,
		updatingFunctionName:        updatingFunctionName,
		terminatingFunctionName:     terminatingFunctionName,
	}

	lambdacore.Start(handler.HandleRequest)
}
