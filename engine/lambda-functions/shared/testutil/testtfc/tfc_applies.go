/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package testtfc

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-tfe"
	"net/http"
)

func (srv *MockTFC) AddApply(apply *tfe.Apply) *tfe.Apply {
	apply.ID = ApplyId()

	// Save the Apply to the mock server
	applyPath := fmt.Sprintf("/api/v2/applies/%s", apply.ID)
	srv.Applies[applyPath] = apply

	return apply
}

func (srv *MockTFC) HandleAppliesGetRequests(w http.ResponseWriter, r *http.Request) bool {
	apply := srv.Applies[r.URL.Path]
	if apply != nil {
		body, err := json.Marshal(srv.MakeGetApplyResponse(*apply))
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

func (srv *MockTFC) MakeGetApplyResponse(apply tfe.Apply) map[string]interface{} {
	selfLink := fmt.Sprintf("/api/v2/applies/%s", apply.ID)

	stateVersionRelationships := []map[string]interface{}{}
	for _, stateVersion := range srv.StateVersionsByApply[apply.ID] {
		stateVersionRelationships = append(stateVersionRelationships, map[string]interface{}{
			"id":   stateVersion.ID,
			"type": "state-versions",
		})
	}

	relationships := map[string]interface{}{
		"state-versions": map[string]interface{}{
			"data": stateVersionRelationships,
		},
	}

	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":   apply.ID,
			"type": "applies",
			"attributes": map[string]interface{}{
				"status": apply.Status,
			},
			"relationships": relationships,
			"links": map[string]interface{}{
				"self": selfLink,
			},
		},
	}
}
