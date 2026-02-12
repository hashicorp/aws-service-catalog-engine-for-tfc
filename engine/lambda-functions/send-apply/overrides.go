/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/fileutils"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tfparser"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tracertag"
)

type ConfigurationOverride struct {
	fileName     string
	fileContents string
}

func CreateAWSProviderOverrides(tarArchive *os.File, region string, tags []AWSTag, tracerTag tracertag.TracerTag) (*ConfigurationOverride, error) {
	// Format AWS billing tags
	formattedTags := map[string]interface{}{}
	for _, tag := range tags {
		formattedTags[tag.Key] = tag.Value
	}

	// Add tracer tag for resource tracking
	formattedTags[tracerTag.TracerTagKey] = tracerTag.TracerTagValue

	// Parse providers from the terraform configuration
	fileMap, err := tfparser.UnzipArchive(tarArchive)
	if err != nil {
		log.Default().Printf("failed to unzip archive for provider parsing: %v", err)
		// If we can't parse providers, fall back to just overriding the default provider
		fileMap = map[string]string{}
	}

	// Rewind the tar archive so it can be used again
	_, err = tarArchive.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	providers, err := tfparser.ParseProvidersFromConfiguration(fileMap)
	if err != nil {
		log.Default().Printf("failed to parse providers from configuration: %v", err)
		// If we can't parse providers, fall back to just overriding the default provider
		providers = []*tfparser.Provider{}
	}

	// Build provider overrides map
	providerOverrides := make(map[string]interface{})

	// Check if there are any AWS providers (default or aliased)
	hasAWSProviders := false
	for _, provider := range providers {
		if provider.Name == "aws" {
			hasAWSProviders = true
			break
		}
	}

	// If no AWS providers found, create override for default AWS provider
	if !hasAWSProviders {
		log.Default().Print("no AWS providers found in configuration, creating override for default provider")
		providerOverrides["aws"] = map[string]interface{}{
			"region": region,
			"default_tags": map[string]interface{}{
				"tags": formattedTags,
			},
		}
	} else {
		// Create overrides for each AWS provider found
		for _, provider := range providers {
			if provider.Name == "aws" {
				providerConfig := map[string]interface{}{
					"default_tags": map[string]interface{}{
						"tags": formattedTags,
					},
				}

				// Only override region if it wasn't defined in the configuration
				if provider.Region == "" {
					providerConfig["region"] = region
					if provider.Alias != "" {
						log.Default().Printf("creating override for aliased AWS provider: aws.%s (region not defined, using %s)", provider.Alias, region)
					} else {
						log.Default().Printf("creating override for default AWS provider (region not defined, using %s)", region)
					}
				} else {
					if provider.Alias != "" {
						log.Default().Printf("creating override for aliased AWS provider: aws.%s (keeping existing region: %s)", provider.Alias, provider.Region)
					} else {
						log.Default().Printf("creating override for default AWS provider (keeping existing region: %s)", provider.Region)
					}
				}

				// If provider has an alias, add it to the config
				if provider.Alias != "" {
					providerConfig["alias"] = provider.Alias
				}

				// Use "aws" for default provider, "aws.alias" for aliased providers
				providerKey := "aws"
				if provider.Alias != "" {
					providerKey = "aws." + provider.Alias
				}
				providerOverrides[providerKey] = providerConfig
			}
		}
	}

	// The keys need to be strings, the values can be
	// any serializable value
	overrideData := map[string]any{
		"provider": providerOverrides,
	}

	// JSON encoding is done the same way as before
	data, err := json.Marshal(overrideData)
	if err != nil {
		return nil, err
	}

	log.Default().Printf("overriding aws provider(s) with the following data: %s", string(data))

	return &ConfigurationOverride{
		fileName:     "provider_override.tf.json",
		fileContents: string(data),
	}, nil
}

func InjectOverrides(tarArchive *os.File, overrides []ConfigurationOverride) (*os.File, error) {
	log.Default().Print("injecting overrides into terraform configuration")

	// Unzip the file so that it can be appended to
	unzippedArchive, err := fileutils.UnzipFile(tarArchive)
	if err != nil {
		return nil, err
	}

	// Loop through the overrides and inject the tags
	for _, override := range overrides {
		err := fileutils.AddEntryToTar(unzippedArchive, override.fileName, override.fileContents)
		if err != nil {
			return nil, err
		}
	}

	// Rewind the tar file so that it can be read/uploaded in the future
	_, err = unzippedArchive.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	// Re-zip the file
	return fileutils.ZipFile(unzippedArchive)
}
