package main

import (
	"testing"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/aws/smithy-go/middleware"
	"github.com/aws/aws-sdk-go-v2/aws"
	"time"
	"errors"
)

func TestProvisioningOperationsHandler_Success(t *testing.T) {

	// Create mock StepFunctions facade
	mockStepFunctions := &MockStepFunctionsWithSuccessfulResponse{}

	// Create a test instance of the Lambda function
	testHandler := &ProvisioningOperationsHandler{
		terraformOrganization: "yolo",
		stepFunctions:         mockStepFunctions,
		stateMachineArn:       "arn:::such-a-great-state-machine/like/wow",
	}

	// Create test request
	testPayload := StateMachinePayload{
		Token:                "tolkien",
		ProvisionedProductId: "the-bestest-product-id",
		RecordId:             "the-bestest-record-id",
	}
	testPayloadJson, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := ProvisioningOperationsHandlerRequest{
		Records: []Record{{
			MessageId: "the-bestest-msg-id",
			Body:      string(testPayloadJson),
		}},
	}

	// Send the test request
	response, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	assert.Empty(t, response.BatchItemFailures, "No failures should be returned")

	// Verify Terraform Organization was set
	assert.Equal(t, "yolo", mockStepFunctions.stateMachinePayload.TerraformOrganization, "terraformOrganization was set")
}

func TestProvisioningOperationsHandler_Failure(t *testing.T) {

	// Create mock StepFunctions facade
	mockStepFunctions := MockStepFunctionsWithErrorResponse{}

	// Create a test instance of the Lambda function
	testHandler := &ProvisioningOperationsHandler{
		stepFunctions: mockStepFunctions,
	}

	// Create test request
	testPayload := StateMachinePayload{
		Token:                "tolkien",
		ProvisionedProductId: "the-bestest-product-id",
		RecordId:             "the-bestest-record-id",
	}
	testPayloadJson, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := ProvisioningOperationsHandlerRequest{
		Records: []Record{{
			MessageId: "the-bestest-msg-id",
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
		ItemIdentifier: "the-bestest-msg-id",
	}}
	assert.Equal(t, expectedFailures, response.BatchItemFailures, "Expected a failure")
}

type MockStepFunctionsWithSuccessfulResponse struct {
	stateMachinePayload StateMachinePayload
}

func (stepFunctions *MockStepFunctionsWithSuccessfulResponse) StartExecution(ctx context.Context, input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error) {
	var stateMachinePayload StateMachinePayload
	if err := json.Unmarshal([]byte(*input.Input), &stateMachinePayload); err != nil {
		return nil, err
	}

	// Capture payload
	stepFunctions.stateMachinePayload = stateMachinePayload

	metadata := middleware.Metadata{}

	metadata.Set("RequestId", "the-bestest-request")

	return &sfn.StartExecutionOutput{
		ExecutionArn:   aws.String("arn:::mostly-successful"),
		StartDate:      aws.Time(time.Now()),
		ResultMetadata: metadata,
	}, nil
}

type MockStepFunctionsWithErrorResponse struct{}

func (stepFunctions MockStepFunctionsWithErrorResponse) StartExecution(ctx context.Context, input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error) {
	return nil, errors.New("whoopsies")
}
