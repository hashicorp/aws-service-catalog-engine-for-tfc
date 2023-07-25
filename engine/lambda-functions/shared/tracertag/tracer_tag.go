/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package tracertag

type TracerTag struct {
	TracerTagKey   string `json:"key"`
	TracerTagValue string `json:"value"`
}
