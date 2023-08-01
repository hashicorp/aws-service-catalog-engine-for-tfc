/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package testtfc

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-tfe"
	"log"
	"net/http"
	"strings"
)

type WorkspaceFactoryParameters struct {
	Name string
}

func (srv *MockTFC) AddWorkspace(id string, p WorkspaceFactoryParameters) *tfe.Workspace {
	name := id
	if p.Name != "" {
		name = p.Name
	}

	// Create the mock workspace
	workspace := &tfe.Workspace{
		ID:   id,
		Name: name,
	}

	// Save the workspace to the mock server
	workspaceId := fmt.Sprintf(id)
	srv.Workspaces[workspaceId] = workspace

	return workspace
}

const ProductIdMetadataHeaderKey = "Tfp-Aws-Service-Catalog-Product-Id"
const ProvisionedProductIdMetadataHeaderKey = "Tfp-Aws-Service-Catalog-Prv-Product-Id"
const ProductVersionMetadataHeaderKey = "Tfp-Aws-Service-Catalog-Product-Ver"

func (srv *MockTFC) HandleWorkspacesPostRequests(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == fmt.Sprintf("/api/v2/organizations/%s/workspaces", srv.OrganizationName) {
		var workspace *tfe.Workspace
		if err := json.NewDecoder(r.Body).Decode(&workspace); err != nil {
			w.WriteHeader(500)
			return true
		}

		// Persist the workspace
		id := WorkspaceId(workspace)
		workspace = srv.AddWorkspace(id, WorkspaceFactoryParameters{Name: workspace.Name})

		// Persist metadata headers (if they were provided)
		metadata := &ServiceCatalogMetadata{
			ProductId:            r.Header.Get(ProductIdMetadataHeaderKey),
			ProvisionedProductId: r.Header.Get(ProvisionedProductIdMetadataHeaderKey),
			ProductVersion:       r.Header.Get(ProductVersionMetadataHeaderKey),
		}
		srv.WorkspaceServiceCatalogMetadata[id] = metadata

		receipt := MakeWorkspaceResponse(workspace)
		body, err := json.Marshal(receipt)
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

func (srv *MockTFC) HandleWorkspacesPatchRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/workspaces/ws-2jmj7l5rSw0yVb_v => "", "api", "v2" "workspaces" "ws-2jmj7l5rSw0yVb_v"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if urlPathParts[3] == "workspaces" && urlPathParts[4] != "" {
		workspaceId := urlPathParts[4]
		workspace := srv.Workspaces[workspaceId]

		if workspace == nil {
			w.WriteHeader(404)
			return true
		}

		reqWorkspace := &WorkspaceUpdateRequest{}
		if err := json.NewDecoder(r.Body).Decode(&reqWorkspace); err != nil {
			w.WriteHeader(500)
			return true
		}

		workspace.TerraformVersion = reqWorkspace.Data.Attributes.TerraformVersion

		body, err := json.Marshal(MakeWorkspaceResponse(workspace))
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

type WorkspaceUpdateRequest struct {
	Data struct {
		Id         int `json:"id"`
		Attributes struct {
			TerraformVersion string `json:"value"`
		} `json:"attributes"`
		Relationships struct{} `json:"relationships"`
	} `json:"data"`
}

func (srv *MockTFC) HandleWorkspacesGetRequests(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == fmt.Sprintf("/api/v2/organizations/%s/workspaces", srv.OrganizationName) {
		workspaces := make([]*tfe.Workspace, 0, len(srv.Workspaces))

		for _, value := range srv.Workspaces {
			workspaces = append(workspaces, value)
		}

		body, err := json.Marshal(MakeListWorkspacesResponse(workspaces))
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

	// /api/v2/organizations/team-rocket-blast-off/workspaces/123456789042-amazingly => "", "api", "v2", "organizations", "team-rocket-blast-off", "workspaces", "123456789042-amazingly"
	urlPathParts := strings.Split(r.URL.Path, "/")
	if urlPathParts[3] == "organizations" && urlPathParts[5] == "workspaces" && urlPathParts[6] != "" {
		workspaceId := urlPathParts[6]

		workspace := srv.Workspaces[workspaceId]

		if workspace == nil {
			w.WriteHeader(404)
			return true
		}

		body, err := json.Marshal(MakeWorkspaceResponse(workspace))
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

func (srv *MockTFC) HandleWorkspacesDeleteRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/organizations/team-rocket-blast-off/workspaces/123456789042-amazingly => "", "api", "v2", "organizations", "team-rocket-blast-off", "workspaces", "123456789042-amazingly"
	urlPathParts := strings.Split(r.URL.Path, "/")
	if urlPathParts[3] == "organizations" && urlPathParts[5] == "workspaces" && urlPathParts[6] != "" {
		workspaceId := urlPathParts[6]

		delete(srv.Workspaces, workspaceId)
		w.WriteHeader(204)
		return true
	}

	return false
}

func MakeListWorkspacesResponse(workspaces []*tfe.Workspace) map[string]interface{} {

	data := make([]map[string]interface{}, 0)

	for _, workspace := range workspaces {
		selfLink := fmt.Sprintf("/api/v2/workspaces/%s", workspace.ID)
		datum := map[string]interface{}{
			"id":   workspace.ID,
			"type": "workspaces",
			"attributes": map[string]interface{}{
				"name": workspace.Name,
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
	}
}

func MakeWorkspaceResponse(workspace *tfe.Workspace) map[string]interface{} {
	selfLink := fmt.Sprintf("/api/v2/workspaces/%s", workspace.ID)

	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":   workspace.ID,
			"type": "workspaces",
			"attributes": map[string]interface{}{
				"name": workspace.Name,
			},
		},
		"relationships": map[string]interface{}{},
		"links": map[string]interface{}{
			"self": selfLink,
		},
	}
}
