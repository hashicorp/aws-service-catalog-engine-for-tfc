/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"encoding/json"
	"log"
)

type Error struct {
	ErrorMessage string `json:"errorMessage"`
	ErrorType    string `json:"errorType"`
}

// SimplifyError simplifies errors that come in the form of {"errorMessage":"unauthorized","errorType":"errorString"}, to simply be "unauthorized"
func SimplifyError(errorString string) string {
	errorParsed := &Error{}
	err := json.Unmarshal([]byte(errorString), errorParsed)
	if err != nil {
		if errorString == "Failed running terraform apply" {
			// If the error was due to the Terraform failing to apply in TFC, give the user helpful next steps
			return "Failed to run terraform apply in Terraform Cloud. Check the workspace in Terraform Cloud for details"
		}

		log.Default().Printf("error message was not valid json, returning as is (without formatting): %s", errorString)

		// If the error failed to be parsed, return it as is, without any additional formatting, as it was likely parsed previously
		return errorString
	}

	log.Default().Printf("simplifying error with type of %s and message: %s", errorParsed.ErrorType, errorParsed.ErrorMessage)

	if errorParsed.ErrorType == "errorString" && errorParsed.ErrorMessage != "" {
		// If the errorType is errorString, simplify the error by de-jsonifying it
		return errorParsed.ErrorMessage
	}

	if errorParsed.ErrorType == "TFEUnauthorized" && errorParsed.ErrorMessage != "" {
		// If the errorType is TFEUnauthorized, simplify the error message
		return errorParsed.ErrorMessage
	}

	// To be sure that we don't hide any useful error information, use the original errorString in order to guarantee that all information was passed along
	return errorString
}
