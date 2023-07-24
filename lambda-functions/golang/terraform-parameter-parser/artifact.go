/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

// Artifact represents the location of a Provisioning Artifact
type Artifact struct {
	Path string `json:"path"`
	Type string `json:"type"`
}
