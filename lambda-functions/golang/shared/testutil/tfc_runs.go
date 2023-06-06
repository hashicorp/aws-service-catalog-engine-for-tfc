package testutil

import (
	"github.com/hashicorp/go-tfe"
	"fmt"
	"net/http"
	"encoding/json"
)

type RunFactoryParameters struct {
	RunStatus tfe.RunStatus
}

func (srv *MockTFC) AddRun(runId string, p RunFactoryParameters) {
	// Create the mock run
	run := &tfe.Run{
		ID:     runId,
		Status: p.RunStatus,
	}

	// Save the run to the mock server
	runPath := fmt.Sprintf("/api/v2/runs/%s", runId)
	srv.runs[runPath] = run
}

func (srv *MockTFC) HandleRunsRequests(w http.ResponseWriter, r *http.Request) bool {
	run := srv.runs[r.URL.Path]
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

	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":   run.ID,
			"type": "runs",
			"attributes": map[string]interface{}{
				"status": run.Status,
			},
		},
		"relationships": map[string]interface{}{},
		"links": map[string]interface{}{
			"self": selfLink,
		},
	}
}
