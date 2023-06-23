package secretsmanager

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/aws"
	"encoding/json"
)

type SecretsManager interface {
	GetSecretValue(ctx context.Context) (*TFECredentialsSecret, error)
	UpdateSecretValue(ctx context.Context, secretValue string) error
}

type TFECredentialsSecret struct {
	Hostname string `json:"hostname"`
	TeamId   string `json:"id"`
	Token    string `json:"token"`
}

type SM struct {
	Client   *secretsmanager.Client
	SecretID string
	Hostname string
	TeamID   string
}

// CurrentVersionStage is AWS' hardcoded label that always indicates the "current" stage version
const CurrentVersionStage = "AWSCURRENT"

func (sm *SM) GetSecretValue(ctx context.Context) (*TFECredentialsSecret, error) {
	tfeCredentialsSecretJson, err := sm.Client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(sm.SecretID),
		VersionStage: aws.String(CurrentVersionStage),
	})
	if err != nil {
		return nil, err
	}

	// Decode the response from AWS Secrets Manager
	tfeCredentialsSecret := &TFECredentialsSecret{}
	err = json.Unmarshal([]byte(*tfeCredentialsSecretJson.SecretString), tfeCredentialsSecret)

	return tfeCredentialsSecret, err
}

func (sm *SM) UpdateTFEToken(ctx context.Context, token string) error {
	secretValue := &TFECredentialsSecret{
		Hostname: sm.Hostname,
		TeamId:   sm.TeamID,
		Token:    token,
	}

	// Serialize the new secret value
	serializedSecretValue, err := json.Marshal(secretValue)
	if err != nil {
		return err
	}

	// Update the secret version
	_, err = sm.Client.UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(sm.SecretID),
		SecretString: aws.String(string(serializedSecretValue)),
	})
	return err
}
