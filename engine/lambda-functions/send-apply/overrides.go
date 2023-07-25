/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"encoding/json"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/fileutils"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tracertag"
	"io"
	"log"
	"os"
)

type ConfigurationOverride struct {
	fileName     string
	fileContents string
}

func CreateAWSProviderOverrides(region string, tags []AWSTag, tracerTag tracertag.TracerTag) (*ConfigurationOverride, error) {
	// Format AWS billing tags
	formattedTags := map[string]interface{}{}
	for _, tag := range tags {
		formattedTags[tag.Key] = tag.Value
	}

	// Add tracer tag for resource tracking
	formattedTags[tracerTag.TracerTagKey] = tracerTag.TracerTagValue

	// The keys need to be strings, the values can be
	// any serializable value
	overrideData := map[string]any{
		"provider": map[string]interface{}{
			"aws": map[string]interface{}{
				"region": region,
				"default_tags": map[string]interface{}{
					"tags": formattedTags,
				},
			},
		},
	}

	// JSON encoding is done the same way as before
	data, err := json.Marshal(overrideData)
	if err != nil {
		return nil, err
	}

	log.Default().Printf("overriding aws provider with the following data: %s", string(data))

	return &ConfigurationOverride{
		fileName:     "provider_override.tf.json",
		fileContents: string(data),
	}, err
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
