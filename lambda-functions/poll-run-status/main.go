package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/go-tfe"
	"log"
)

type PollRunStatus struct {
	TerraformRunId string `json:"terraformRunId"`
}

type PollRunStatusResponse struct {
	ProductProvisioningStatus string        `json:"productProvisioningStatus"`
	RunStatus                 tfe.RunStatus `json:"runStatus"`
	ErrorMessage              string        `json:"errorMessage"`
}

func HandleRequest(ctx context.Context, request PollRunStatus) (PollRunStatusResponse, error) {
	client, err := tfe.NewClient(tfe.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	run, err := client.Runs.Read(ctx, request.TerraformRunId)
	if err != nil {
		log.Fatal(err)
	}

	runStatus := run.Status
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

func main() {
	lambda.Start(HandleRequest)
}

func failed(runStatus tfe.RunStatus, message string) PollRunStatusResponse {
	return PollRunStatusResponse{
		ProductProvisioningStatus: "failed",
		RunStatus:                 runStatus,
		ErrorMessage:              message,
	}
}

func inProgress(runStatus tfe.RunStatus) PollRunStatusResponse {
	return PollRunStatusResponse{
		ProductProvisioningStatus: "inProgress",
		RunStatus:                 runStatus,
		ErrorMessage:              "",
	}
}

func awaitingDecision(runStatus tfe.RunStatus) PollRunStatusResponse {
	return PollRunStatusResponse{
		ProductProvisioningStatus: "awaitingDecision",
		RunStatus:                 runStatus,
		ErrorMessage:              "",
	}
}

func success(runStatus tfe.RunStatus) PollRunStatusResponse {
	return PollRunStatusResponse{
		ProductProvisioningStatus: "success",
		RunStatus:                 runStatus,
		ErrorMessage:              "",
	}
}
