/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/lambdafunction"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/stepfunction"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/testtfc"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Check for success during Team Token Rotation

func TestTokenRotationHandler_SuccessPausing(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock StepFunctions facade
	mockStepFunctions := &stepfunction.MockStepFunctionsWithSuccessfulResponse{}

	// Create mock Lambda function
	mockLambdaFunction := &lambdafunction.MockLambdaFunction{
		Provisioning: true,
		Updating:     true,
		Terminating:  true,
	}

	// Create a test instance of the Lambda function
	testHandler := RotateTeamTokensHandler{
		secretsManager: mockSecretsManager,
		stepFunctions:  mockStepFunctions,
		lambda:         mockLambdaFunction,
	}

	// Create test request
	testPayload := RotateTeamTokensRequest{
		Operation: Pausing,
	}
	_, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := testPayload

	// Send the test request
	response, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	assert.Empty(t, response, "No failures should be returned")

	// Verify that the Lambda event source mappings have been paused
	assert.Equal(t, false, mockLambdaFunction.Provisioning)
	assert.Equal(t, false, mockLambdaFunction.Updating)
	assert.Equal(t, false, mockLambdaFunction.Terminating)
}

func TestTokenRotationHandler_SuccessPolling(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock StepFunctions facade
	mockStepFunctions := &stepfunction.MockStepFunctionsWithSuccessfulResponse{}

	// Create mock Lambda function
	mockLambdaFunction := &lambdafunction.MockLambdaFunction{}

	// Create a test instance of the Lambda function
	testHandler := RotateTeamTokensHandler{
		secretsManager:              mockSecretsManager,
		stepFunctions:               mockStepFunctions,
		lambda:                      mockLambdaFunction,
		provisioningStateMachineArn: "arn:provision-thing-123",
		updatingStateMachineArn:     "arn:update-thing-123",
		terminatingStateMachineArn:  "arn:terminate-thing-123",
	}

	// Create test request
	testPayload := RotateTeamTokensRequest{
		Operation: Polling,
	}
	_, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := testPayload

	// Send the test request
	response, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	// Verify that the count is returned
	assert.Equal(t, 23, response.StateMachineExecutionCount)
}

func TestTokenRotationHandler_SuccessRotating(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-roLYatraNNailuJ2",
		Token:    "supers3cret",
	}

	// Create mock StepFunctions facade
	mockStepFunctions := &stepfunction.MockStepFunctionsWithSuccessfulResponse{}

	// Create mock Lambda function
	mockLambdaFunction := &lambdafunction.MockLambdaFunction{}

	// Create a test instance of the Lambda function
	testHandler := RotateTeamTokensHandler{
		secretsManager:              mockSecretsManager,
		stepFunctions:               mockStepFunctions,
		lambda:                      mockLambdaFunction,
		provisioningStateMachineArn: "arn:provision-thing-123",
		updatingStateMachineArn:     "arn:update-thing-123",
		terminatingStateMachineArn:  "arn:terminate-thing-123",
	}

	// Create test request
	testPayload := RotateTeamTokensRequest{
		Operation: Rotating,
	}
	_, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := testPayload

	// Send the test request
	_, err = testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	// Verify that the Team Token has been rotated
	assert.Equal(t, "newsupers3cret", mockSecretsManager.Token)
}

func TestTokenRotationHandler_SuccessResuming(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock StepFunctions facade
	mockStepFunctions := &stepfunction.MockStepFunctionsWithSuccessfulResponse{}

	// Create mock Lambda function
	mockLambdaFunction := &lambdafunction.MockLambdaFunction{
		Provisioning: false,
		Updating:     false,
		Terminating:  false,
	}

	// Create a test instance of the Lambda function
	testHandler := RotateTeamTokensHandler{
		secretsManager: mockSecretsManager,
		stepFunctions:  mockStepFunctions,
		lambda:         mockLambdaFunction,
	}

	// Create test request
	testPayload := RotateTeamTokensRequest{
		Operation: Resuming,
	}
	_, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := testPayload

	// Send the test request
	response, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	assert.Empty(t, response, "No failures should be returned")

	// Verify that the Lambda event source mappings have been resumed
	assert.Equal(t, true, mockLambdaFunction.Provisioning)
	assert.Equal(t, true, mockLambdaFunction.Updating)
	assert.Equal(t, true, mockLambdaFunction.Terminating)
}

// Check for errors during Team Token rotation

func TestTokenRotationHandler_ErrorPausing(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock StepFunctions facade
	mockStepFunctions := &stepfunction.MockStepFunctionsWithErrorResponse{}

	// Create mock Lambda function
	mockLambdaFunction := &lambdafunction.MockLambdaFunctionWithErrorResponse{
		Provisioning: false,
		Updating:     false,
		Terminating:  false,
	}

	// Create a test instance of the Lambda function
	testHandler := RotateTeamTokensHandler{
		secretsManager: mockSecretsManager,
		stepFunctions:  mockStepFunctions,
		lambda:         mockLambdaFunction,
	}

	// Create test request
	testPayload := RotateTeamTokensRequest{
		Operation: Pausing,
	}
	_, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := testPayload

	// Send the test request
	_, err = testHandler.HandleRequest(context.Background(), testRequest)
	// Verify the handler returned an error
	assert.Error(t, err, "function name or uuid not found")

	// Verify that the Lambda event source mappings have not been paused
	assert.Equal(t, false, mockLambdaFunction.Provisioning)
	assert.Equal(t, false, mockLambdaFunction.Updating)
	assert.Equal(t, false, mockLambdaFunction.Terminating)
}

func TestTokenRotationHandler_ErrorPolling(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock StepFunctions facade
	mockStepFunctions := &stepfunction.MockStepFunctionsWithErrorResponse{}

	// Create mock Lambda function
	mockLambdaFunction := &lambdafunction.MockLambdaFunctionWithErrorResponse{}

	// Create a test instance of the Lambda function
	testHandler := RotateTeamTokensHandler{
		secretsManager:              mockSecretsManager,
		stepFunctions:               mockStepFunctions,
		lambda:                      mockLambdaFunction,
		provisioningStateMachineArn: "",
		updatingStateMachineArn:     "",
		terminatingStateMachineArn:  "",
	}

	// Create test request
	testPayload := RotateTeamTokensRequest{
		Operation: Polling,
	}
	_, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := testPayload

	// Send the test request
	_, err = testHandler.HandleRequest(context.Background(), testRequest)
	// Verify the handler returned an error
	assert.Error(t, err, "invalid state machine arn")
}

func TestTokenRotationHandler_ErrorRotatingToken(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManagerWithoutUpdate{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock StepFunctions facade
	mockStepFunctions := &stepfunction.MockStepFunctionsWithErrorResponse{}

	// Create mock Lambda function
	mockLambdaFunction := &lambdafunction.MockLambdaFunctionWithErrorResponse{}

	// Create a test instance of the Lambda function
	testHandler := RotateTeamTokensHandler{
		secretsManager: mockSecretsManager,
		stepFunctions:  mockStepFunctions,
		lambda:         mockLambdaFunction,
	}

	// Create test request
	testRequest := RotateTeamTokensRequest{
		Operation: Rotating,
	}

	// Send the test request
	_, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify the handler returned an error
	assert.Error(t, err, "function name or uuid not found")
}

func TestTokenRotationHandler_ErrorResuming(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock StepFunctions facade
	mockStepFunctions := &stepfunction.MockStepFunctionsWithErrorResponse{}

	// Create mock Lambda function
	mockLambdaFunction := &lambdafunction.MockLambdaFunctionWithErrorResponse{
		Provisioning: true,
		Updating:     true,
		Terminating:  true,
	}

	// Create a test instance of the Lambda function
	testHandler := RotateTeamTokensHandler{
		secretsManager: mockSecretsManager,
		stepFunctions:  mockStepFunctions,
		lambda:         mockLambdaFunction,
	}

	// Create test request
	testPayload := RotateTeamTokensRequest{
		Operation: Resuming,
	}
	_, err := json.Marshal(testPayload)
	if err != nil {
		t.Error(err)
	}

	testRequest := testPayload

	// Send the test request
	_, err = testHandler.HandleRequest(context.Background(), testRequest)
	// Verify the handler returned an error
	assert.Error(t, err, "function name or uuid not found")

	// Verify that the Lambda event source mappings have not been resumed
	assert.Equal(t, true, mockLambdaFunction.Provisioning)
	assert.Equal(t, true, mockLambdaFunction.Updating)
	assert.Equal(t, true, mockLambdaFunction.Terminating)
}
