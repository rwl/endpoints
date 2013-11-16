// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package endpoints_server

import (
	"encoding/json"
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"log"
	"net/http"
)

const getRestApi = "apisdev.getRest"
const getRpcApi = "apisdev.getRpc"
const listApi = "apisdev.list"

var discoveryApiConfig = &endpoints.ApiDescriptor{
	Name:    "discovery",
	Version: "v1",
	Methods: map[string]*endpoints.ApiMethod{
		"discovery.apis.getRest": &endpoints.ApiMethod{
			Path:       "apis/{api}/{version}/rest",
			HttpMethod: "GET",
			RosyMethod: getRestApi,
		},
		"discovery.apis.getRpc": &endpoints.ApiMethod{
			Path:       "apis/{api}/{version}/rpc",
			HttpMethod: "GET",
			RosyMethod: getRpcApi,
		},
		"discovery.apis.list": &endpoints.ApiMethod{
			Path:       "apis",
			HttpMethod: "GET",
			RosyMethod: listApi,
		},
	},
}

// Implements the local devserver discovery service.
//
// This has a static minimal version of the discoverable part of the
// discovery .api file.
//
// It only handles returning the discovery doc and directory, and ignores
// directory parameters to filter the results.
//
// The discovery docs/directory are created by calling a cloud endpoint
// discovery service to generate the discovery docs/directory from an .api
// file/set of .api files.
type discoveryService struct {
	configManager *apiConfigManager
}

func newDiscoveryService(config_manager *apiConfigManager) *discoveryService {
	return &discoveryService{config_manager}
}

// Sends an HTTP 200 json success response with the given body.
func sendSuccessResponse(response string, w http.ResponseWriter) string {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(response)))
	fmt.Fprintf(w, response)
	return response
}

// Sends back HTTP response with API directory. It will return
// the discovery doc for the requested api/version.
func (ds *discoveryService) getRpcOrRest(apiFormat apiFormat, request *apiRequest, w http.ResponseWriter) string {
	api, ok := request.bodyJson["api"]
	version, _ := request.bodyJson["version"]
	apiStr, _ := api.(string)
	versionStr, _ := version.(string)
	lookupKey := lookupKey{apiStr, versionStr}
	apiConfig, ok := ds.configManager.configs()[lookupKey]
	if !ok {
		log.Printf("No discovery doc for version %s of api %s", version, api)
		return sendNotFoundResponse(w, nil)
	}
	doc, err := generateDiscoveryDoc(apiConfig, apiFormat)
	if err != nil {
		errorMsg := fmt.Sprintf(`Failed to convert .api to discovery doc for version "%s" of api "%s": %s`, version, api, err.Error())
		log.Println(errorMsg)
		return sendErrorResponse(errorMsg, w, nil)
	}
	return sendSuccessResponse(doc, w)
}

// Sends HTTP response containing the API directory.
func (ds *discoveryService) list(w http.ResponseWriter) string {
	apiConfigs := make([]string, 0)
	for _, apiConfig := range ds.configManager.configs() {
		if apiConfig != discoveryApiConfig {
			ac, err := json.Marshal(apiConfig)
			if err != nil {
				log.Printf("Failed to marshal API config: %v", apiConfig)
				return sendNotFoundResponse(w, nil)
			}
			apiConfigs = append(apiConfigs, string(ac))
		}
	}
	directory, err := generateDiscoveryDirectory(apiConfigs)
	if err != nil {
		log.Printf("Failed to get API directory: %s", err.Error())
		// By returning a 404, code explorer still works if you select the
		// API in the URL
		return sendNotFoundResponse(w, nil)
	}
	return sendSuccessResponse(directory, w)
}

// Returns the result of a discovery service request and false if the request
// wasn't handled by discoveryService.
func (ds *discoveryService) handleDiscoveryRequest(path string, request *apiRequest, w http.ResponseWriter) (string, bool) {
	switch path {
	case getRestApi:
		return ds.getRpcOrRest(rest, request, w), true
	case getRpcApi:
		return ds.getRpcOrRest(rpc, request, w), true
	case listApi:
		return ds.list(w), true
	}
	return "", false
}
