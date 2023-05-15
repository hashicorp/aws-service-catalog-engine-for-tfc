package tfeauth

import (
	"fmt"
	"github.com/hashicorp/go-tfe"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"os"
	"context"
	"encoding/json"
)

type TFECredentialsSecret struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token"`
}

func GetTFEClient(ctx context.Context, sdkConfig aws.Config) (*tfe.Client, error) {
	// Create secrets client SDK to fetch TFE credentials
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
