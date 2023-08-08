/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tfc"
	"github.com/hashicorp/go-tfe"
	"log"
)

type PollRunStatusHandler struct {
	secretsManager secretsmanager.SecretsManager
}

type PollRunStatus struct {
	TerraformRunId string `json:"terraformRunId"`
}

type PollRunStatusResponse struct {
	ProductProvisioningStatus string        `json:"productProvisioningStatus"`
	RunStatus                 tfe.RunStatus `json:"runStatus"`
	ErrorMessage              string        `json:"errorMessage"`
}

func (h *PollRunStatusHandler) HandleRequest(ctx context.Context, request PollRunStatus) (*PollRunStatusResponse, error) {
	// Get TFE Client
	tfeClient, err := tfc.GetTFEClient(ctx, h.secretsManager)
	if err != nil {
		log.Printf("failed to initialize TFE client: %s", err)
		return nil, err
	}

	// Fetch the latest status of the run
	run, err := tfeClient.Runs.Read(ctx, request.TerraformRunId)
	if err != nil {
		return nil, tfc.Error(err)
	}

	// Respond with the appropriate status so the AWS Step Functions state machine will know what the next step is
	return RespondWithRunStatus(run.Status)
}

func RespondWithRunStatus(runStatus tfe.RunStatus) (*PollRunStatusResponse, error) {
	switch {
	case runStatus == tfe.RunApplied:
		return success(runStatus), nil
	case runStatus == tfe.RunCanceled:
		return failed(runStatus, "Run was cancelled"), nil
	case runStatus == tfe.RunDiscarded:
		return failed(runStatus, "Run was discarded"), nil
	case runStatus == tfe.RunErrored:
		return failed(runStatus, "Failed running terraform apply"), nil
	case runStatus == tfe.RunPlannedAndFinished:
		return success(runStatus), nil
	case runStatus == tfe.RunPostPlanAwaitingDecision:
		return awaitingDecision(runStatus), nil
	default:
		return inProgress(runStatus), nil
	}
}

func failed(runStatus tfe.RunStatus, message string) *PollRunStatusResponse {
	return &PollRunStatusResponse{
		ProductProvisioningStatus: "failed",
		RunStatus:                 runStatus,
		ErrorMessage:              message,
	}
}

func inProgress(runStatus tfe.RunStatus) *PollRunStatusResponse {
	return &PollRunStatusResponse{
		ProductProvisioningStatus: "inProgress",
		RunStatus:                 runStatus,
		ErrorMessage:              "",
	}
}

func awaitingDecision(runStatus tfe.RunStatus) *PollRunStatusResponse {
	return &PollRunStatusResponse{
		ProductProvisioningStatus: "failed",
		RunStatus:                 runStatus,
		ErrorMessage:              "Run requires approval in TFC. Approve the run in TFC, then update the example-product in Service Catalog to clear the error.",
	}
}

func success(runStatus tfe.RunStatus) *PollRunStatusResponse {
	return &PollRunStatusResponse{
		ProductProvisioningStatus: "success",
		RunStatus:                 runStatus,
		ErrorMessage:              "",
	}
}
