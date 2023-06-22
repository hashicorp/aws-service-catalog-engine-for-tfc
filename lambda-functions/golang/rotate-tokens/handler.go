package main

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/stepfunctions"
	"github.com/hashicorp/go-tfe"
	"log"
)

type RotateTeamTokensHandler struct {
	tfeClient                   *tfe.Client
	region                      string
	provisioningStateMachineArn string
	updatingStateMachineArn     string
	terminatingStateMachineArn  string
	provisioningFunctionName    string
	updatingFunctionName        string
	terminatingFunctionName     string
	teamID                      string
	stepFunctions               stepfunctions.StepFunctions
	lambdaClient                lambda.Client
	secretsManager              secretsmanager.SecretsManager
}

func (h *RotateTeamTokensHandler) HandleRequest(ctx context.Context, request RotateTeamTokensRequest) (*RotateTeamTokensResponse, error) {
	switch {
	case request.Operation == Pausing:
		eventSourceMappingsUuids, err := h.GetEventSourceMappingUuidTuples(ctx)
		if err != nil {
			log.Default().Printf("error getting event source mappings for function: %v", err)
			return nil, err
		}

		err = h.UpdateEventSourceMappings(ctx, eventSourceMappingsUuids, false)
		if err != nil {
			log.Default().Printf("error updating event source mappings for function: %v", err)
			return nil, err
		}
	case request.Operation == Polling:
		count, err := h.StateMachineExecutions(ctx)
		if err != nil {
			log.Default().Printf("error polling state machine executions: %v", err)
		}
		return &RotateTeamTokensResponse{StateMachineExecutionCount: count}, err
	case request.Operation == Rotating:
		err := h.RotateToken(ctx, teamID)
		if err != nil {
			log.Default().Printf("error polling state machine executions: %v", err)
		}
	case request.Operation == Erroring:
	default:
		log.Printf("Unknown serviceCatalogOperation: %s\n", request.Operation)
		return nil, errors.New("unknown operation")
	}
	// Get the team token via an ENV var via TF
	return &RotateTeamTokensResponse{}, nil
}
