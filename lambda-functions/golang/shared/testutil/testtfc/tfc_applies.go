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

func (srv *MockTFC) AddApply(applyId string, apply *tfe.Apply) *tfe.Apply {
	// Save the Apply to the mock server
	applyPath := fmt.Sprintf("/api/v2/applies/%s", applyId)
	srv.Applies[applyPath] = apply

	return apply
}

func (srv *MockTFC) HandleAppliesGetRequests(w http.ResponseWriter, r *http.Request) bool {
	apply := srv.Applies[r.URL.Path]
	if apply != nil {
		body, err := json.Marshal(MakeGetApplyResponse(*apply))
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

func MakeGetApplyResponse(apply tfe.Apply) map[string]interface{} {
	selfLink := fmt.Sprintf("/api/v2/applies/%s", apply.ID)

	relationships := map[string]interface{}{}

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
