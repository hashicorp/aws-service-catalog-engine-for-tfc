/*
 * Copyright IBM Corp. 2023, 2025
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

	// Count AWS providers (default and aliased separately)
	// Filter out providers that are only from required_providers (no actual configuration)
	// A provider with no alias and no region is likely just from required_providers
	var defaultProvider *tfparser.Provider
	aliasedProviders := []*tfparser.Provider{}
	hasDefaultProviderConfig := false
	
	for _, provider := range providers {
		if provider.Name == "aws" {
			// Check if this is a real provider configuration (has alias or region)
			if provider.Alias == "" && provider.Region == "" {
				// This is likely just from required_providers, not an actual provider block
				// Only count it if we don't find any other default provider
				if defaultProvider == nil {
					defaultProvider = provider
				}
			} else if provider.Alias == "" {
				// This is a real default provider configuration
				defaultProvider = provider
				hasDefaultProviderConfig = true
			} else {
				// This is an aliased provider
				aliasedProviders = append(aliasedProviders, provider)
			}
		}
	}
	
	// If we only have a default provider from required_providers (no real config), treat it as no provider
	if defaultProvider != nil && !hasDefaultProviderConfig && len(aliasedProviders) > 0 {
		defaultProvider = nil
	}

	// Determine if we should create a region override
	shouldOverrideRegion := false
	
	if defaultProvider == nil && len(aliasedProviders) == 0 {
		// No providers declared - create default provider with region
		log.Default().Print("no AWS providers found in configuration, creating override for default provider")
		shouldOverrideRegion = true
	} else if defaultProvider != nil && len(aliasedProviders) == 0 {
		// Only a single unaliased provider - override region if not set
		if defaultProvider.Region == "" {
			log.Default().Printf("creating override for default AWS provider (region not defined, using %s)", region)
			shouldOverrideRegion = true
		} else {
			log.Default().Printf("creating override for default AWS provider (keeping existing region: %s)", defaultProvider.Region)
		}
	} else if defaultProvider == nil && len(aliasedProviders) > 0 {
		// Only aliased provider(s) exist, no default provider - create default provider with region
		// This ensures there's a default provider available for resources that don't specify an alias
		log.Default().Printf("only aliased AWS provider(s) found, creating default provider with region %s", region)
		shouldOverrideRegion = true
	} else {
		// Multiple providers or mix of aliased and unaliased - don't override region
		log.Default().Print("multiple AWS providers found, skipping region override but adding tags")
	}

	// Create override for default provider only (never for aliased providers)
	providerConfig := map[string]interface{}{
		"default_tags": map[string]interface{}{
			"tags": formattedTags,
		},
	}
	
	if shouldOverrideRegion {
		providerConfig["region"] = region
	}
	
	providerOverrides["aws"] = providerConfig

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
