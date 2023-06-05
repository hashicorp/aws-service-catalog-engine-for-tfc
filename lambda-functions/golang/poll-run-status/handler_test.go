package main

import (
	"testing"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil"
	"github.com/hashicorp/go-tfe"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
)

func TestRunStatusHandler_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testutil.NewMockTFC()
	defer tfcServer.Stop()

	// Create tfe client that will send requests to the mock TFC instance
	tfeClient, err := tfc.ClientWithDefaultConfig(tfcServer.Address, "supers3cret")
	if err != nil {
		t.Error(err)
	}

	// Create a test instance of the Lambda function
	testHandler := &PollRunStatusHandler{
		tfeClient: tfeClient,
	}

	t.Run("pending runs are evaluated as inProgress", func(t *testing.T) {
		// Add a mock Run to the mock TFC server
		tfcServer.AddRun("run-421337pending", testutil.RunFactoryParameters{RunStatus: tfe.RunPending})

		// Create a test request
		testRequest := PollRunStatus{
			TerraformRunId: "run-421337pending",
		}

		// Send the test request to the test instance
		response, err := testHandler.HandleRequest(context.TODO(), testRequest)
		if err != nil {
			t.Error(err)
		}

		// Check the Lambda response
		assert.Equal(t, "inProgress", response.ProductProvisioningStatus, "product provisioning status should have been correctly evaluated")
		assert.Equal(t, tfe.RunPending, response.RunStatus, "correct run status should be returned")
		assert.Empty(t, response.ErrorMessage, "no error should be present in response")
	})

	t.Run("applied runs are evaluated as a success", func(t *testing.T) {
		// Add a mock Run to the mock TFC server
		tfcServer.AddRun("run-421337applied", testutil.RunFactoryParameters{RunStatus: tfe.RunApplied})

		// Create a test request
		testRequest := PollRunStatus{
			TerraformRunId: "run-421337applied",
		}

		// Send the test request to the test instance
		response, err := testHandler.HandleRequest(context.TODO(), testRequest)
		if err != nil {
			t.Error(err)
		}

		// Check the Lambda response
		assert.Equal(t, "success", response.ProductProvisioningStatus, "product provisioning status should have been correctly evaluated")
		assert.Equal(t, tfe.RunApplied, response.RunStatus, "correct run status should be returned")
		assert.Empty(t, response.ErrorMessage, "no error should be present in response")
	})

	t.Run("canceled runs are evaluated as failed", func(t *testing.T) {
		// Add a mock Run to the mock TFC server
		tfcServer.AddRun("run-421337canceled", testutil.RunFactoryParameters{RunStatus: tfe.RunCanceled})

		// Create a test request
		testRequest := PollRunStatus{
			TerraformRunId: "run-421337canceled",
		}

		// Send the test request to the test instance
		response, err := testHandler.HandleRequest(context.TODO(), testRequest)
		if err != nil {
			t.Error(err)
		}

		// Check the Lambda response
		assert.Equal(t, "failed", response.ProductProvisioningStatus, "product provisioning status should have been correctly evaluated")
		assert.Equal(t, tfe.RunCanceled, response.RunStatus, "correct run status should be returned")
		assert.Equal(t, "Run was cancelled", response.ErrorMessage, "error should be present in response")
	})

	t.Run("discarded runs are evaluated as failed", func(t *testing.T) {
		// Add a mock Run to the mock TFC server
		tfcServer.AddRun("run-421337discarded", testutil.RunFactoryParameters{RunStatus: tfe.RunDiscarded})

		// Create a test request
		testRequest := PollRunStatus{
			TerraformRunId: "run-421337discarded",
		}

		// Send the test request to the test instance
		response, err := testHandler.HandleRequest(context.TODO(), testRequest)
		if err != nil {
			t.Error(err)
		}

		// Check the Lambda response
		assert.Equal(t, "failed", response.ProductProvisioningStatus, "product provisioning status should have been correctly evaluated")
		assert.Equal(t, tfe.RunDiscarded, response.RunStatus, "correct run status should be returned")
		assert.Equal(t, "Run was discarded", response.ErrorMessage, "error should be present in response")
	})

	t.Run("errored runs are evaluated as failed", func(t *testing.T) {
		// Add a mock Run to the mock TFC server
		tfcServer.AddRun("run-421337errored", testutil.RunFactoryParameters{RunStatus: tfe.RunErrored})

		// Create a test request
		testRequest := PollRunStatus{
			TerraformRunId: "run-421337errored",
		}

		// Send the test request to the test instance
		response, err := testHandler.HandleRequest(context.TODO(), testRequest)
		if err != nil {
			t.Error(err)
		}

		// Check the Lambda response
		assert.Equal(t, "failed", response.ProductProvisioningStatus, "product provisioning status should have been correctly evaluated")
		assert.Equal(t, tfe.RunErrored, response.RunStatus, "correct run status should be returned")
		assert.Equal(t, "Failed running terraform apply", response.ErrorMessage, "error should be present in response")
	})
}

func TestRunStatusHandler_InvalidTFCToken(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testutil.NewMockTFC()
	defer tfcServer.Stop()

	// Create tfe client that will send requests to the mock TFC instance
	tfeClient, err := tfc.ClientWithDefaultConfig(tfcServer.Address, "supers3cret")
	if err != nil {
		t.Fatalf("failed to initialize TFE client: %v", err)
	}

	// Create a test instance of the Lambda function
	testHandler := &PollRunStatusHandler{
		tfeClient: tfeClient,
	}

	tfcServer.SetToken("a-different-secret")

	// Add a mock Run to the mock TFC server
	tfcServer.AddRun("run-everything-is-fine", testutil.RunFactoryParameters{RunStatus: tfe.RunApplied})

	// Create a test request
	testRequest := PollRunStatus{
		TerraformRunId: "run-everything-is-fine",
	}

	// Send the test request to the test instance
	_, err = testHandler.HandleRequest(context.TODO(), testRequest)

	// Check the Lambda response
	assert.ErrorContains(t, err, "authorization token for TFC was acquired, but invalid or lacks sufficient permissions")
}

func TestRunStatusHandler_CannotConnect(t *testing.T) {
	t.Skip("Skipping test because it takes 9 minutes for the client to expend all of its retries")

	// Create mock TFC instance
	tfcServer := testutil.NewMockTFC()

	// Create tfe client that will send requests to the mock TFC instance
	tfeClient, err := tfc.ClientWithDefaultConfig(tfcServer.Address, "supers3cret")
	if err != nil {
		t.Fatalf("failed to initialize TFE client: %v", err)
	}

	// Add a mock Run to the mock TFC server
	tfcServer.AddRun("run-everything-is-fine", testutil.RunFactoryParameters{RunStatus: tfe.RunApplied})

	// Stop mock TFC instance so that requests fail to reach it
	tfcServer.Stop()

	// Create a test instance of the Lambda function
	testHandler := &PollRunStatusHandler{
		tfeClient: tfeClient,
	}

	// Create a test request
	testRequest := PollRunStatus{
		TerraformRunId: "run-everything-is-fine",
	}

	// Send the test request to the test instance
	_, err = testHandler.HandleRequest(context.TODO(), testRequest)

	// Check the Lambda response
	assert.ErrorContains(t, err, "failed to connect to Terraform Cloud servers")

}

func TestRunStatusHandler_RetriesFailures(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testutil.NewMockTFC()

	// Create tfe client that will send requests to the mock TFC instance
	tfeClient, err := tfc.ClientWithDefaultConfig(tfcServer.Address, "supers3cret")
	if err != nil {
		t.Fatalf("failed to initialize TFE client: %v", err)
	}

	// Create a test instance of the Lambda function
	testHandler := &PollRunStatusHandler{
		tfeClient: tfeClient,
	}

	// Add a mock Run to the mock TFC server
	tfcServer.AddRun("run-everything-is-fine", testutil.RunFactoryParameters{RunStatus: tfe.RunApplied})

	// Create a test request
	testRequest := PollRunStatus{
		TerraformRunId: "run-everything-is-fine",
	}

	// Tell the mock to fail the next few requests from the Lambda
	tfcServer.FailRequests(2)

	// Send the test request to the test instance
	response, err := testHandler.HandleRequest(context.TODO(), testRequest)
	if err != nil {
		assert.Failf(t, "request returned an error %v", err.Error())
	}

	// Check the Lambda response
	assert.Equal(t, "success", response.ProductProvisioningStatus, "product provisioning status should have been correctly evaluated")
	assert.Equal(t, tfe.RunApplied, response.RunStatus, "correct run status should be returned")
	assert.Empty(t, response.ErrorMessage, "no error should be present in response")
}