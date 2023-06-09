package testtfc

import (
	"github.com/hashicorp/go-tfe"
	"fmt"
	"net/http"
	"encoding/json"
	"strings"
)

func (srv *MockTFC) AddStateVersion(workspaceId string, stateVersion *tfe.StateVersion) *tfe.StateVersion {
	stateVersion.ID = StateVersionId(workspaceId)

	// Save the StateVersion to the mock server
	srv.StateVersions[workspaceId] = stateVersion

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
	if urlPathParts[3] == "state-versions" && urlPathParts[5] == "outputs" {
		stateVersionId := urlPathParts[4]

		stateVersionOutputs := srv.StateVersionOutputs[stateVersionId]
		if stateVersionOutputs == nil {
			w.WriteHeader(404)
			return true
		}

		body, err := json.Marshal(MakeGetStateVersionOutputsResponse(stateVersionOutputs))
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
			},
			"relationships": relationships,
			"links": map[string]interface{}{
				"self": selfLink,
			},
		},
	}
}

func MakeGetStateVersionOutputsResponse(stateVersionOutputs []*tfe.StateVersionOutput) map[string]interface{} {
	data := make([]map[string]interface{}, 0)

	for _, output := range stateVersionOutputs {
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
	}
}
