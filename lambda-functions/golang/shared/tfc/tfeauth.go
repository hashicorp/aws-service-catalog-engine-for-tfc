package tfc

import (
	"context"
	"fmt"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/lambda-functions/golang/shared/secretsmanager"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/go-tfe"
	"net/http"
	"strings"
	"log"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/lambda-functions/golang/shared/exceptions"
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
	log.Default().Print("fetching TFC credentials from secretsmanager...")
	tfeCredentialsSecret, err := secretsManager.GetSecretValue(ctx)
	if err != nil {
		return nil, err
	}

	// Prepend protocol onto hostname if it does not yet have one specified
	hostname := tfeCredentialsSecret.Hostname
	if !(strings.HasPrefix(hostname, "https:") || strings.HasPrefix(hostname, "http:")) {
		hostname = fmt.Sprintf("https://%s", tfeCredentialsSecret.Hostname)
	}

	log.Default().Printf("creating TFE client with hostname: %s", hostname)
	return ClientWithDefaultConfig(hostname, tfeCredentialsSecret.Token, headers)
}

func ClientWithDefaultConfig(address string, token string, headers http.Header) (*tfe.Client, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	retryClient.ErrorHandler = ErrorHandler

	return tfe.NewClient(&tfe.Config{
		Address:           fmt.Sprintf(address),
		Token:             token,
		RetryServerErrors: true,
		HTTPClient:        retryClient.HTTPClient,
		Headers:           headers,
	})
}

func ErrorHandler(resp *http.Response, err error, _ int) (*http.Response, error) {
	// Replace the "unauthorized" error with an error that provides the user with next steps to solve their issue
	if err.Error() == "unauthorized" {
		return resp, exceptions.TFEUnauthorizedToken
	}

	return resp, exceptions.TFEException{
		Message: fmt.Sprintf("request to Terraform Cloud failed. cause: %s", err.Error()),
	}
}
