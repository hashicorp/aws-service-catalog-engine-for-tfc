package tfc

import (
	"fmt"
	"github.com/hashicorp/go-tfe"
	"github.com/aws/aws-sdk-go-v2/aws"
	"context"
	"github.com/hashicorp/go-retryablehttp"
	sm "github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/secretsmanager"
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
	secretsManager, err := sm.NewWithConfig(ctx, sdkConfig)
	if err != nil {
		return nil, err
	}

	// Fetch the TFE credentials/config from AWS Secrets Manager
	tfeCredentialsSecret, err := secretsManager.GetSecretValue(ctx)
	if err != nil {
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
