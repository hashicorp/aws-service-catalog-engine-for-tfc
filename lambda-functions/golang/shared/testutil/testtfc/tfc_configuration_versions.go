/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package testtfc

import (
	"github.com/hashicorp/go-tfe"
	"fmt"
	"net/http"
	"strings"
	"log"
	"encoding/json"
	"io"
)

func (srv *MockTFC) AddConfigurationVersion(workspaceId string, configVersion *tfe.ConfigurationVersion) *tfe.ConfigurationVersion {
	configVersion.ID = ConfigVersionId(workspaceId)

	uploadUrl := fmt.Sprintf("%s/configuration-version-uploads/%s", srv.Address, configVersion.ID)
	configVersion.UploadURL = uploadUrl

	// Save the configuration version to the server's 'configuration versions by id" map
	srv.configurationVersionsById[configVersion.ID] = configVersion

	return configVersion
}

func (srv *MockTFC) HandleConfigurationVersionsPostRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/workspaces/ws-2jmj7l5rSw0yVb_v/configuration-versions => "", "api", "v2" "workspaces" "ws-2jmj7l5rSw0yVb_v" "configuration-versions"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if urlPathParts[3] == "workspaces" && urlPathParts[5] == "configuration-versions" {
		workspaceId := urlPathParts[4]

		var configVersion *tfe.ConfigurationVersion
		if err := json.NewDecoder(r.Body).Decode(&configVersion); err != nil {
			w.WriteHeader(500)
			return true
		}

		configVersion = srv.AddConfigurationVersion(workspaceId, configVersion)

		body, err := json.Marshal(MakeConfigurationVersionResponse(configVersion))
		if err != nil {
			w.WriteHeader(500)
			return true
		}
		w.WriteHeader(200)
		_, err = w.Write(body)
		if err != nil {
			log.Fatal(err)
			return true
		}
		return true
	}

	return false
}

func (srv *MockTFC) HandleConfigurationVersionsUploads(w http.ResponseWriter, r *http.Request) bool {
	// /configuration-version-uploads/cv-ZHZqfDii4BjHoGuL => "", "configuration-version-uploads", "cv-ZHZqfDii4BjHoGuL"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if urlPathParts[1] == "configuration-version-uploads" {
		configVersionId := urlPathParts[2]

		uploadedArtifact, err := io.ReadAll(r.Body)
		if err != nil {
			return false
		}
		srv.SetUploadedArtifact(uploadedArtifact)

		// set the configuration version's status to uploaded
		configVersion := srv.configurationVersionsById[configVersionId]
		configVersion.Status = tfe.ConfigurationUploaded
		srv.configurationVersionsById[configVersionId] = configVersion

		w.WriteHeader(200)
		return true
	}

	return false
}

func (srv *MockTFC) HandleConfigurationVersionsGetRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/configuration-versions/cv-WgF_1Sl_BfL3AOgT => "", "api", "v2" "configuration-versions" "cv-WgF_1Sl_BfL3AOgT"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if urlPathParts[3] == "configuration-versions" {
		configurationVersionId := urlPathParts[4]

		configurationVersion := srv.configurationVersionsById[configurationVersionId]

		body, err := json.Marshal(MakeConfigurationVersionResponse(configurationVersion))
		if err != nil {
			w.WriteHeader(500)
			return true
		}
		w.WriteHeader(200)
		_, err = w.Write(body)
		if err != nil {
			log.Fatal(err)
			return true
		}
		return true
	}

	return false
}

func MakeConfigurationVersionResponse(configVersion *tfe.ConfigurationVersion) map[string]interface{} {
	selfLink := fmt.Sprintf("/api/v2/configuration-versions/%s", configVersion.ID)

	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":   configVersion.ID,
			"type": "configuration-versions",
			"attributes": map[string]interface{}{
				"upload-url": configVersion.UploadURL,
				"status":     configVersion.Status,
			},
		},
		"relationships": map[string]interface{}{},
		"links": map[string]interface{}{
			"self": selfLink,
		},
	}
}
