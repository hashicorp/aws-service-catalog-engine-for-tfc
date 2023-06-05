package tfc

import (
	"errors"
	"strings"
)

// MapErrorAfterRequest exists to map errors from requests to provide more helpful (and specific to this application)
// messages to aid in configuration and debugging
func MapErrorAfterRequest(err error) error {
	if err.Error() == "unauthorized" {
		// We know that the token was acquired because the client exists, so the token must have been acquired, but is
		// invalid or lacks permissions
		return errors.New("authorization token for TFC was acquired, but invalid or lacks sufficient permissions")
	}

	if strings.Contains(err.Error(), "connection refused") {
		// This should only happen if the Lambda is running a VPC or an AWS Outpost
		return errors.New("failed to connect to Terraform Cloud servers")
	}

	return err
}
