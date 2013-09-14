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

package endpoint

import (
	"encoding/json"
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"github.com/golang/glog"
	"net/http"
)

const GET_REST_API = "apisdev.getRest"
const GET_RPC_API = "apisdev.getRpc"
const LIST_API = "apisdev.list"

var DiscoveryApiConfig = &endpoints.ApiDescriptor{
	Name:    "discovery",
	Version: "v1",
	Methods: map[string]*endpoints.ApiMethod{
		"discovery.apis.getRest": &endpoints.ApiMethod{
			Path:       "apis/{api}/{version}/rest",
			HttpMethod: "GET",
			RosyMethod: GET_REST_API,
		},
		"discovery.apis.getRpc": &endpoints.ApiMethod{
			Path:       "apis/{api}/{version}/rpc",
			HttpMethod: "GET",
			RosyMethod: GET_RPC_API,
		},
		"discovery.apis.list": &endpoints.ApiMethod{
			Path:       "apis",
			HttpMethod: "GET",
			RosyMethod: LIST_API,
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
type DiscoveryService struct {
	configManager *ApiConfigManager
}

func NewDiscoveryService(config_manager *ApiConfigManager) *DiscoveryService {
	return &DiscoveryService{config_manager}
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
func (ds *DiscoveryService) getRpcOrRest(apiFormat ApiFormat, request *ApiRequest, w http.ResponseWriter) string {
	api, ok := request.BodyJson["api"]
	version, _ := request.BodyJson["version"]
	apiStr, _ := api.(string)
	versionStr, _ := version.(string)
	lookupKey := lookupKey{apiStr, versionStr}
	apiConfig, ok := ds.configManager.configs[lookupKey]
	if !ok {
		glog.Infof("No discovery doc for version %s of api %s", version, api)
		return sendNotFoundResponse(w, nil)
	}
	doc, err := generateDiscoveryDoc(apiConfig, apiFormat)
	if err != nil {
		errorMsg := fmt.Sprintf(`Failed to convert .api to discovery doc for version "%s" of api "%s": %s`, version, api, err.Error())
		glog.Error(errorMsg)
		return sendErrorResponse(errorMsg, w, nil)
	}
	return sendSuccessResponse(doc, w)
}

// Sends HTTP response containing the API directory.
func (ds *DiscoveryService) list(w http.ResponseWriter) string {
	apiConfigs := make([]string, 0)
	for _, apiConfig := range ds.configManager.configs {
		if apiConfig != DiscoveryApiConfig {
			ac, err := json.Marshal(apiConfig)
			if err != nil {
				glog.Errorf("Failed to marshal API config: %v", apiConfig)
				return sendNotFoundResponse(w, nil)
			}
			apiConfigs = append(apiConfigs, string(ac))
		}
	}
	directory, err := generateDiscoveryDirectory(apiConfigs)
	if err != nil {
		glog.Errorf("Failed to get API directory: %s", err.Error())
		// By returning a 404, code explorer still works if you select the
		// API in the URL
		return sendNotFoundResponse(w, nil)
	}
	return sendSuccessResponse(directory, w)
}

// Returns the result of a discovery service request and false if the request
// wasn't handled by DiscoveryService.
func (ds *DiscoveryService) handleDiscoveryRequest(path string, request *ApiRequest, w http.ResponseWriter) (string, bool) {
	switch path {
	case GET_REST_API:
		return ds.getRpcOrRest(REST, request, w), true
	case GET_RPC_API:
		return ds.getRpcOrRest(RPC, request, w), true
	case LIST_API:
		return ds.list(w), true
	}
	return "", false
}
