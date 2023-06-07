package testutil

import (
	"github.com/hashicorp/go-tfe"
	"fmt"
	"net/http"
	"strings"
	"log"
	"encoding/json"
)

func (srv *MockTFC) AddConfigurationVersion(workspaceId string, configVersion *tfe.ConfigurationVersion) *tfe.ConfigurationVersion {
	configVersion.ID = ConfigVersionId(workspaceId)

	uploadUrl := fmt.Sprintf("%s/configuration-version-uploads/%s", srv.Address, configVersion.ID)
	configVersion.UploadURL = uploadUrl

	// Save the configuration version to the server's 'configuration versions by id" map
	srv.configurationVersionsById[configVersion.ID] = configVersion

	// Save the configuration version to the server's mappings of configuration versions to workspaces
	configVersions := make([]*tfe.ConfigurationVersion, 0)
	if existingConfigVersions := srv.configurationVersionsByWorkspace[workspaceId]; existingConfigVersions != nil {
		configVersions = existingConfigVersions
	}
	srv.configurationVersionsByWorkspace[workspaceId] = append(configVersions, configVersion)

	return configVersion
}

func (srv *MockTFC) HandleConfigurationVersionsPostRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/workspaces/ws-2jmj7l5rSw0yVb_v/configuration-versions => "", "api", "v2" "workspaces" "ws-2jmj7l5rSw0yVb_v" "configuration-versions"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if urlPathParts[3] == "workspaces" && urlPathParts[5] == "configuration-versions" {
		workspaceId := urlPathParts[4]

		var configVersion *tfe.ConfigurationVersion
		if err := json.NewDecoder(r.Body).Decode(&configVersion); err != nil {
			w.WriteHeader(500)
			return true
		}

		configVersion = srv.AddConfigurationVersion(workspaceId, configVersion)

		body, err := json.Marshal(MakeConfigurationVersionResponse(configVersion))
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

func (srv *MockTFC) HandleConfigurationVersionsUploads(w http.ResponseWriter, r *http.Request) bool {
	// /configuration-version-uploads/cv-ZHZqfDii4BjHoGuL => "", "configuration-version-uploads", "cv-ZHZqfDii4BjHoGuL"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if urlPathParts[1] == "configuration-version-uploads" {
		configVersionId := urlPathParts[2]

		// set the configuration version's status to uploaded
		configVersion := srv.configurationVersionsById[configVersionId]
		configVersion.Status = tfe.ConfigurationUploaded
		srv.configurationVersionsById[configVersionId] = configVersion

		w.WriteHeader(200)
		return true
	}

	return false
}

func (srv *MockTFC) HandleConfigurationVersionsGetRequests(w http.ResponseWriter, r *http.Request) bool {
	// /api/v2/configuration-versions/cv-WgF_1Sl_BfL3AOgT => "", "api", "v2" "configuration-versions" "cv-WgF_1Sl_BfL3AOgT"
	urlPathParts := strings.Split(r.URL.Path, "/")

	if urlPathParts[3] == "configuration-versions" {
		configurationVersionId := urlPathParts[4]

		configurationVersion := srv.configurationVersionsById[configurationVersionId]

		body, err := json.Marshal(MakeConfigurationVersionResponse(configurationVersion))
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

//func MakeListVarsResponse(vars []*tfe.Variable) map[string]interface{} {
//
//	data := make([]map[string]interface{}, 0)
//
//	for _, variable := range vars {
//		selfLink := fmt.Sprintf("/api/v2/vars/%s", variable.ID)
//		datum := map[string]interface{}{
//			"id":   variable.ID,
//			"type": "vars",
//			"attributes": map[string]interface{}{
//				"key":   variable.Key,
//				"value": variable.Value,
//			},
//			"relationships": map[string]interface{}{},
//			"links": map[string]interface{}{
//				"self": selfLink,
//			},
//		}
//
//		data = append(data, datum)
//	}
//
//	return map[string]interface{}{
//		"data": data,
//	}
//}

func MakeConfigurationVersionResponse(configVersion *tfe.ConfigurationVersion) map[string]interface{} {
	selfLink := fmt.Sprintf("/api/v2/configuration-versions/%s", configVersion.ID)

	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":   configVersion.ID,
			"type": "configuration-versions",
			"attributes": map[string]interface{}{
				"upload-url": configVersion.UploadURL,
				"status":     configVersion.Status,
			},
		},
		"relationships": map[string]interface{}{},
		"links": map[string]interface{}{
			"self": selfLink,
		},
	}
}
