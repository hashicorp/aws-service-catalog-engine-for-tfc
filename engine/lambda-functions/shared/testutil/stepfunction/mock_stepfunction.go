/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package stepfunction

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/smithy-go/middleware"
	"time"
)

type MockStepFunctionsWithSuccessfulResponse struct {
	StateMachinePayload string
}

type MockStepFunctionsWithErrorResponse struct{}

func (stepFunctions *MockStepFunctionsWithSuccessfulResponse) StartExecution(ctx context.Context, input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error) {
	// Capture payload
	stepFunctions.StateMachinePayload = *input.Input

	metadata := middleware.Metadata{}

	metadata.Set("RequestId", "the-best-request")

	return &sfn.StartExecutionOutput{
		ExecutionArn:   aws.String("arn:::mostly-successful"),
		StartDate:      aws.Time(time.Now()),
		ResultMetadata: metadata,
	}, nil
}

func (stepFunctions *MockStepFunctionsWithSuccessfulResponse) GetStateMachineExecutionCount(ctx context.Context, stateMachineArn string) (int, error) {
	if stateMachineArn == "arn:provision-thing-123" {
		return 11, nil
	}

	if stateMachineArn == "arn:update-thing-123" {
		return 11, nil
	}

	if stateMachineArn == "arn:terminate-thing-123" {
		return 1, nil
	}

	return 0, errors.New("invalid state machine arn")
}

func (stepFunctions *MockStepFunctionsWithErrorResponse) StartExecution(ctx context.Context, input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error) {
	return nil, errors.New("whoopsies")
}

func (stepFunctions *MockStepFunctionsWithErrorResponse) GetStateMachineExecutionCount(ctx context.Context, stateMachineArn string) (int, error) {
	return 0, errors.New("wrong function called")
}
