package main

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/token-rotation/lambda"
	"log"
)

func (h *RotateTeamTokensHandler) GetEventSourceMappingUuidTuples(ctx context.Context) ([]lambda.FunctionNameUuidTuple, error) {
	functionNames := []string{h.provisioningFunctionName, h.updatingFunctionName, h.terminatingFunctionName}

	return h.lambda.GetEventSourceMappingUuidTuples(ctx, functionNames)
}

func (h *RotateTeamTokensHandler) UpdateEventSourceMappings(ctx context.Context, tuples []lambda.FunctionNameUuidTuple, enabled bool) error {
	// Update the event source mappings asynchronously and restart the SQS queues
	// The update is an asynchronous operation, so await its completion
	for _, tuple := range tuples {
		var err error

		// Update the event mapping based on the "enabled" parameter
		if enabled {
			err = h.lambda.EnableEventSourceMapping(ctx, tuple.FunctionName, tuple.EventSourceMapping)
		} else {
			err = h.lambda.DisableEventSourceMapping(ctx, tuple.FunctionName, tuple.EventSourceMapping)
		}

		// Return an error if one is encountered
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
	// Get TFE Client
	tfeClient, err := tfc.GetTFEClient(ctx, h.secretsManager)
	if err != nil {
		log.Fatalf("failed to initialize TFE client: %s", err)
	}

	// Creates a new team token, replacing any existing token once all the state machine executions have finished
	tt, err := tfeClient.TeamTokens.Create(ctx, teamID)
	if err != nil {
		return err
	}

	// Store the team token in Secrets Manager
	return h.secretsManager.UpdateSecretValue(ctx, tt.Token)
}
