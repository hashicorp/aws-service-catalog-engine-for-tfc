package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"log"
)

type FunctionNameUuidTuple struct {
	FunctionName       string
	EventSourceMapping string
}

func (h *RotateTeamTokensHandler) GetEventSourceMappingUuidTuples(ctx context.Context) ([]FunctionNameUuidTuple, error) {
	var functionNameUuidTuples []FunctionNameUuidTuple
	functionNames := []string{h.provisioningFunctionName, h.updatingFunctionName, h.terminatingFunctionName}

	for _, functionName := range functionNames {
		log.Default().Printf("getting event source mappings for function %s", functionName)
		// Get the event source mapping UUIDs and disable the SQS queues
		eventSourceMappings, err := h.lambdaClient.ListEventSourceMappings(ctx, &lambda.ListEventSourceMappingsInput{
			FunctionName: aws.String(functionName),
		})

		if err != nil {
			return nil, err
		}

		for _, eventSourceMapping := range eventSourceMappings.EventSourceMappings {
			functionNameUuidTuples = append(functionNameUuidTuples, FunctionNameUuidTuple{
				FunctionName:       functionName,
				EventSourceMapping: *eventSourceMapping.UUID,
			})
		}
	}

	return functionNameUuidTuples, nil
}

// Need another way to call this outside of Lambda -- it's costly to call this within a lambda
//func AwaitEventSourceMappingState() {
//	// Wait for the event source mapping state
//
//}

func (h *RotateTeamTokensHandler) UpdateEventSourceMappings(ctx context.Context, tuples []FunctionNameUuidTuple, enabled bool) error {
	// Update the event source mappings asynchronously and restart the SQS queues.
	// The update is an asynchronous operation, so await its completion.
	for _, tuple := range tuples {
		log.Default().Printf("Updating enabled setting of event source mapping of %s:%s to %t", tuple.FunctionName, tuple.EventSourceMapping, enabled)

		_, err := h.lambdaClient.UpdateEventSourceMapping(ctx, &lambda.UpdateEventSourceMappingInput{
			FunctionName: aws.String(tuple.FunctionName),
			UUID:         aws.String(tuple.EventSourceMapping),
			Enabled:      aws.Bool(enabled),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *RotateTeamTokensHandler) StateMachineExecutions(ctx context.Context) (int, error) {
	// Return the number of executions for all state machines
	count := 0
	stateMachineArns := []string{h.provisioningStateMachineArn, h.updatingStateMachineArn, h.terminatingStateMachineArn}

	for _, stateMachineArn := range stateMachineArns {
		log.Default().Printf("getting state machine executions count for: %s", stateMachineArn)
		// Get the state machine executions count
		executionsCount, err := h.stepFunctions.GetStateMachineExecutionCount(ctx, stateMachineArn)
		if err != nil {
			return 0, err
		}
		// Return the count
		count = count + executionsCount
	}
	return count, nil
}

func (h *RotateTeamTokensHandler) RotateToken(ctx context.Context, teamID string) error {
	// Creates a new team token, replacing any existing token once all the state machine executions have finished
	tt, err := h.tfeClient.TeamTokens.Create(ctx, teamID)
	if err != nil {
		return err
	}

	// Reinitialize the client (???) with the new team token
	// TODO: Reinitialize the client (???) with the new team token

	// Store team token in secrets manager
	return h.secretsManager.UpdateSecretValue(ctx, tt.Token)

	// Return any error
	//return tt, err
}
