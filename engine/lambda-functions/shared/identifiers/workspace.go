/*
 * Copyright IBM Corp. 2023, 2025
 * SPDX-License-Identifier: MPL-2.0
 */

package identifiers

import "fmt"

// GetWorkspaceName gets the workspace name, which is `${accountId} - ${provisionedProductId}`
func GetWorkspaceName(awsAccountId string, provisionedProductId string) string {
	return fmt.Sprintf("%s-%s", awsAccountId, provisionedProductId)
}
