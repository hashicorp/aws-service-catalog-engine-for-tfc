/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/fileutils"
)

type TerraformParameterParserHandler struct {
	s3Downloader fileutils.S3Downloader
}
