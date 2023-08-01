/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package testtfc

import (
	"encoding/json"
	"github.com/hashicorp/go-tfe"
	"log"
	"net/http"
	"strings"
)

func (srv *MockTFC) HandleTokensPostRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/teams/team-roLYatraNNailuJ2/authentication-token => "", "api", "v2" "teams" "team-roLYatraNNailuJ2" "authentication-token"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if len(urlPathParts) < 6 {
		return false
	}
	if urlPathParts[3] == "teams" && urlPathParts[5] == "authentication-token" {
		teamId := urlPathParts[4]

		teamToken := &tfe.TeamToken{ID: teamId, Token: "newsupers3cret"}
		srv.SetToken(teamToken.Token)

		body, err := json.Marshal(MakeTeamTokenResponse(teamToken))
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

func MakeTeamTokenResponse(teamToken *tfe.TeamToken) map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":   "1337023",
			"type": "authentication-tokens",
			"attributes": map[string]interface{}{
				"created-at":   teamToken.CreatedAt,
				"last-used-at": teamToken.LastUsedAt,
				"description":  teamToken.Description,
				"token":        teamToken.Token,
				"expired-at":   "",
			},
		},
		"relationships": map[string]interface{}{},
	}
}
