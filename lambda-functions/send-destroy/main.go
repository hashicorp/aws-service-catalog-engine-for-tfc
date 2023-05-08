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

type SendDestroyRequest struct {
	AwsAccountId          string `json:"awsAccountId"`
	TerraformOrganization string `json:"terraformOrganization"`
	ProvisionedProductId  string `json:"provisionedProductId"`
}

type SendDestroyResponse struct {
	TerraformRunId string `json:"terraformRunId"`
}

func HandleRequest(ctx context.Context, request SendDestroyRequest) (*SendDestroyResponse, error) {
	client, err := getTFEClient(ctx)
	if err != nil {
		log.Printf("Failed to initialize TFE client: %s", err)
		return nil, err
	}

	workspaceId := getWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)

	// Get the workspace
	workspace, err := client.Workspaces.Read(ctx, request.TerraformOrganization, workspaceId)
	if err != nil {
		log.Printf("Workspace does not exist or couldn't be found: %s", err)
		return nil, err
	}

	// Queue "Terraform destroy"
	run, err := client.Runs.Create(ctx, tfe.RunCreateOptions{
		IsDestroy: tfe.Bool(true),
		Message:   tfe.String("Terminating product via AWS Service Catalog"),
		Workspace: workspace,
		AutoApply: tfe.Bool(true),
	})
	if err != nil {
		log.Printf("Failed to queue destroy run: %s", err)
		return nil, err
	}

	return &SendDestroyResponse{TerraformRunId: run.ID}, err
}

func main() {
	lambda.Start(HandleRequest)
}

type TFECredentialsSecret struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token"`
}

func getTFEClient(ctx context.Context) (*tfe.Client, error) {
	// create secrets client SDK to fetch tfe credentials
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	secretsManagerClient := secretsmanager.NewFromConfig(sdkConfig)

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

// Get the workspace name, which is `${accountId} - ${provisionedProductId}`
func getWorkspaceName(awsAccountId string, provisionedProductId string) string {
	return fmt.Sprintf("%s-%s", awsAccountId, provisionedProductId)
}
