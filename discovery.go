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
	"io/ioutil"
	"net/http"
)

var (
	// The endpoint host we're using to proxy discovery and static requests.
	// Using separate constants to make it easier to change the discovery service.
	discoveryProxyHost     = "https://webapis-discovery.appspot.com"
	staticProxyHost        = "https://webapis-discovery.appspot.com"
	discoveryApiPathPrefix = "/_ah/api/discovery/v1/"
)

type apiFormat string

const (
	rest apiFormat = "rest"
	rpc  apiFormat = "rpc"
)

// Proxies GET requests to the discovery service API. Takes the URL path
// relative to discovery service and the HTTP POST request body and returns
// the HTTP response body or an error if it failed.
func dispatchDiscoveryRequest(path, body string) (string, error) {
	fullPath := discoveryProxyHost + discoveryApiPathPrefix + path
	client := &http.Client{}

	req, err := http.NewRequest("POST", fullPath, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	responseBody := string(respBody)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Discovery API proxy failed on %s with %d.\r\nRequest: %s\r\nResponse: %s",
			fullPath, resp.StatusCode, body, responseBody)
	}
	return responseBody, nil
}

// Generates a discovery document from an API file. Takes the .api file
// contents and the kind of discvoery doc requested and returns the discovery
// doc as JSON string.
func generateDiscoveryDoc(apiConfig *endpoints.ApiDescriptor, apiFormat apiFormat) (string, error) {
	path := "apis/generate/" + string(apiFormat)
	requestMap := map[string]interface{}{"config": apiConfig}
	requestBody, err := json.Marshal(requestMap)
	if err != nil {
		return "", err
	}
	return dispatchDiscoveryRequest(path, string(requestBody))
}

// Generates an API directory from a list of API files. Takes an array of
// .api file contents and returns the API directory as JSON string.
func generateDiscoveryDirectory(apiConfigs []string) (string, error) {
	requestMap := map[string]interface{}{"configs": apiConfigs}
	requestBody, err := json.Marshal(requestMap)
	if err != nil {
		return "", err
	}
	return dispatchDiscoveryRequest("apis/generate/directory", string(requestBody))
}

// Returns static content via a GET request. Takes the URL path after the
// domain and returns a Response from the static proxy host and the response
// body.
func getStaticFile(path string) (*http.Response, string, error) {
	resp, err := http.Get(staticProxyHost + path)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return resp, string(body), nil
}
