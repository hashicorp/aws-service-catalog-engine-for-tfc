package fileutils

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"os"
	"context"
)

type S3Downloader interface {
	Download(ctx context.Context, tmp *os.File, bucket string, key string) (n int64, err error)
}

type S3ManagerDownloader struct {
	S3Client *s3.Client
}

func (downloader S3ManagerDownloader) Download(ctx context.Context, tmp *os.File, bucket string, objectKey string) (n int64, err error) {
	downloadManager := manager.NewDownloader(downloader.S3Client)

	return downloadManager.Download(ctx, tmp, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})
}
