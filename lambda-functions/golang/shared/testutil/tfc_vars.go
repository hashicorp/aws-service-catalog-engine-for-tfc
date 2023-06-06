package testutil

import (
	"github.com/hashicorp/go-tfe"
	"fmt"
	"net/http"
	"strings"
	"log"
	"encoding/json"
)

func (srv *MockTFC) AddVar(variable *tfe.Variable) *tfe.Variable {
	workspaceId := variable.Workspace.ID

	variable.ID = VarId(variable)

	// Get or create existing array of Variables for the workspace
	vars := make([]*tfe.Variable, 0)
	if existingVars := srv.vars[workspaceId]; existingVars != nil {
		vars = existingVars
	}

	srv.vars[workspaceId] = append(vars, variable)

	return variable
}

func (srv *MockTFC) HandleVarsPostRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/workspaces/ws-2jmj7l5rSw0yVb_v/vars => "", "api", "v2" "workspaces" "ws-2jmj7l5rSw0yVb_v" "vars"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if urlPathParts[3] == "workspaces" && urlPathParts[5] == "vars" {
		workspaceId := urlPathParts[4]

		var variable *tfe.Variable
		if err := json.NewDecoder(r.Body).Decode(&variable); err != nil {
			w.WriteHeader(500)
			return true
		}

		variable.Workspace = &tfe.Workspace{ID: workspaceId}

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

func (srv *MockTFC) HandleVarsGetRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/workspaces/ws-2jmj7l5rSw0yVb_v/vars => "", "api", "v2" "workspaces" "ws-2jmj7l5rSw0yVb_v" "vars"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if urlPathParts[3] == "workspaces" && urlPathParts[5] == "vars" {
		workspaceId := urlPathParts[4]

		vars := srv.vars[workspaceId]

		body, err := json.Marshal(MakeListVarsResponse(vars))
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

func MakeListVarsResponse(vars []*tfe.Variable) map[string]interface{} {

	data := make([]map[string]interface{}, 0)

	for _, variable := range vars {
		selfLink := fmt.Sprintf("/api/v2/vars/%s", variable.ID)
		datum := map[string]interface{}{
			"id":   variable.ID,
			"type": "vars",
			"attributes": map[string]interface{}{
				"key":   variable.Key,
				"value": variable.Value,
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

func MakeVarResponse(variable *tfe.Variable) map[string]interface{} {
	selfLink := fmt.Sprintf("/api/v2/variables/%s", variable.ID)

	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":   variable.ID,
			"type": "vars",
			"attributes": map[string]interface{}{
				"key":   variable.Key,
				"value": variable.Value,
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
