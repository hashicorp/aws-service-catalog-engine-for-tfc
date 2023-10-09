/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package testtfc

import (
	"github.com/hashicorp/go-tfe"
	"fmt"
	"net/http"
	"encoding/json"
	"strings"
	"strconv"
	"time"
)

func (srv *MockTFC) AddStateVersion(applyId string, stateVersion *tfe.StateVersion) *tfe.StateVersion {
	stateVersion.ID = StateVersionId(applyId)

	// Save the StateVersion to the mock server
	srv.StateVersions[stateVersion.ID] = stateVersion

	// Index the StateVersions by Apply Id so they can be fetched later
	stateVersions := make([]*tfe.StateVersion, 0)
	if existingVars := srv.StateVersionsByApply[applyId]; existingVars != nil {
		stateVersions = existingVars
	}

	srv.StateVersionsByApply[applyId] = append(stateVersions, stateVersion)

	// Save the StateVersionOutputs, if they were set
	if stateVersion.Outputs != nil {
		srv.StateVersionOutputs[stateVersion.ID] = stateVersion.Outputs
	}

	return stateVersion
}

func (srv *MockTFC) HandleStateVersionsGetRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/workspaces/my-workspace/current-state-version => "", "api", "v2", "workspaces", "my-workspace", "current-state-version"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if urlPathParts[3] == "workspaces" && urlPathParts[5] == "current-state-version" {
		workspaceId := urlPathParts[4]

		stateVersion := srv.StateVersions[workspaceId]
		if stateVersion == nil {
			w.WriteHeader(404)
			return true
		}

		body, err := json.Marshal(MakeGetStateVersionResponse(*stateVersion))
		if err != nil {
			w.WriteHeader(500)
			return true
		}
		w.WriteHeader(200)
		w.Write(body)
		return true
	}

	// /api/v2/state-versions/sv-6DzZZJg0D_V0rcKz/outputs =>  "", "api", "v2", "state-versions", "sv-some-id", "outputs"
	if len(urlPathParts) > 5 && urlPathParts[3] == "state-versions" && urlPathParts[5] == "outputs" {
		stateVersionId := urlPathParts[4]

		stateVersionOutputs := srv.StateVersionOutputs[stateVersionId]
		if stateVersionOutputs == nil {
			w.WriteHeader(404)
			return true
		}

		page, err := strconv.Atoi(r.URL.Query().Get("page[number]"))
		if err != nil {
			page = 0
		}
		size, err := strconv.Atoi(r.URL.Query().Get("page[size]"))
		if err != nil {
			size = 20
		}
		body, err := json.Marshal(MakeGetStateVersionOutputsResponse(stateVersionOutputs, page, size))
		if err != nil {
			w.WriteHeader(500)
			return true
		}
		w.WriteHeader(200)
		w.Write(body)
		return true
	}

	// /api/v2/state-versions/sv-Ppz2mGCSVkJMCXXAe => "", "api", "v2", "state-versions", "sv-Ppz2mGCSVkJMCXXAe"
	if urlPathParts[3] == "state-versions" && len(urlPathParts) == 5 {
		stateVersionId := urlPathParts[4]

		stateVersion := srv.StateVersions[stateVersionId]
		if stateVersion == nil {
			w.WriteHeader(404)
			return true
		}

		body, err := json.Marshal(MakeGetStateVersionResponse(*stateVersion))
		if err != nil {
			w.WriteHeader(500)
			return true
		}
		w.WriteHeader(200)
		w.Write(body)
		return true
	}

	return false
}

func MakeGetStateVersionResponse(stateVersion tfe.StateVersion) map[string]interface{} {
	selfLink := fmt.Sprintf("/api/v2/state-versions/%s", stateVersion.ID)

	relationships := map[string]interface{}{}

	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":   stateVersion.ID,
			"type": "state-versions",
			"attributes": map[string]interface{}{
				"download-url": stateVersion.DownloadURL,
				"created-at":   stateVersion.CreatedAt.UTC().Format(time.RFC3339),
			},
			"relationships": relationships,
			"links": map[string]interface{}{
				"self": selfLink,
			},
		},
	}
}

func MakeGetStateVersionOutputsResponse(stateVersionOutputs []*tfe.StateVersionOutput, page int, size int) map[string]interface{} {
	data := make([]map[string]interface{}, 0)

	startIndex := page * size
	endIndex := startIndex + size
	if endIndex > len(stateVersionOutputs) {
		endIndex = len(stateVersionOutputs)
	}
	paginatedData := stateVersionOutputs[startIndex:endIndex]

	for _, output := range paginatedData {
		selfLink := fmt.Sprintf("/api/v2/state-version-outputs/%s", output.ID)
		datum := map[string]interface{}{
			"id":   output.ID,
			"type": "state-version-outputs",
			"attributes": map[string]interface{}{
				"name":      output.Name,
				"value":     output.Value,
				"sensitive": output.Sensitive,
			},
			"relationships": map[string]interface{}{},
			"links": map[string]interface{}{
				"self": selfLink,
			},
		}

		data = append(data, datum)
	}

	return map[string]interface{}{
		"data": data,
		"meta": map[string]interface{}{
			"pagination": map[string]interface{}{
				"current-page": page,
				"page-size":    size,
				"prev-page":    nil,
				"next-page":    nil,
				"total-pages":  (len(stateVersionOutputs) / size) + 1,
				"total-count":  len(stateVersionOutputs),
			},
		},
	}
}
