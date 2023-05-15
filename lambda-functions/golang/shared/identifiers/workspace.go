package identifiers

import "fmt"

// GetWorkspaceName gets the workspace name, which is `${accountId} - ${provisionedProductId}`
func GetWorkspaceName(awsAccountId string, provisionedProductId string) string {
	return fmt.Sprintf("%s-%s", awsAccountId, provisionedProductId)
}
