/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package secretsmanager

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/secretsmanager"
	"errors"
)

type MockSecretsManager struct {
	Hostname string
	TeamId   string
	Token    string
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

type MockSecretsManagerWithoutUpdate struct {
	Hostname string
	TeamId   string
	Token    string
}

func (msm *MockSecretsManagerWithoutUpdate) GetSecretValue(ctx context.Context) (*secretsmanager.TFECredentialsSecret, error) {
	return &secretsmanager.TFECredentialsSecret{
		Hostname: msm.Hostname,
		TeamId:   msm.TeamId,
		Token:    msm.Token,
	}, nil
}

func (msm *MockSecretsManagerWithoutUpdate) UpdateSecretValue(ctx context.Context, secretValue string) error {
	return errors.New("no update for you! ")
}
