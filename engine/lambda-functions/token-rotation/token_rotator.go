/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tfc"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/token-rotation/lambda"
	"log"
	"net/http"
)

func (h *RotateTeamTokensHandler) UpdateEventSourceMappings(ctx context.Context, tuples *lambda.FunctionNameUuidTuples, enabled bool) error {
	// Update the event source mappings asynchronously and restart the SQS queues
	// The update is an asynchronous operation, so await its completion
	tuplesList := []*lambda.FunctionNameUuidTuple{tuples.ProvisioningLambdaEventSourceMapping, tuples.UpdatingLambdaEventSourceMapping, tuples.TerminatingLambdaEventSourceMapping}
	for _, tuple := range tuplesList {
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

func (h *RotateTeamTokensHandler) RotateToken(ctx context.Context) error {
	// Fetch the TFE credentials/config from AWS Secrets Manager
	tfeCredentialsSecret, err := h.secretsManager.GetSecretValue(ctx)
	if err != nil {
		return err
	}

	// Use the credentials to create a TFE client
	tfeClient, err := tfc.GetTFEClientWithCredentials(tfeCredentialsSecret, http.Header{})
	if err != nil {
		return err
	}

	// Creates a new Team Token, replacing any existing token, once all the state machine executions have finished
	tt, err := tfeClient.TeamTokens.Create(ctx, tfeCredentialsSecret.TeamId)
	if err != nil {
		return tfc.Error(err)
	}

	// Store the team token in Secrets Manager
	return h.secretsManager.UpdateSecretValue(ctx, tt.Token)
}
