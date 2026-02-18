/*
 * Copyright IBM Corp. 2023, 2025
 * SPDX-License-Identifier: MPL-2.0
 */

package tfparser

// Parameter represents a single parsed variable from a Provisioning Artifact
type Parameter struct {
	Key          string `json:"key"`
	DefaultValue string `json:"defaultValue"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	IsNoEcho     bool   `json:"isNoEcho"`
}

// Provider represents a single parsed provider from a Provisioning Artifact
type Provider struct {
	Name    string `json:"name"`
	Alias   string `json:"alias"`
	Version string `json:"version"`
	Source  string `json:"source"`
	Region  string `json:"region"`
}

// Made with Bob
