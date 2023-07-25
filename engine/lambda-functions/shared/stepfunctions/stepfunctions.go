/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package stepfunctions

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/aws/aws-sdk-go-v2/aws"
)

type StepFunctions interface {
	StartExecution(ctx context.Context, input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error)
	GetStateMachineExecutionCount(ctx context.Context, stateMachineArn string) (int, error)
}

type SFN struct {
	Client *sfn.Client
}

// NewFromConfig creates a new aws StepFunctions client
func NewFromConfig(sdkConfig aws.Config) *SFN {
	innerClient := sfn.NewFromConfig(sdkConfig)

	return &SFN{
		Client: innerClient,
	}
}

func (stepFunctions SFN) StartExecution(ctx context.Context, input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error) {
	return stepFunctions.Client.StartExecution(ctx, input)
}

func (stepFunctions SFN) GetStateMachineExecutionCount(ctx context.Context, stateMachineArn string) (int, error) {
	stateMachineExecutionsList, err := stepFunctions.Client.ListExecutions(ctx, &sfn.ListExecutionsInput{
		StateMachineArn: &stateMachineArn,
		StatusFilter:    types.ExecutionStatusRunning,
	})
	if err != nil {
		return 0, err
	}

	return len(stateMachineExecutionsList.Executions), nil
}
