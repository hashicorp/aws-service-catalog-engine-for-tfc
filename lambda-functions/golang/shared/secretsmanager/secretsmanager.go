package secretsmanager

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type SecretsManager interface {
	GetSecretValue(ctx context.Context) (string, error)
	UpdateSecretValue(ctx context.Context, secretValue string) error
}

type SM struct {
	Client   *secretsmanager.Client
	SecretID string
	Hostname string
	TeamID   string
}

func (secretsManager *SM) GetSecretValue(ctx context.Context) (string, error) {
	// Get the secret version
	secret, err := secretsManager.Client.DescribeSecret(ctx)
	if err != nil {
		return "", err
	}

	secret.VersionIdsToStages
	// Get version ID
	// Fetch the ID via version ID fetch

	// Potential: Grab the currently tagged version
}

func (secretsManager *SM) UpdateSecretValue(ctx context.Context, secretValue string) error {
	// Update the secret version
	secret, err := secretsManager.Client.UpdateSecret(ctx, secretValue)
	if err != nil {
		return "", err
	}
}
