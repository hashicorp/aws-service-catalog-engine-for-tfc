package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/hashicorp/go-tfe"
	"io"
	"log"
	"os"
	"strings"
)

type SendApplyRequest struct {
	TerraformOrganization string   `json:"terraformOrganization"`
	ProvisionedProductId  string   `json:"provisionedProductId"`
	Artifact              Artifact `json:"artifact"`
}

type Artifact struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type SendApplyResponse struct {
	TerraformRunId string `json:"terraformRunId"`
}

func HandleRequest(ctx context.Context, request SendApplyRequest) (SendApplyResponse, error) {
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
		return SendApplyResponse{}, err
	}

	s3Client := s3.NewFromConfig(sdkConfig)
	secretsManager := secretsmanager.NewFromConfig(sdkConfig)

	client, err := getTFEClient(ctx, secretsManager)
	if err != nil {
		log.Fatal(err)
		return SendApplyResponse{}, err
	}

	w, err := client.Workspaces.Create(ctx, request.TerraformOrganization, tfe.WorkspaceCreateOptions{
		Name: &request.ProvisionedProductId,
	})
	if err != nil {
		log.Fatal(err)
		return SendApplyResponse{}, err
	}

	cv, err := client.ConfigurationVersions.Create(ctx,
		w.ID,
		tfe.ConfigurationVersionCreateOptions{
			// Disable auto queue runs, so we can create the run ourselves to get the runId
			AutoQueueRuns: tfe.Bool(false),
		},
	)
	if err != nil {
		log.Fatal(err)
		return SendApplyResponse{}, err
	}

	bucket, key := resolveArtifactPath(request.Artifact.Path)
	body, err := DownloadS3File(ctx, key, bucket, s3Client)
	if err != nil {
		log.Fatal(err)
		return SendApplyResponse{}, err
	}

	err = client.ConfigurationVersions.UploadTarGzip(ctx, cv.UploadURL, body)
	if err != nil {
		log.Fatal(err)
		return SendApplyResponse{}, err
	}

	run, err := client.Runs.Create(ctx, tfe.RunCreateOptions{
		Workspace:            w,
		ConfigurationVersion: cv,
		AutoApply:            tfe.Bool(true),
	})
	if err != nil {
		log.Fatal(err)
		return SendApplyResponse{}, err
	}

	return SendApplyResponse{TerraformRunId: run.ID}, err
}

func main() {
	lambda.Start(HandleRequest)
}

type TFECredentialsSecret struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token"`
}

func getTFEClient(ctx context.Context, secretsManagerClient *secretsmanager.Client) (*tfe.Client, error) {
	// Fetch the TFE credentials/config from AWS Secrets Manager
	secretId := os.Getenv("TFE_CREDENTIALS_SECRET_ID")
	versionId := os.Getenv("TFE_CREDENTIALS_SECRET_VERSION_ID")

	tfeCredentialsSecretJson, err := secretsManagerClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:  aws.String(secretId),
		VersionId: aws.String(versionId),
	})
	if err != nil {
		return nil, err
	}

	// Decode the response from AWS Secrets Manager
	var tfeCredentialsSecret TFECredentialsSecret
	if err = json.Unmarshal([]byte(*tfeCredentialsSecretJson.SecretString), &tfeCredentialsSecret); err != nil {
		return nil, err
	}

	// Use the credentials to create a TFE client
	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", tfeCredentialsSecret.Hostname),
		Token:   tfeCredentialsSecret.Token,
	})

	return client, err
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
