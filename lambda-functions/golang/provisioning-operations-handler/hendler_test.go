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
)

func TestProvisioningOperationsHandler_Success(t *testing.T) {

	// Create mock StepFunctions facade
	mockStepFunctions := MockStepFunctions{}

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

	assert.Empty(t, response.BatchItemFailures, "No failures should be returned")
}

type MockStepFunctions struct{}

func (stepFunctions MockStepFunctions) StartExecution(ctx context.Context, input *sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error) {
	metadata := middleware.Metadata{}

	metadata.Set("RequestId", "the-bestest-request")

	return &sfn.StartExecutionOutput{
		ExecutionArn:   aws.String("arn:::mostly-successful"),
		StartDate:      aws.Time(time.Now()),
		ResultMetadata: metadata,
	}, nil
}
