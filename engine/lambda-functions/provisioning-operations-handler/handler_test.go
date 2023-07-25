/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/stepfunction"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProvisioningOperationsHandler_Success(t *testing.T) {

	// Create mock StepFunctions facade
	mockStepFunctions := &stepfunction.MockStepFunctionsWithSuccessfulResponse{}

	// Create a test instance of the Lambda function
	testHandler := &ProvisioningOperationsHandler{
		terraformOrganization: "the-best-org",
		stepFunctions:         mockStepFunctions,
		stateMachineArn:       "arn:::such-a-great-state-machine/like/wow",
	}

	// Create test request
	testPayload := StateMachinePayload{
		Token:                "tolkien",
		ProvisionedProductId: "the-best-product-id",
		RecordId:             "the-best-record-id",
	}
	testPayloadJson, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := ProvisioningOperationsHandlerRequest{
		Records: []Record{{
			MessageId: "the-best-msg-id",
			Body:      string(testPayloadJson),
		}},
	}

	// Send the test request
	response, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	stateMachinePayload := &StateMachinePayload{}
	if err := json.Unmarshal([]byte(mockStepFunctions.StateMachinePayload), &stateMachinePayload); err != nil {
		t.Fatal(err)
	}

	assert.Empty(t, response.BatchItemFailures, "No failures should be returned")

	// Verify Terraform Organization was set
	assert.Equal(t, "the-best-org", stateMachinePayload.TerraformOrganization, "terraformOrganization was set")
}

func TestProvisioningOperationsHandler_Failure(t *testing.T) {

	// Create mock StepFunctions facade
	mockStepFunctions := &stepfunction.MockStepFunctionsWithErrorResponse{}

	// Create a test instance of the Lambda function
	testHandler := &ProvisioningOperationsHandler{
		stepFunctions: mockStepFunctions,
	}

	// Create test request
	testPayload := StateMachinePayload{
		Token:                "tolkien",
		ProvisionedProductId: "the-best-product-id",
		RecordId:             "the-best-record-id",
	}
	testPayloadJson, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := ProvisioningOperationsHandlerRequest{
		Records: []Record{{
			MessageId: "the-best-msg-id",
			Body:      string(testPayloadJson),
		}},
	}

	// Send the test request
	response, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	expectedFailures := []BatchItemFailure{{
		ItemIdentifier: "the-best-msg-id",
	}}
	assert.Equal(t, expectedFailures, response.BatchItemFailures, "Expected a failure")
}

func TestProvisioningOperationsHandler_StateMachinePayload(t *testing.T) {

	// Create mock StepFunctions facade
	mockStepFunctions := &stepfunction.MockStepFunctionsWithErrorResponse{}

	// Create a test instance of the Lambda function
	testHandler := &ProvisioningOperationsHandler{
		stepFunctions: mockStepFunctions,
	}

	// Create test request
	testPayload := StateMachinePayload{
		Token:                "tolkien",
		ProvisionedProductId: "the-best-product-id",
		RecordId:             "the-best-record-id",
	}
	testPayloadJson, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := ProvisioningOperationsHandlerRequest{
		Records: []Record{{
			MessageId: "the-best-msg-id",
			Body:      string(testPayloadJson),
		}},
	}

	// Send the test request
	response, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	expectedFailures := []BatchItemFailure{{
		ItemIdentifier: "the-best-msg-id",
	}}
	assert.Equal(t, expectedFailures, response.BatchItemFailures, "Expected a failure")
}
