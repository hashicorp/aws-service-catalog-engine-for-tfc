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
)

type ProjectFactoryParameters struct {
	Name string
}

func (srv *MockTFC) AddProject(id string, p ProjectFactoryParameters) {
	name := id
	if p.Name != "" {
		name = p.Name
	}

	// Create the mock Project
	project := &tfe.Project{
		ID:   id,
		Name: name,
	}

	// Save the Project to the mock server
	projectId := fmt.Sprintf(id)
	srv.Projects[projectId] = project
}

func (srv *MockTFC) HandleProjectsPostRequests(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == fmt.Sprintf("/api/v2/organizations/%s/projects", srv.OrganizationName) {
		var project *tfe.Project
		if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
			w.WriteHeader(500)
			return true
		}
		srv.AddProject(project.Name, ProjectFactoryParameters{Name: project.Name})

		receipt := MakeProjectResponse(project)
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

func (srv *MockTFC) HandleProjectsGetRequests(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == fmt.Sprintf("/api/v2/organizations/%s/projects", srv.OrganizationName) {
		projects := make([]*tfe.Project, 0, len(srv.Workspaces))

		for _, value := range srv.Projects {
			projects = append(projects, value)
		}

		body, err := json.Marshal(MakeListProjectsResponse(projects))
		if err != nil {
			w.WriteHeader(500)
			return true
		}

		w.WriteHeader(200)
		_, err = w.Write(body)
		if err != nil {
			w.WriteHeader(500)
			return true
		}

		return true
	}

	return false
}

func MakeListProjectsResponse(projects []*tfe.Project) map[string]interface{} {

	data := make([]map[string]interface{}, 0)

	for _, project := range projects {
		selfLink := fmt.Sprintf("/api/v2/projects/%s", project.ID)
		datum := map[string]interface{}{
			"id":   project.ID,
			"type": "projects",
			"attributes": map[string]interface{}{
				"name": project.Name,
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

func MakeProjectResponse(project *tfe.Project) map[string]interface{} {
	selfLink := fmt.Sprintf("/api/v2/projects/%s", project.ID)

	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":   project.ID,
			"type": "projects",
			"attributes": map[string]interface{}{
				"name": project.Name,
			},
		},
		"relationships": map[string]interface{}{},
		"links": map[string]interface{}{
			"self": selfLink,
		},
	}
}
