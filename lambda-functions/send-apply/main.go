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
	"time"
)

type SendApplyRequest struct {
	AwsAccountId          string   `json:"awsAccountId"`
	TerraformOrganization string   `json:"terraformOrganization"`
	ProvisionedProductId  string   `json:"provisionedProductId"`
	Artifact              Artifact `json:"artifact"`
	LaunchRoleArn         string   `json:"launchRoleArn"`
}

type Artifact struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type SendApplyResponse struct {
	TerraformRunId string `json:"terraformRunId"`
}

func HandleRequest(ctx context.Context, request SendApplyRequest) (*SendApplyResponse, error) {
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	s3Client := s3.NewFromConfig(sdkConfig)
	secretsManager := secretsmanager.NewFromConfig(sdkConfig)

	client, err := getTFEClient(ctx, secretsManager)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	// Create the workspace
	w, err := client.Workspaces.Create(ctx, request.TerraformOrganization, tfe.WorkspaceCreateOptions{
		Name: tfe.String(getWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)),
	})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	// Configure ENV variables for OIDC
	_, err = client.Variables.Create(ctx, w.ID, tfe.VariableCreateOptions{
		Key:         tfe.String("TFC_AWS_PROVIDER_AUTH"),
		Value:       tfe.String("true"),
		Description: tfe.String("Enable the Workload Identity integration for AWS."),
		Category:    tfe.Category(tfe.CategoryEnv),
		HCL:         tfe.Bool(false),
		Sensitive:   tfe.Bool(false),
	})
	if err != nil {
		return nil, err
	}

	_, err = client.Variables.Create(ctx, w.ID, tfe.VariableCreateOptions{
		Key:         tfe.String("TFC_AWS_RUN_ROLE_ARN"),
		Value:       tfe.String(request.LaunchRoleArn),
		Description: tfe.String("The AWS role arn runs will use to authenticate."),
		Category:    tfe.Category(tfe.CategoryEnv),
		HCL:         tfe.Bool(false),
		Sensitive:   tfe.Bool(false),
	})
	if err != nil {
		return nil, err
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
		return nil, err
	}

	bucket, key := resolveArtifactPath(request.Artifact.Path)
	body, err := DownloadS3File(ctx, key, bucket, s3Client)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	err = client.ConfigurationVersions.UploadTarGzip(ctx, cv.UploadURL, body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	uploadTimeoutInSeconds := 120
	for i := 0; ; i++ {
		refreshed, err := client.ConfigurationVersions.Read(ctx, cv.ID)

		if refreshed.Status == tfe.ConfigurationUploaded {
			break
		}

		if i > uploadTimeoutInSeconds {
			log.Fatal(err)
			return nil, err
		}

		time.Sleep(1 * time.Second)
	}

	run, err := client.Runs.Create(ctx, tfe.RunCreateOptions{
		Workspace:            w,
		ConfigurationVersion: cv,
		AutoApply:            tfe.Bool(true),
	})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &SendApplyResponse{TerraformRunId: run.ID}, err
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

// Get the workspace name, which is `${accountId} - ${provisionedProductId}`
func getWorkspaceName(awsAccountId string, provisionedProductId string) string {
	return fmt.Sprintf("%s-%s", awsAccountId, provisionedProductId)
}
