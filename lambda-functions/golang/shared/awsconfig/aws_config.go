/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package awsconfig

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"log"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
)

func GetSdkConfig(ctx context.Context) aws.Config {
	setRetryMode := func(configuration *config.LoadOptions) error {
		configuration.RetryMaxAttempts = 3
		configuration.RetryMode = aws.RetryModeStandard
		return nil
	}
	configuration, err := config.LoadDefaultConfig(ctx, setRetryMode)

	if err != nil {
		log.Fatal("failed to initialize AWS SDK Configuration")
	}

	return configuration
}

func GetSdkConfigWithRoleArn(ctx context.Context, initialConfig aws.Config, launchRoleArn string) (aws.Config, error) {
	// Create an STS client with the initial config
	stsClient := sts.NewFromConfig(initialConfig)

	// Create a new credential provider that will assume the IAM Role provided by the launchRoleArn parameter
	assumeRoleProvider := stscreds.NewAssumeRoleProvider(stsClient, launchRoleArn)

	// Create a new configuration with the assume role credential provider
	return config.LoadDefaultConfig(
		ctx,
		config.WithCredentialsProvider(assumeRoleProvider),
	)
}
