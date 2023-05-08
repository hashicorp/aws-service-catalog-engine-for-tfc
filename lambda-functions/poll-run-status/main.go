package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/hashicorp/go-tfe"
	"log"
	"os"
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
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
		return PollRunStatusResponse{}, err
	}

	secretsManager := secretsmanager.NewFromConfig(sdkConfig)

	client, err := getTFEClient(ctx, secretsManager)
	if err != nil {
		log.Fatal(err)
		return PollRunStatusResponse{}, err
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

type TFECredentialsSecret struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token"`
}

func getTFEClient(ctx context.Context, secretsManagerClient *secretsmanager.Client) (*tfe.Client, error) {
	// Fetch the TFE credentials/config from AWS Secrets Manager
	secretId := os.Getenv("TFE_CREDENTIALS_SECRET_ID")
	versionId := os.Getenv("TFE_CREDENTIALS_SECRET_VERSION_ID")

	tfeCredentialsSecretJson, err := secretsManagerClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:  aws.String(secretId),
		VersionId: aws.String(versionId),
	})
	if err != nil {
		return nil, err
	}

	// Decode the response from AWS Secrets Manager
	var tfeCredentialsSecret TFECredentialsSecret
	if err = json.Unmarshal([]byte(*tfeCredentialsSecretJson.SecretString), &tfeCredentialsSecret); err != nil {
		return nil, err
	}

	// Use the credentials to create a TFE client
	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", tfeCredentialsSecret.Hostname),
		Token:   tfeCredentialsSecret.Token,
	})

	return client, err
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
		ProductProvisioningStatus: "failed",
		RunStatus:                 runStatus,
		ErrorMessage:              "Run requires approval in TFC. Approve the run in TFC, then update the product in Service Catalog to clear the error.",
	}
}

func success(runStatus tfe.RunStatus) PollRunStatusResponse {
	return PollRunStatusResponse{
		ProductProvisioningStatus: "success",
		RunStatus:                 runStatus,
		ErrorMessage:              "",
	}
}
