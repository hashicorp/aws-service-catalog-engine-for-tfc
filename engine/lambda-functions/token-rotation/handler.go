/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"errors"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/stepfunctions"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/token-rotation/lambda"
	"log"
)

type RotateTeamTokensHandler struct {
	// AWS service clients
	stepFunctions  stepfunctions.StepFunctions
	lambda         lambda.Lambda
	secretsManager secretsmanager.SecretsManager
	// State machines to poll executions
	provisioningStateMachineArn string
	updatingStateMachineArn     string
	terminatingStateMachineArn  string
}

func (h *RotateTeamTokensHandler) HandleRequest(ctx context.Context, request RotateTeamTokensRequest) (*RotateTeamTokensResponse, error) {
	switch {
	case request.Operation == Pausing:
		eventSourceMappingsUuids, err := h.lambda.GetEventSourceMappingUuidTuples(ctx)
		if err != nil {
			log.Default().Printf("error getting event source mappings for function: %v", err)
			return nil, err
		}

		err = h.UpdateEventSourceMappings(ctx, eventSourceMappingsUuids, false)
		if err != nil {
			log.Default().Printf("error updating event source mappings for function: %v", err)
			return nil, err
		}
		return &RotateTeamTokensResponse{}, nil
	case request.Operation == Polling:
		count, err := h.StateMachineExecutions(ctx)
		if err != nil {
			log.Default().Printf("error polling state machine executions: %v", err)
			return nil, err
		}
		statuses, err := h.lambda.GetEventSourceMappingUuidTuples(ctx)
		if err != nil {
			return nil, err
		}

		aggregatedStatus := lambda.EventSourceDisabling
		if statuses.ProvisioningLambdaEventSourceMapping.EventSourceMappingStatus == lambda.EventSourceDisabled &&
			statuses.UpdatingLambdaEventSourceMapping.EventSourceMappingStatus == lambda.EventSourceDisabled &&
			statuses.TerminatingLambdaEventSourceMapping.EventSourceMappingStatus == lambda.EventSourceDisabled {
			aggregatedStatus = lambda.EventSourceDisabled
		}

		return &RotateTeamTokensResponse{StateMachineExecutionCount: count, EventSourceMappingStatus: aggregatedStatus}, err
	case request.Operation == Rotating:
		err := h.RotateToken(ctx)
		if err != nil {
			log.Default().Printf("error rotating team token: %v", err)
			return nil, err
		}
		return &RotateTeamTokensResponse{}, nil
	case request.Operation == Resuming:
		eventSourceMappingsUuids, err := h.lambda.GetEventSourceMappingUuidTuples(ctx)
		if err != nil {
			log.Default().Printf("error resuming event source mappings for function: %v", err)
			return &RotateTeamTokensResponse{}, err
		}

		err = h.UpdateEventSourceMappings(ctx, eventSourceMappingsUuids, true)
		if err != nil {
			log.Default().Printf("error updating event source mappings for function: %v", err)
			return nil, err
		}
		return &RotateTeamTokensResponse{}, nil
	default:
		log.Printf("Unknown serviceCatalogOperation: %s\n", request.Operation)
		return nil, errors.New("unknown operation")
	}
	// Get the team token via an ENV var via TF
	return &RotateTeamTokensResponse{}, errors.New("the lambda token rotation failed. this is due to a problem with the lambda code. please file an issue in the repository: https://github.com/hashicorp/aws-service-catalog-engine-for-tfc")
}
