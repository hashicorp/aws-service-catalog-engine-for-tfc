/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package s3

import (
	"os"
	"context"
	"errors"
)

type MockDownloader struct {
	MockArtifactPath string
	AssumedRole      string
}

func (downloader *MockDownloader) Download(ctx context.Context, launchRoleArn string, tmp *os.File, bucket string, objectKey string) (n int64, err error) {
	unzippedBytes, err := os.ReadFile(downloader.MockArtifactPath)

	downloader.AssumedRole = launchRoleArn

	write, err := tmp.Write(unzippedBytes)
	if err != nil {
		return 0, err
	}

	return int64(write), nil
}

type MockErrorDownloader struct {
}

func (downloader MockErrorDownloader) Download(ctx context.Context, launchRoleArn string, tmp *os.File, bucket string, objectKey string) (n int64, err error) {
	return 0, errors.New("whoopsies")
}
