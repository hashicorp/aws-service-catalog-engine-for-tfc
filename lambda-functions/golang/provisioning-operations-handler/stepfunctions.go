package main

import (
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"context"
)

type StepFunctions interface {
	StartExecution(ctx context.Context, input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error)
}

type SF struct {
	Client *sfn.Client
}

func (stepFunctions SF) StartExecution(ctx context.Context, input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error) {
	return stepFunctions.Client.StartExecution(ctx, input)
}
