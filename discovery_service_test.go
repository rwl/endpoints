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
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func commonSetup() (*apiConfigManager, *apiRequest, *discoveryService) {
	apiConfigMap := map[string]interface{}{"items": []string{apiConfigJson}}
	apiConfigManager := newApiConfigManager()
	apiConfig, _ := json.Marshal(apiConfigMap)
	apiConfigManager.parseApiConfigResponse(string(apiConfig))

	apiRequest := buildApiRequest("/_ah/api/foo",
		`{"api": "tictactoe", "version": "v1"}`, nil)

	discovery := newDiscoveryService(apiConfigManager)

	return apiConfigManager, apiRequest, discovery
}

func TestGenerateDiscoveryDocRestService(t *testing.T) {
	_, apiRequest, discovery := commonSetup()
	body, _ := json.Marshal(map[string]interface{}{
		"baseUrl": "https://tictactoe.appspot.com/_ah/api/tictactoe/v1/",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, string(body))
	}))
	defer ts.Close()
	discoveryProxyHost = ts.URL

	w := httptest.NewRecorder()

	discovery.handleDiscoveryRequest(getRestApi, apiRequest, w)

	assertHttpMatchRecorder(t, w, 200,
		http.Header{
			"Content-Type":   []string{"application/json; charset=UTF-8"},
			"Content-Length": []string{fmt.Sprintf("%d", len(body))},
		}, string(body))
}

func TestGenerateDiscoveryDocRpcService(t *testing.T) {
	_, apiRequest, discovery := commonSetup()
	body, _ := json.Marshal(map[string]interface{}{
		"rpcUrl": "https://tictactoe.appspot.com/_ah/api/rpc",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, string(body))
	}))
	defer ts.Close()
	discoveryProxyHost = ts.URL

	w := httptest.NewRecorder()

	discovery.handleDiscoveryRequest(getRpcApi, apiRequest, w)

	assertHttpMatchRecorder(t, w, 200,
		http.Header{
			"Content-Type":   []string{"application/json; charset=UTF-8"},
			"Content-Length": []string{fmt.Sprintf("%d", len(body))},
		}, string(body))
}

func TestGenerateDiscoveryDocRestUnknownApi(t *testing.T) {
	_, _, discoveryApi := commonSetup()
	request := buildApiRequest("/_ah/api/foo",
		`{"api": "blah", "version": "v1"}`, nil)
	w := httptest.NewRecorder()
	discoveryApi.handleDiscoveryRequest(getRestApi, request, w)
	assert.Equal(t, w.Code, 404)
}

func TestGenerateDirectory(t *testing.T) {
	_, apiRequest, discovery := commonSetup()
	body, _ := json.Marshal(map[string]interface{}{"kind": "discovery#directoryItem"})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, string(body))
	}))
	defer ts.Close()
	discoveryProxyHost = ts.URL

	w := httptest.NewRecorder()

	discovery.handleDiscoveryRequest(listApi, apiRequest, w)

	assertHttpMatchRecorder(t, w, 200,
		http.Header{
			"Content-Type":   []string{"application/json; charset=UTF-8"},
			"Content-Length": []string{fmt.Sprintf("%d", len(body))},
		}, string(body))
}
