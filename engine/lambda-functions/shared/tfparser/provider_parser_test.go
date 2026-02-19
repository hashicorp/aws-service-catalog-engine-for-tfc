/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package tfparser

import (
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/exceptions"
	"reflect"
	"testing"
)

const ProviderFileContent1 = `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "example" {
  ami           = "ami-0742b4e673072066f"
  instance_type = "t3.micro"
}
`

const ProviderFileContent2 = `
terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

provider "random" {}
`

const ProviderFileContentWithAlias = `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

provider "aws" {
  alias  = "west"
  region = "us-west-2"
}
`

const ProviderOverrideFileContent = `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = "us-west-1"
}
`

const ProviderFileContentWithVariableRegion = `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

variable "aws_region" {
  type = string
  default = "us-east-1"
}
`

func TestParseProvidersFromConfigurationHappy(t *testing.T) {
	// setup
	fileMap := make(map[string]string)
	fileMap["main.tf"] = ProviderFileContent1
	fileMap["random.tf"] = ProviderFileContent2

	expectedResultMap := make(map[string]*Provider)

	expectedProvider1 := &Provider{
		Name:    "aws",
		Alias:   "",
		Version: "~> 5.0",
		Source:  "hashicorp/aws",
		Region:  "us-east-1",
	}

	expectedProvider2 := &Provider{
		Name:    "random",
		Alias:   "",
		Version: "~> 3.0",
		Source:  "hashicorp/random",
		Region:  "",
	}

	expectedResultMap["aws"] = expectedProvider1
	expectedResultMap["random"] = expectedProvider2

	// act
	actualResult, err := ParseProvidersFromConfiguration(fileMap)

	// assert
	if err != nil {
		t.Errorf("Unexpected error returned. %v", err)
	}

	// assert the number of providers parsed is as expected
	if len(actualResult) != len(expectedResultMap) {
		t.Errorf("The number of providers contained in the result is %v, not %v as expected", len(actualResult), len(expectedResultMap))
	}

	// assert the content of parsed providers is as expected
	for _, actualProvider := range actualResult {
		expectedProvider, ok := expectedResultMap[actualProvider.Name]

		if ok {
			if !reflect.DeepEqual(actualProvider, expectedProvider) {
				t.Errorf("Parsed provider with name %v is not the same as expected. Got: %+v, Expected: %+v", actualProvider.Name, actualProvider, expectedProvider)
			}
			delete(expectedResultMap, actualProvider.Name)
		} else {
			t.Errorf("Parsed provider with name %v is not expected", actualProvider.Name)
		}
	}

	// assert all providers were parsed
	if len(expectedResultMap) != 0 {
		t.Errorf("Not all expected providers were parsed. Remaining: %v", expectedResultMap)
	}
}

func TestParseProvidersFromConfigurationWithAliasHappy(t *testing.T) {
	// setup
	fileMap := make(map[string]string)
	fileMap["main.tf"] = ProviderFileContentWithAlias

	expectedResultMap := make(map[string]*Provider)

	expectedProvider1 := &Provider{
		Name:    "aws",
		Alias:   "",
		Version: "~> 5.0",
		Source:  "hashicorp/aws",
		Region:  "us-east-1",
	}

	expectedProvider2 := &Provider{
		Name:    "aws",
		Alias:   "west",
		Version: "~> 5.0",
		Source:  "hashicorp/aws",
		Region:  "us-west-2",
	}

	expectedResultMap["aws"] = expectedProvider1
	expectedResultMap["aws.west"] = expectedProvider2

	// act
	actualResult, err := ParseProvidersFromConfiguration(fileMap)

	// assert
	if err != nil {
		t.Errorf("Unexpected error returned. %v", err)
	}

	// assert the number of providers parsed is as expected
	if len(actualResult) != len(expectedResultMap) {
		t.Errorf("The number of providers contained in the result is %v, not %v as expected", len(actualResult), len(expectedResultMap))
	}

	// assert the content of parsed providers is as expected
	for _, actualProvider := range actualResult {
		key := actualProvider.Name
		if actualProvider.Alias != "" {
			key = actualProvider.Name + "." + actualProvider.Alias
		}

		expectedProvider, ok := expectedResultMap[key]

		if ok {
			if !reflect.DeepEqual(actualProvider, expectedProvider) {
				t.Errorf("Parsed provider with key %v is not the same as expected. Got: %+v, Expected: %+v", key, actualProvider, expectedProvider)
			}
			delete(expectedResultMap, key)
		} else {
			t.Errorf("Parsed provider with key %v is not expected", key)
		}
	}

	// assert all providers were parsed
	if len(expectedResultMap) != 0 {
		t.Errorf("Not all expected providers were parsed. Remaining: %v", expectedResultMap)
	}
}

func TestParseProvidersFromConfigurationWithOverrideFilesHappy(t *testing.T) {
	// setup
	fileMap := make(map[string]string)
	fileMap["main.tf"] = ProviderFileContent1
	fileMap["override.tf"] = ProviderOverrideFileContent

	expectedResultMap := make(map[string]*Provider)

	// Override file should override the version and region from main.tf
	expectedProvider1 := &Provider{
		Name:    "aws",
		Alias:   "",
		Version: "~> 4.0",
		Source:  "hashicorp/aws",
		Region:  "us-west-1",
	}

	expectedResultMap["aws"] = expectedProvider1

	// act
	actualResult, err := ParseProvidersFromConfiguration(fileMap)

	// assert
	if err != nil {
		t.Errorf("Unexpected error returned. %v", err)
	}

	// assert the number of providers parsed is as expected
	if len(actualResult) != len(expectedResultMap) {
		t.Errorf("The number of providers contained in the result is %v, not %v as expected", len(actualResult), len(expectedResultMap))
	}

	// assert the content of parsed providers is as expected
	for _, actualProvider := range actualResult {
		expectedProvider, ok := expectedResultMap[actualProvider.Name]

		if ok {
			if !reflect.DeepEqual(actualProvider, expectedProvider) {
				t.Errorf("Parsed provider with name %v is not the same as expected. Got: %+v, Expected: %+v", actualProvider.Name, actualProvider, expectedProvider)
			}
			delete(expectedResultMap, actualProvider.Name)
		} else {
			t.Errorf("Parsed provider with name %v is not expected", actualProvider.Name)
		}
	}

	// assert all providers were parsed
	if len(expectedResultMap) != 0 {
		t.Errorf("Not all expected providers were parsed. Remaining: %v", expectedResultMap)
	}
}

func TestParseProvidersFromConfigurationWithNoFilesThrowsParserInvalidParameterException(t *testing.T) {
	// setup
	fileMap := make(map[string]string)
	expectedErrorMessage := "No .tf files found. Nothing to parse. Make sure the root directory of the Terraform Cloud configuration file contains the .tf files for the root module."

	// act
	_, err := ParseProvidersFromConfiguration(fileMap)

	// assert
	if !reflect.DeepEqual(err, exceptions.ParserInvalidParameterException{Message: expectedErrorMessage}) {
		t.Errorf("Parser did not throw ParserInvalidParameterException with expected error message")
	}
}

func TestParseProvidersFromConfigurationWithVariableRegion(t *testing.T) {
	// setup
	fileMap := make(map[string]string)
	fileMap["main.tf"] = ProviderFileContentWithVariableRegion

	expectedResultMap := make(map[string]*Provider)

	expectedProvider := &Provider{
		Name:    "aws",
		Alias:   "",
		Version: "~> 5.0",
		Source:  "hashicorp/aws",
		Region:  "var.aws_region", // Should capture the variable reference
	}

	expectedResultMap["aws"] = expectedProvider

	// act
	actualResult, err := ParseProvidersFromConfiguration(fileMap)

	// assert
	if err != nil {
		t.Errorf("Unexpected error returned. %v", err)
	}

	// assert the number of providers parsed is as expected
	if len(actualResult) != len(expectedResultMap) {
		t.Errorf("The number of providers contained in the result is %v, not %v as expected", len(actualResult), len(expectedResultMap))
	}

	// assert the content of parsed providers is as expected
	for _, actualProvider := range actualResult {
		expectedProvider, ok := expectedResultMap[actualProvider.Name]

		if ok {
			if !reflect.DeepEqual(actualProvider, expectedProvider) {
				t.Errorf("Parsed provider with name %v is not the same as expected. Got: %+v, Expected: %+v", actualProvider.Name, actualProvider, expectedProvider)
			}
			delete(expectedResultMap, actualProvider.Name)
		} else {
			t.Errorf("Parsed provider with name %v is not expected", actualProvider.Name)
		}
	}

	// assert all providers were parsed
	if len(expectedResultMap) != 0 {
		t.Errorf("Not all expected providers were parsed. Remaining: %v", expectedResultMap)
	}
}

// Made with Bob
