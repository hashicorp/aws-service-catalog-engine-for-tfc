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
	"time"
)

type RunFactoryParameters struct {
	RunStatus tfe.RunStatus
	Apply     *tfe.Apply
}

func (srv *MockTFC) AddRun(runId string, p RunFactoryParameters) *tfe.Run {
	// Create the mock run
	run := &tfe.Run{
		ID:     runId,
		Status: p.RunStatus,
		Apply:  p.Apply,
	}

	// Save the run to the mock server
	runPath := fmt.Sprintf("/api/v2/runs/%s", runId)
	srv.Runs[runPath] = run

	return run
}

func (srv *MockTFC) PersistRun(run *tfe.Run) *tfe.Run {
	runId := RunId(run)
	run.ID = runId

	// Save the run to the mock server
	runPath := fmt.Sprintf("/api/v2/runs/%s", runId)
	srv.Runs[runPath] = run

	return run
}

func (srv *MockTFC) HandleRunsPostRequests(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/api/v2/runs" {
		var runRequest *RunPostRequest
		if err := json.NewDecoder(r.Body).Decode(&runRequest); err != nil {
			w.WriteHeader(500)
			return true
		}

		run := RunFromRequest(*runRequest)
		run = srv.PersistRun(run)

		receipt := MakeGetRunResponse(*run)
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

func (srv *MockTFC) HandleRunsGetRequests(w http.ResponseWriter, r *http.Request) bool {
	run := srv.Runs[r.URL.Path]
	if run != nil {
		body, err := json.Marshal(MakeGetRunResponse(*run))
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

func MakeGetRunResponse(run tfe.Run) map[string]interface{} {
	selfLink := fmt.Sprintf("/api/v2/runs/%s", run.ID)

	relationships := map[string]interface{}{}

	if run.Apply != nil {
		relationships["apply"] = map[string]interface{}{
			"data": map[string]interface{}{
				"id":   run.Apply.ID,
				"type": "applies",
				"attributes": map[string]interface{}{
					"status": run.Apply.Status,
				},
			},
		}
	}

	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":   run.ID,
			"type": "runs",
			"attributes": map[string]interface{}{
				"status": run.Status,
			},
			"relationships": relationships,
			"links": map[string]interface{}{
				"self": selfLink,
			},
		},
	}
}

type RunPostRequest struct {
	Data struct {
		Id         int `json:"id"`
		Attributes struct {
			AutoApply bool `json:"auto-apply"`
			IsDestroy bool `json:"is-destroy"`
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

func RunFromRequest(req RunPostRequest) *tfe.Run {
	return &tfe.Run{
		AutoApply:              req.Data.Attributes.AutoApply,
		IsDestroy:              req.Data.Attributes.IsDestroy,
		CreatedAt:              time.Now(),
		ForceCancelAvailableAt: time.Now(),
		Workspace: &tfe.Workspace{
			ID: req.Data.Relationships.Workspace.Data.Id,
		},
	}
}
