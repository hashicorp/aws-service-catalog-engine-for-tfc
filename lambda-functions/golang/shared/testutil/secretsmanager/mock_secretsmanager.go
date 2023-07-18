package secretsmanager

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/lambda-functions/golang/shared/secretsmanager"
)

type MockSecretsManager struct {
	Hostname string
	TeamId   string
	Token    string
}

func CreateMockSecretsManager(hostname string, teamId string, token string) *MockSecretsManager {
	return &MockSecretsManager{
		Hostname: hostname,
		TeamId:   teamId,
		Token:    token,
	}
}

func (msm *MockSecretsManager) GetSecretValue(ctx context.Context) (*secretsmanager.TFECredentialsSecret, error) {
	return &secretsmanager.TFECredentialsSecret{
		Hostname: msm.Hostname,
		TeamId:   msm.TeamId,
		Token:    msm.Token,
	}, nil
}

func (msm *MockSecretsManager) UpdateSecretValue(ctx context.Context, secretValue string) error {
	msm.Token = secretValue
	return nil
}
