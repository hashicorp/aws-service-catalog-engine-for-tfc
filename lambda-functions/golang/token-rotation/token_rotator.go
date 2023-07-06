package main

import (
	"context"
	"log"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/token-rotation/lambda"
	"fmt"
)

func (h *RotateTeamTokensHandler) GetEventSourceMappingUuidTuples(ctx context.Context) ([]lambda.FunctionNameUuidTuple, error) {
	functionNames := []string{h.provisioningFunctionName, h.updatingFunctionName, h.terminatingFunctionName}

	return h.lambda.GetEventSourceMappingUuidTuples(ctx, functionNames)
}

func (h *RotateTeamTokensHandler) UpdateEventSourceMappings(ctx context.Context, tuples []lambda.FunctionNameUuidTuple, enabled bool) error {
	// Update the event source mappings asynchronously and restart the SQS queues.
	// The update is an asynchronous operation, so await its completion.
	for _, tuple := range tuples {
		var err error

		// Update the event mapping based on the "enabled" parameter
		if enabled {
			err = h.lambda.EnableEventSourceMapping(ctx, tuple.FunctionName, tuple.EventSourceMapping)
		} else {
			err = h.lambda.DisableEventSourceMapping(ctx, tuple.FunctionName, tuple.EventSourceMapping)
		}

		// return an error if one is encountered
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

	// Re-initialize the client with the new team token
	tfeCredentialsSecret, err := h.secretsManager.GetSecretValue(ctx)
	if err != nil {
		return err
	}
	tfeClient, err := tfc.ClientWithDefaultConfig(
		fmt.Sprintf("https://%s", tfeCredentialsSecret.Hostname),
		tt.Token,
		make(map[string][]string),
	)
	if err != nil {
		return err
	}
	h.tfeClient = tfeClient

	// Store team token in secrets manager
	return h.secretsManager.UpdateSecretValue(ctx, tt.Token)
}
