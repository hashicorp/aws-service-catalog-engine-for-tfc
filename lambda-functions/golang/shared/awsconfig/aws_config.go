package awsconfig

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"log"
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
