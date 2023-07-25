/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/testtfc"
	"github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPollRunStatusHandler_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	// Create the TFE client that will send requests to the mock TFC instance
	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create a test instance of the Lambda function
	testHandler := &PollRunStatusHandler{
		secretsManager: mockSecretsManager,
	}

	t.Run("pending runs are evaluated as inProgress", func(t *testing.T) {
		// Add a mock Run to the mock TFC server
		tfcServer.AddRun("run-421337pending", testtfc.RunFactoryParameters{RunStatus: tfe.RunPending})

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
		tfcServer.AddRun("run-421337applied", testtfc.RunFactoryParameters{RunStatus: tfe.RunApplied})

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
		tfcServer.AddRun("run-421337canceled", testtfc.RunFactoryParameters{RunStatus: tfe.RunCanceled})

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
		tfcServer.AddRun("run-421337discarded", testtfc.RunFactoryParameters{RunStatus: tfe.RunDiscarded})

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
		tfcServer.AddRun("run-421337errored", testtfc.RunFactoryParameters{RunStatus: tfe.RunErrored})

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

func TestPollRunStatusHandler_InvalidTFCToken(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	// Create tfe client that will send requests to the mock TFC instance
	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create a test instance of the Lambda function
	testHandler := &PollRunStatusHandler{
		secretsManager: mockSecretsManager,
	}

	tfcServer.SetToken("a-different-secret")

	// Add a mock Run to the mock TFC server
	tfcServer.AddRun("run-everything-is-fine", testtfc.RunFactoryParameters{RunStatus: tfe.RunApplied})

	// Create a test request
	testRequest := PollRunStatus{
		TerraformRunId: "run-everything-is-fine",
	}

	// Send the test request to the test instance
	_, err := testHandler.HandleRequest(context.TODO(), testRequest)

	// Check the Lambda response
	assert.ErrorContains(t, err, "The current authorization token is not valid")
}

func TestPollRunStatusHandler_CannotConnect(t *testing.T) {
	t.Skip("Skipping test because it takes 9 minutes for the client to expend all of its retries")

	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()

	// Create tfe client that will send requests to the mock TFC instance
	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Add a mock Run to the mock TFC server
	tfcServer.AddRun("run-everything-is-fine", testtfc.RunFactoryParameters{RunStatus: tfe.RunApplied})

	// Stop mock TFC instance so that requests fail to reach it
	tfcServer.Stop()

	// Create a test instance of the Lambda function
	testHandler := &PollRunStatusHandler{
		secretsManager: mockSecretsManager,
	}

	// Create a test request
	testRequest := PollRunStatus{
		TerraformRunId: "run-everything-is-fine",
	}

	// Send the test request to the test instance
	_, err := testHandler.HandleRequest(context.TODO(), testRequest)

	// Check the Lambda response
	assert.ErrorContains(t, err, "failed to connect to Terraform Cloud servers")

}

func TestPollRunStatusHandler_RetriesFailures(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()

	// Create tfe client that will send requests to the mock TFC instance
	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create a test instance of the Lambda function
	testHandler := &PollRunStatusHandler{
		secretsManager: mockSecretsManager,
	}

	// Add a mock Run to the mock TFC server
	tfcServer.AddRun("run-everything-is-fine", testtfc.RunFactoryParameters{RunStatus: tfe.RunApplied})

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
