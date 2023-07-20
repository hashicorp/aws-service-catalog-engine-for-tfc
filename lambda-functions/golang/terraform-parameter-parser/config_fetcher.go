package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/lambda-functions/golang/shared/fileutils"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/lambda-functions/golang/shared/exceptions"
)

const LaunchRoleAccessDeniedErrorMessage = "Access denied while assuming launch role %s: %s"
const ArtifactFetchAccessDeniedErrorMessage = "Access denied while downloading artifact from %s: %s"
const UnzipFailureErrorMessage = "Artifact from %s is not a valid tar.gz file: %s"

// Fetches the artifact file and returns it as a map of the entry names to their respective contents in string format
func (h *TerraformParameterParserHandler) fetchArtifact(ctx context.Context, request TerraformOpenSourceParameterParserInput) (map[string]string, error) {
	// Download the artifact from S3
	sourceProductConfig, err := fileutils.DownloadS3File(ctx, h.s3Downloader, request.LaunchRoleArn, request.Artifact.Path)
	if err != nil {
		return map[string]string{},
			exceptions.ParserAccessDeniedException{Message: fmt.Sprintf(ArtifactFetchAccessDeniedErrorMessage, request.Artifact.Path, err.Error())}
	}

	fileMap, err := UnzipArchive(sourceProductConfig)
	if err != nil {
		return fileMap,
			exceptions.ParserInvalidParameterException{Message: fmt.Sprintf(UnzipFailureErrorMessage, request.Artifact.Path, err.Error())}
	}

	return fileMap, nil
}
