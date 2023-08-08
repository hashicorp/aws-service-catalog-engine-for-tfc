/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	lambdacore "github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/awsconfig"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/stepfunctions"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/token-rotation/lambda"
	"log"
	"os"
)

type RotateTeamTokensRequest struct {
	Operation Operation `json:"operation"`
}

type Operation string

// Enum values for Operation
const (
	Pausing  Operation = "PAUSING"
	Polling  Operation = "POLLING"
	Rotating Operation = "ROTATING"
	Resuming Operation = "RESUMING"
)

type RotateTeamTokensResponse struct {
	StateMachineExecutionCount int                             `json:"stateMachineExecutionCount"`
	EventSourceMappingStatus   lambda.EventSourceMappingStatus `json:"eventSourceMappingStatus"`
}

func main() {
	// Create temporary context to initialize the handler with
	initContext := context.TODO()

	// Initialize the TFE client
	sdkConfig := awsconfig.GetSdkConfig(initContext)

	// Create secrets client SDK to fetch TFE credentials
	secretsManager, err := secretsmanager.NewWithConfig(initContext, sdkConfig)
	if err != nil {
		log.Fatalf("failed to initialize secrets manager client: %s", err)
	}

	// Get provisioning state machine ARN
	provisioningStateMachineArn := os.Getenv("PROVISIONING_STATE_MACHINE_ARN")

	// Get updating state machine ARN
	updatingStateMachineArn := os.Getenv("UPDATING_STATE_MACHINE_ARN")

	// Get terminating state machine ARN
	terminatingStateMachineArn := os.Getenv("TERMINATING_STATE_MACHINE_ARN")

	handler := RotateTeamTokensHandler{
		secretsManager:              secretsManager,
		stepFunctions:               stepfunctions.NewFromConfig(sdkConfig),
		lambda:                      lambda.NewFromConfig(sdkConfig),
		provisioningStateMachineArn: provisioningStateMachineArn,
		updatingStateMachineArn:     updatingStateMachineArn,
		terminatingStateMachineArn:  terminatingStateMachineArn,
	}

	lambdacore.Start(handler.HandleRequest)
}
