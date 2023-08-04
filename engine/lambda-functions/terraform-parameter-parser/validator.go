/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"fmt"
	"net/url"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/exceptions"
)

const ArtifactKey = "Artifact"
const LaunchRoleArnKey = "LaunchRoleArn"
const ArtifactPathKey = "Artifact.Path"
const ArtifactTypeKey = "Artifact.Type"

const DefaultArtifactType = "AWS_S3"
const IamArnServiceKey = "iam"
const S3Scheme = "s3"

const RequiredKeyMissingOrEmptyErrorMessage = "%s is required and must be non empty"
const InvalidLaunchRoleArnSyntaxErrorMessage = "LaunchRoleArn %s is not a syntactically valid ARN"
const InvalidIamLaunchRoleArnErrorMessage = "LaunchRoleArn %s is not a valid iam ARN"
const InvalidArtifactTypeErrorMessage = "Artifact type %s is not supported, must be AWS_S3"
const InvalidArtifactPathErrorMessage = "Artifact path %s is not a valid S3 URI"

// ValidateInput - Validates TerraformParameterParserInput
// Returns a non nil error if an invalid input is provided
func ValidateInput(input TerraformParameterParserInput) error {
	// validate required keys exist in the input
	if err := validateRequiredKeysExist(input); err != nil {
		return err
	}

	// validate the format of LaunchRoleArn (if it was provided)
	if input.LaunchRoleArn != "" {
		if err := validateLaunchRoleArnIsSyntacticallyCorrect(input.LaunchRoleArn); err != nil {
			return err
		}
	}

	// validate the Artifact
	if err := validateArtifact(input.Artifact); err != nil {
		return err
	}

	return nil
}

func validateRequiredKeysExist(input TerraformParameterParserInput) error {
	if reflect.DeepEqual(input.Artifact, Artifact{}) {
		return exceptions.ParserInvalidParameterException{
			Message: fmt.Sprintf(RequiredKeyMissingOrEmptyErrorMessage, ArtifactKey),
		}
	}

	if input.Artifact.Path == "" {
		return exceptions.ParserInvalidParameterException{
			Message: fmt.Sprintf(RequiredKeyMissingOrEmptyErrorMessage, ArtifactPathKey),
		}
	}

	if input.Artifact.Type == "" {
		return exceptions.ParserInvalidParameterException{
			Message: fmt.Sprintf(RequiredKeyMissingOrEmptyErrorMessage, ArtifactTypeKey),
		}
	}

	return nil
}

func validateLaunchRoleArnIsSyntacticallyCorrect(launchRoleArnString string) error {
	launchRoleArn, err := arn.Parse(launchRoleArnString)
	if err != nil {
		return exceptions.ParserInvalidParameterException{
			Message: fmt.Sprintf(InvalidLaunchRoleArnSyntaxErrorMessage, launchRoleArnString),
		}
	}

	if launchRoleArn.Service != IamArnServiceKey {
		return exceptions.ParserInvalidParameterException{
			Message: fmt.Sprintf(InvalidIamLaunchRoleArnErrorMessage, launchRoleArnString),
		}
	}

	return nil
}

func validateArtifact(artifact Artifact) error {
	if artifact.Type != DefaultArtifactType {
		return exceptions.ParserInvalidParameterException{
			Message: fmt.Sprintf(InvalidArtifactTypeErrorMessage, artifact.Type),
		}
	}

	artifactUri, err := url.Parse(artifact.Path)
	if err != nil {
		return exceptions.ParserInvalidParameterException{
			Message: fmt.Sprintf(InvalidArtifactPathErrorMessage, artifact.Path),
		}
	}

	if artifactUri.Scheme != S3Scheme || artifactUri.Host == "" || artifactUri.Path == "" {
		return exceptions.ParserInvalidParameterException{
			Message: fmt.Sprintf(InvalidArtifactPathErrorMessage, artifact.Path),
		}
	}

	return nil
}
