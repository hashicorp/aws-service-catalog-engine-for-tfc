package stepfunctions

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type StepFunctions interface {
	StartExecution(ctx context.Context, input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error)
	GetStateMachineExecutionCount(ctx context.Context, stateMachineArn string) (int, error)
}

type SF struct {
	Client *sfn.Client
}

func (stepFunctions SF) StartExecution(ctx context.Context, input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error) {
	return stepFunctions.Client.StartExecution(ctx, input)
}

func (stepFunctions SF) GetStateMachineExecutionCount(ctx context.Context, stateMachineArn string) (int, error) {
	stateMachineExecutionsList, err := stepFunctions.Client.ListExecutions(ctx, &sfn.ListExecutionsInput{
		StateMachineArn: &stateMachineArn,
		StatusFilter:    types.ExecutionStatusRunning,
	})
	if err != nil {
		return 0, err
	}

	return len(stateMachineExecutionsList.Executions), nil
}
