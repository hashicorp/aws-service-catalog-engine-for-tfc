/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package fileutils

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/awsconfig"
	"os"
	"fmt"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/exceptions"
	"log"
)

type S3Downloader interface {
	Download(ctx context.Context, launchRoleArn string, tmp *os.File, bucket string, key string) (n int64, err error)
}

type S3ManagerDownloader struct {
	S3ClientProvider S3ClientProvider
}

type S3ClientProvider func(launchRoleArn string) (*s3.Client, error)

// NewS3DownloaderWithAssumedRole creates a new S3 Downloader that will assume the provided launch role to make requests to fetch files from S3
func NewS3DownloaderWithAssumedRole(ctx context.Context, sdkConfig aws.Config) S3ManagerDownloader {

	// Create the S3 Client with the provided IAM launch role
	return S3ManagerDownloader{
		S3ClientProvider: func(launchRoleArn string) (*s3.Client, error) {
			// use default lambda execution role creds to retrieve configuration templates if launch role is not provided
			if launchRoleArn == "" {
				log.Print("Launch role is not provided. Using ServiceCatalogTerraformCloudParameterParserRole to fetch artifact.")
				return s3.NewFromConfig(sdkConfig), nil
			}

			// Assume the provided IAM launch role
			log.Printf("Using launch role %s credentials to fetch artifact.", launchRoleArn)
			assumedRoleConfig, err := awsconfig.GetSdkConfigWithRoleArn(ctx, sdkConfig, launchRoleArn)
			if err != nil {
				return nil, exceptions.ParserAccessDeniedException{Message: fmt.Sprintf("Access denied while assuming launch role %s: %s", launchRoleArn, err.Error())}
			}
			return s3.NewFromConfig(assumedRoleConfig), nil
		},
	}
}

func (downloader S3ManagerDownloader) Download(ctx context.Context, launchRoleArn string, tmp *os.File, bucket string, objectKey string) (n int64, err error) {
	s3Client, err := downloader.S3ClientProvider(launchRoleArn)
	if err != nil {
		return 0, err
	}

	downloadManager := manager.NewDownloader(s3Client)

	return downloadManager.Download(ctx, tmp, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})
}
