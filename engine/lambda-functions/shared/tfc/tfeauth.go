/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package tfc

import (
	"context"
	"fmt"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/secretsmanager"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/go-tfe"
	"net/http"
	"strings"
	"log"
)

type TFECredentialsSecret struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token"`
}

func GetTFEClient(ctx context.Context, secretsManager secretsmanager.SecretsManager) (*tfe.Client, error) {
	return GetTFEClientWithHeaders(ctx, secretsManager, http.Header{})
}

func GetTFEClientWithHeaders(ctx context.Context, secretsManager secretsmanager.SecretsManager, headers http.Header) (*tfe.Client, error) {
	// Fetch the TFE credentials/config from AWS Secrets Manager
	log.Default().Print("fetching TFC credentials from Secrets Manager")
	tfeCredentialsSecret, err := secretsManager.GetSecretValue(ctx)
	if err != nil {
		return nil, err
	}

	// Use the credentials to create a TFE client
	return GetTFEClientWithCredentials(tfeCredentialsSecret, headers)
}

func GetTFEClientWithCredentials(tfeCredentialsSecret *secretsmanager.TFECredentialsSecret, headers http.Header) (*tfe.Client, error) {
	if strings.HasPrefix(tfeCredentialsSecret.Hostname, "https:") || strings.HasPrefix(tfeCredentialsSecret.Hostname, "http:") {
		return ClientWithDefaultConfig(tfeCredentialsSecret.Hostname, tfeCredentialsSecret.Token, headers)
	}
	log.Default().Print("prepending protocol to TFC client hostname")
	return ClientWithDefaultConfig(fmt.Sprintf("https://%s", tfeCredentialsSecret.Hostname), tfeCredentialsSecret.Token, headers)
}

func ClientWithDefaultConfig(address string, token string, headers http.Header) (*tfe.Client, error) {
	log.Default().Printf("creating new TFC client for %s", address)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10

	return tfe.NewClient(&tfe.Config{
		Address:           fmt.Sprintf(address),
		Token:             token,
		RetryServerErrors: true,
		HTTPClient:        retryClient.HTTPClient,
		Headers:           headers,
	})
}
