/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package tfc

import (
	"fmt"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/exceptions"
)

// Error is used to wrap errors from TFC requests. This sets the type of the error so that users know where the error
// came from.
func Error(err error) error {
	if err == nil {
		return err
	}

	// Replace the "unauthorized" error with an error that provides the user with next steps to solve their issue
	if err.Error() == "unauthorized" {
		return exceptions.TFEUnauthorizedToken
	}

	return exceptions.TFEException{
		Message: fmt.Sprintf("request to Terraform Cloud failed. cause: %s", err.Error()),
	}
}
