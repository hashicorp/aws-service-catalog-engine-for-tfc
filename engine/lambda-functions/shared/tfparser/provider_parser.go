/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package tfparser

import (
	"log"
	"strings"

	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/exceptions"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/zclconf/go-cty/cty"
)

// ParseProvidersFromConfiguration - Takes Terraform configuration represented as a map from file name to string contents
// parses out the provider blocks and returns slice of Provider pointers
func ParseProvidersFromConfiguration(fileMap map[string]string) ([]*Provider, error) {
	if len(fileMap) == 0 {
		return nil, exceptions.ParserInvalidParameterException{
			Message: NoFilesToParseExceptionMessage,
		}
	}

	primaryFileMap, overrideFileMap := bisectFileMap(fileMap)

	primaryProviderMap := parseProviderMapFromFileMap(primaryFileMap, PrimaryModuleName)
	overrideProviderMap := parseProviderMapFromFileMap(overrideFileMap, OverrideModuleName)
	providers := mergeProviderMaps(primaryProviderMap, overrideProviderMap)

	return providers, nil
}

// parses provider map from provided file map and TF module
func parseProviderMapFromFileMap(fileMap map[string]string, moduleName string) map[string]*Provider {
	providerMap := make(map[string]*Provider)

	parser := hclparse.NewParser()
	mod := tfconfig.NewModule(moduleName)

	if fileMap == nil || len(fileMap) == 0 {
		return providerMap
	}

	// Parse HCL files and extract region information
	providerRegions := make(map[string]string) // key is "name" or "name.alias"

	for fileName, fileContents := range fileMap {
		log.Printf("Parsing file %s as HCL for providers", fileName)
		fileBytes := []byte(fileContents)
		file, _ := parser.ParseHCL(fileBytes, fileName)
		if file == nil {
			log.Panicf("Failed to parse file %s as HCL", fileName)
			continue
		}
		tfconfig.LoadModuleFromFile(file, mod)

		// Extract region from provider blocks in the HCL body
		extractRegionsFromHCL(file.Body, fileBytes, providerRegions)
	}

	// Parse required providers (from terraform.required_providers block)
	for name, providerReq := range mod.RequiredProviders {
		providerMap[name] = &Provider{
			Name:    name,
			Alias:   "",
			Version: strings.Join(providerReq.VersionConstraints, ", "),
			Source:  providerReq.Source,
			Region:  providerRegions[name],
		}
	}

	// Also parse provider configurations which may include aliases
	// The map key is already formatted as "name.alias" or just "name"
	for providerKey, providerConfig := range mod.ProviderConfigs {
		// Get region for this provider (with or without alias)
		region := providerRegions[providerKey]

		// If we already have this provider from RequiredProviders and it has no alias, update it with region
		if existingProvider, ok := providerMap[providerConfig.Name]; ok && providerConfig.Alias == "" {
			// Update the existing entry with region information
			existingProvider.Region = region
			continue
		}

		// Create new entry for aliased provider or provider not in required_providers
		version := ""
		source := ""
		// Try to get version and source from RequiredProviders if available
		if reqProvider, ok := mod.RequiredProviders[providerConfig.Name]; ok {
			version = strings.Join(reqProvider.VersionConstraints, ", ")
			source = reqProvider.Source
		}

		providerMap[providerKey] = &Provider{
			Name:    providerConfig.Name,
			Alias:   providerConfig.Alias,
			Version: version,
			Source:  source,
			Region:  region,
		}
	}

	return providerMap
}

// extractRegionsFromHCL extracts region attributes from provider blocks in HCL body
func extractRegionsFromHCL(body hcl.Body, fileBytes []byte, providerRegions map[string]string) {
	content, _, diags := body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       "provider",
				LabelNames: []string{"name"},
			},
		},
	})

	if diags.HasErrors() {
		return
	}

	for _, block := range content.Blocks {
		if block.Type == "provider" && len(block.Labels) > 0 {
			providerName := block.Labels[0]

			// Parse the block body to get attributes
			attrs, diags := block.Body.JustAttributes()
			if diags.HasErrors() {
				continue
			}

			// Look for alias attribute
			alias := ""
			if aliasAttr, ok := attrs["alias"]; ok {
				val, diags := aliasAttr.Expr.Value(nil)
				if !diags.HasErrors() && val.Type() == cty.String {
					alias = val.AsString()
				}
			}

			// Look for region attribute
			if regionAttr, ok := attrs["region"]; ok {
				var region string
				
				// Try to evaluate as a literal value first
				val, diags := regionAttr.Expr.Value(nil)
				if !diags.HasErrors() && val.Type() == cty.String {
					// It's a literal string, use the evaluated value (without quotes)
					region = val.AsString()
				} else {
					// It's a variable reference or other expression, extract raw text
					exprRange := regionAttr.Expr.Range()
					if exprRange.Start.Byte >= 0 && exprRange.End.Byte <= len(fileBytes) {
						region = string(fileBytes[exprRange.Start.Byte:exprRange.End.Byte])
					}
				}
				
				// Store the region if we successfully extracted it
				if region != "" {
					key := providerName
					if alias != "" {
						key = providerName + "." + alias
					}
					providerRegions[key] = region
				}
			}
		}
	}
}

// merges primary provider map and override provider map into a single list of providers
func mergeProviderMaps(primaryProviderMap map[string]*Provider, overrideProviderMap map[string]*Provider) []*Provider {
	var providers []*Provider

	if overrideProviderMap != nil && len(overrideProviderMap) != 0 {
		for key, overrideProvider := range overrideProviderMap {
			primaryProvider, ok := primaryProviderMap[key]
			if ok {
				mergedProvider := mergeProviders(primaryProvider, overrideProvider)
				providers = append(providers, mergedProvider)
			} else {
				providers = append(providers, overrideProvider)
			}
		}
	}

	if primaryProviderMap != nil && len(primaryProviderMap) != 0 {
		for key, primaryProvider := range primaryProviderMap {
			_, ok := overrideProviderMap[key]
			if !ok {
				providers = append(providers, primaryProvider)
			}
		}
	}

	return providers
}

// merges the primary provider with the override provider into a single provider
func mergeProviders(primaryProvider *Provider, overrideProvider *Provider) *Provider {
	mergedProvider := &Provider{}

	mergedProvider.Name = primaryProvider.Name

	if overrideProvider.Alias == "" {
		mergedProvider.Alias = primaryProvider.Alias
	} else {
		mergedProvider.Alias = overrideProvider.Alias
	}

	if overrideProvider.Version == "" {
		mergedProvider.Version = primaryProvider.Version
	} else {
		mergedProvider.Version = overrideProvider.Version
	}

	if overrideProvider.Source == "" {
		mergedProvider.Source = primaryProvider.Source
	} else {
		mergedProvider.Source = overrideProvider.Source
	}

	if overrideProvider.Region == "" {
		mergedProvider.Region = primaryProvider.Region
	} else {
		mergedProvider.Region = overrideProvider.Region
	}

	return mergedProvider
}

// Made with Bob
