package testutil

import (
	"github.com/hashicorp/go-tfe"
	"fmt"
)

type RunFactoryParameters struct {
	RunStatus tfe.RunStatus
}

type MockRunGetResponse struct {
}

func (srv *MockTFC) AddRun(runId string, p RunFactoryParameters) {
	// create the mock run
	run := &tfe.Run{
		ID:     runId,
		Status: p.RunStatus,
	}

	// save the run to the mock server
	runPath := fmt.Sprintf("/api/v2/runs/%s", runId)
	srv.runs[runPath] = run
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
