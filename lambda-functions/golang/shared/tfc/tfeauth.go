package tfc

import (
	"fmt"
	"github.com/hashicorp/go-tfe"
	"github.com/aws/aws-sdk-go-v2/aws"
	"os"
	"context"
	"encoding/json"
	"github.com/hashicorp/go-retryablehttp"
	sm "github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type TFECredentialsSecret struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token"`
}

func GetTFEClientFromSecretsManager(ctx context.Context, sm sm.SecretsManager) (*tfe.Client, error) {
	tfeCredentialsSecret, err := sm.GetSecretValue(ctx)
	if err != nil {
		return nil, err
	}

	// Use the credentials to create a TFE client
	return ClientWithDefaultConfig(fmt.Sprintf("https://%s", tfeCredentialsSecret.Hostname), tfeCredentialsSecret.Token)
}

func GetTFEClient(ctx context.Context, sdkConfig aws.Config) (*tfe.Client, error) {
	// Create secrets client SDK to fetch TFE credentials
	secretsManagerClient := secretsmanager.NewFromConfig(sdkConfig)

	// Fetch the TFE credentials/config from AWS Secrets Manager
	secretId := os.Getenv("TFE_CREDENTIALS_SECRET_ID")

	tfeCredentialsSecretJson, err := secretsManagerClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretId),
		VersionStage: aws.String("AWSCURRENT"),
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
	return ClientWithDefaultConfig(fmt.Sprintf("https://%s", tfeCredentialsSecret.Hostname), tfeCredentialsSecret.Token)
}

func ClientWithDefaultConfig(address string, token string) (*tfe.Client, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10

	return tfe.NewClient(&tfe.Config{
		Address:           fmt.Sprintf(address),
		Token:             token,
		RetryServerErrors: true,
		HTTPClient:        retryClient.HTTPClient,
	})
}
