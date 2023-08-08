/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/stepfunctions"
	"log"
)

type ProvisioningOperationsHandler struct {
	terraformOrganization string
	stepFunctions         stepfunctions.StepFunctions
	stateMachineArn       string
}

func (h *ProvisioningOperationsHandler) HandleRequest(ctx context.Context, request ProvisioningOperationsHandlerRequest) (*ProvisioningOperationsHandlerResponse, error) {
	response := &ProvisioningOperationsHandlerResponse{}
	for _, record := range request.Records {
		err := h.StartStateMachineExecution(ctx, record)
		if err != nil {
			log.Default().Printf("Failed to start state machine execution for record, cause: %s", err.Error())

			// Add the failure to the dead letter queue
			response.BatchItemFailures = append(response.BatchItemFailures, BatchItemFailure{ItemIdentifier: record.MessageId})
		}
	}

	return response, nil
}

func (h *ProvisioningOperationsHandler) StartStateMachineExecution(ctx context.Context, record Record) error {
	log.Default().Printf("Deserializing event from SQS: %s", record.Body)

	stateMachinePayload := &StateMachinePayload{}
	if err := json.Unmarshal([]byte(record.Body), stateMachinePayload); err != nil {
		return err
	}

	stateMachinePayload.TerraformOrganization = h.terraformOrganization
	modifiedPayload, err := json.Marshal(stateMachinePayload)
	if err != nil {
		return err
	}

	executionName := fmt.Sprintf("%s-%s", stateMachinePayload.ProvisionedProductId, stateMachinePayload.RecordId)
	execution, err := h.stepFunctions.StartExecution(ctx, &sfn.StartExecutionInput{
		StateMachineArn: &h.stateMachineArn,
		Input:           aws.String(string(modifiedPayload)),
		Name:            &executionName,
	})
	if err != nil {
		return err
	}

	log.Default().Printf("Started state machine execution with arn: %s for request Id: %s", *execution.ExecutionArn, execution.ResultMetadata.Get("RequestId"))

	return nil
}
