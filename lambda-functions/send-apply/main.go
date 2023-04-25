package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/go-tfe"
	"io"
	"log"
	"strings"
)

type SendApplyRequest struct {
	TerraformOrganization string   `json:"terraformOrganization"`
	ProductId             string   `json:"productId"`
	Artifact              Artifact `json:"artifact"`
}

type Artifact struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

func HandleRequest(ctx context.Context, request SendApplyRequest) (string, error) {
	client, err := tfe.NewClient(tfe.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
		return "", err
	}

	s3Client := s3.NewFromConfig(sdkConfig)

	w, err := client.Workspaces.Create(ctx, request.TerraformOrganization, tfe.WorkspaceCreateOptions{
		Name: &request.ProductId,
	})

	if err != nil {
		log.Fatal(err)
	}

	cv, err := client.ConfigurationVersions.Create(ctx,
		w.ID,
		tfe.ConfigurationVersionCreateOptions{},
	)

	bucket, key := resolveArtifactPath(request.Artifact.Path)

	body, err := DownloadS3File(ctx, key, bucket, s3Client)

	if err != nil {
		log.Fatal(err)
	}

	err = client.ConfigurationVersions.UploadTarGzip(ctx, cv.UploadURL, body)

	return "", err
}

func main() {
	lambda.Start(HandleRequest)
	//HandleRequest(context.Background(), SendApplyRequest{ProductId: "SeaOtter", TerraformOrganization: "tf-rocket-tfcb-test", Artifact: Artifact{Path: "S3://sc-7b770157b2d833132fca6044b17db13b-us-west-2/out/7daa762ddeac5912c5541e569746ae55/75a85cfdb2998c07ad1f23fccd9aacca-98d65a1c224dc9625a4d87712a1175e5513cc5062570444d684991995fd3d6a5-bc4357186a4add0fea10da19ef3a31b9e45684f8f3d47b4c268aab73a428ee46-1682456166901-01ad0095-e396-4876-b544-6f51bc71d14a"}})
}

// Resolves artifactPath to bucket and key
func resolveArtifactPath(artifactPath string) (string, string) {
	bucket := strings.Split(artifactPath, "/")[2]
	key := strings.SplitN(artifactPath, "/", 4)[3]
	return bucket, key
}

func DownloadS3File(ctx context.Context, objectKey string, bucket string, s3Client *s3.Client) (io.Reader, error) {

	buffer := manager.NewWriteAtBuffer([]byte{})

	downloader := manager.NewDownloader(s3Client)

	numBytes, err := downloader.Download(ctx, buffer, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, err
	}

	if numBytes < 1 {
		return nil, errors.New("zero bytes written to memory")
	}

	return bytes.NewReader(buffer.Bytes()), nil
}
