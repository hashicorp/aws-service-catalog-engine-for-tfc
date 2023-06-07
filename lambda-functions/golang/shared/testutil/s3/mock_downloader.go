package s3

import (
	"os"
	"context"
)

type MockDownloader struct {
	MockArtifactPath string
}

func (downloader MockDownloader) Download(ctx context.Context, tmp *os.File, bucket string, objectKey string) (n int64, err error) {
	unzippedBytes, err := os.ReadFile(downloader.MockArtifactPath)

	write, err := tmp.Write(unzippedBytes)
	if err != nil {
		return 0, err
	}

	return int64(write), nil
}
