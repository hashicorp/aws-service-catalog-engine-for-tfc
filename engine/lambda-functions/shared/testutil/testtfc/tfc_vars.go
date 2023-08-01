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
	"strconv"
)

func (srv *MockTFC) AddVar(variable *tfe.Variable) *tfe.Variable {
	workspaceId := variable.Workspace.ID

	variable.ID = VarId(variable)

	// Get or create existing array of Variables for the workspace
	vars := make([]*tfe.Variable, 0)
	srv.requestLock.Lock()
	defer srv.requestLock.Unlock()
	if existingVars := srv.Vars[workspaceId]; existingVars != nil {
		vars = existingVars
	}

	srv.Vars[workspaceId] = append(vars, variable)

	return variable
}

func (srv *MockTFC) HandleVarsPostRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/workspaces/ws-2jmj7l5rSw0yVb_v/vars => "", "api", "v2" "workspaces" "ws-2jmj7l5rSw0yVb_v" "vars"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if urlPathParts[3] == "workspaces" && urlPathParts[5] == "vars" {
		workspaceId := urlPathParts[4]

		variable := &tfe.Variable{}

		reqVar := &VarUpdateOrCreateRequest{}
		if err := json.NewDecoder(r.Body).Decode(&reqVar); err != nil {
			w.WriteHeader(500)
			return true
		}

		copyRequestToVariable(variable, reqVar)
		workspace := &tfe.Workspace{ID: workspaceId}
		variable.Workspace = workspace

		variable = srv.AddVar(variable)

		body, err := json.Marshal(MakeVarResponse(variable))
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

func (srv *MockTFC) HandleVarsPatchRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/workspaces/ws-2jmj7l5rSw0yVb_v/vars/var-rOOv9Dd => "", "api", "v2" "workspaces" "ws-2jmj7l5rSw0yVb_v" "vars" "var-rOOv9Dd"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if len(urlPathParts) < 7 {
		return false
	}
	if urlPathParts[3] == "workspaces" && urlPathParts[5] == "vars" && urlPathParts[6] != "" {
		workspaceId := urlPathParts[4]
		workspaceVars := srv.Vars[workspaceId]

		varId := urlPathParts[6]
		var varToUpdate *tfe.Variable
		for _, workspaceVar := range workspaceVars {
			if workspaceVar.ID == varId {
				varToUpdate = workspaceVar
				break
			}
		}

		if varToUpdate == nil {
			w.WriteHeader(404)
			return true
		}

		reqVar := &VarUpdateOrCreateRequest{}
		if err := json.NewDecoder(r.Body).Decode(&reqVar); err != nil {
			w.WriteHeader(500)
			return true
		}

		copyRequestToVariable(varToUpdate, reqVar)

		body, err := json.Marshal(MakeVarResponse(varToUpdate))
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

func (srv *MockTFC) HandleVarsGetRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/workspaces/ws-2jmj7l5rSw0yVb_v/vars => "", "api", "v2" "workspaces" "ws-2jmj7l5rSw0yVb_v" "vars"
	urlPathParts := strings.Split(r.URL.Path, "/")

	srv.requestLock.Lock()
	defer srv.requestLock.Unlock()

	if urlPathParts[3] == "workspaces" && urlPathParts[5] == "vars" {
		workspaceId := urlPathParts[4]

		vars := srv.Vars[workspaceId]

		page, err := strconv.Atoi(r.URL.Query().Get("page[number]"))
		if err != nil {
			page = 0
		}
		size, err := strconv.Atoi(r.URL.Query().Get("page[size]"))
		if err != nil {
			size = 20
		}

		body, err := json.Marshal(MakeListVarsResponse(vars, page, size))
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

func (srv *MockTFC) HandleVarsDeleteRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/workspaces/ws-2jmj7l5rSw0yVb_v/vars/var-rOOv9Dd => "", "api", "v2" "workspaces" "ws-2jmj7l5rSw0yVb_v" "vars" "var-rOOv9Dd"
	urlPathParts := strings.Split(r.URL.Path, "/")

	srv.requestLock.Lock()
	defer srv.requestLock.Unlock()

	if urlPathParts[3] == "workspaces" && urlPathParts[5] == "vars" && urlPathParts[6] != "" {
		varId := urlPathParts[6]
		workspaceId := urlPathParts[4]
		workspaceVars := srv.Vars[workspaceId]

		found := false
		newWorkspaceVars := make([]*tfe.Variable, 0)
		for _, workspaceVar := range workspaceVars {
			if workspaceVar.ID == varId {
				found = true
			} else {
				newWorkspaceVars = append(newWorkspaceVars, workspaceVar)
			}
		}

		if found == false {
			w.WriteHeader(404)
			return true
		}

		srv.Vars[workspaceId] = newWorkspaceVars
		w.WriteHeader(204)
		return true
	}

	return false
}

func MakeListVarsResponse(vars []*tfe.Variable, page int, size int) map[string]interface{} {
	data := make([]map[string]interface{}, 0)

	startIndex := page * size
	endIndex := startIndex + size
	if endIndex > len(vars) {
		endIndex = len(vars)
	}
	paginatedData := vars[startIndex:endIndex]

	for _, variable := range paginatedData {
		selfLink := fmt.Sprintf("/api/v2/vars/%s", variable.ID)
		datum := map[string]interface{}{
			"id":   variable.ID,
			"type": "vars",
			"attributes": map[string]interface{}{
				"key":      variable.Key,
				"value":    variable.Value,
				"category": variable.Category,
				"hcl":      variable.HCL,
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
				"total-pages":  (len(vars) / size) + 1,
				"total-count":  len(vars),
			},
		},
	}
}

func MakeVarResponse(variable *tfe.Variable) map[string]interface{} {
	selfLink := fmt.Sprintf("/api/v2/variables/%s", variable.ID)

	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":   variable.ID,
			"type": "vars",
			"attributes": map[string]interface{}{
				"key":   variable.Key,
				"value": variable.Value,
				"hcl":   variable.HCL,
			},
		},
		"relationships": map[string]interface{}{
			"data": map[string]interface{}{
				"id":   variable.Workspace.ID,
				"type": "workspaces",
			},
		},
		"links": map[string]interface{}{
			"self": selfLink,
		},
	}
}

func copyRequestToVariable(variable *tfe.Variable, reqVar *VarUpdateOrCreateRequest) {
	variable.Key = reqVar.Data.Attributes.Key
	variable.Value = reqVar.Data.Attributes.Value
	variable.Category = reqVar.Data.Attributes.Category
	variable.HCL = reqVar.Data.Attributes.HCL
}

type VarUpdateOrCreateRequest struct {
	Data struct {
		Id         int `json:"id"`
		Attributes struct {
			Key      string           `json:"key"`
			Value    string           `json:"value"`
			Category tfe.CategoryType `json:"category"`
			HCL      bool             `json:"hcl"`
		} `json:"attributes"`
		Relationships struct {
			Workspace struct {
				Data struct {
					Id string `json:"id"`
				} `json:"data"`
			} `json:"workspace"`
		} `json:"relationships"`
	} `json:"data"`
}
