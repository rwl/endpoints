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
	"bytes"
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"log"
)

// Build an apiRequest for the given path and body.
func buildApiRequest(url, body string, httpHeaders http.Header) *apiRequest {
	req := buildRequest(url, body, httpHeaders)
	apiRequest, err := newApiRequest(req)
	if err != nil {
		log.Fatal(err.Error())
	}

	return apiRequest
}

// Build a Request for the given path and body.
func buildRequest(url, body string, httpHeaders http.Header) *http.Request {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:42%s", url),
		ioutil.NopCloser(bytes.NewBufferString(body)))
	if err != nil {
		log.Fatal(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	if httpHeaders != nil {
		for key, value := range httpHeaders {
			req.Header.Set(key, value[0])
		}
	}
	return req
}

// Test that the headers and body match.
func assertHttpMatch(t *testing.T, response *http.Response, expectedStatus int,
	expectedHeaders http.Header, expectedBody string) {
	assert.Equal(t, expectedStatus, response.StatusCode)

	// Verify that headers match. Order shouldn't matter.
	assert.Equal(t, response.Header, expectedHeaders)

	// Convert the body to a string.
	body, _ := ioutil.ReadAll(response.Body)
	assert.Equal(t, expectedBody, string(body))
}

// Test that the headers and body match.
func assertHttpMatchRecorder(t *testing.T, recorder *httptest.ResponseRecorder,
	expectedStatus int, expectedHeaders http.Header, expectedBody string) {
	assert.Equal(t, expectedStatus, recorder.Code)

	// Verify that headers match. Order shouldn't matter.
	assert.Equal(t, recorder.Header(), expectedHeaders)

	// Convert the body to a string.
	assert.Equal(t, expectedBody, recorder.Body.String())
}

type MockEndpointsServer struct {
	mock.Mock
	*EndpointsServer
}

func newMockEndpointsServer() *MockEndpointsServer {
	return &MockEndpointsServer{
		EndpointsServer: newEndpointsServer(),
	}
}

// fixme: mock handleSpiResponse without duplicating serveHTTP
func (ed *MockEndpointsServer) serveHTTP(w http.ResponseWriter, ar *apiRequest) string {
	apiConfigResponse, _ := ed.getApiConfigs()
	ed.handleApiConfigResponse(apiConfigResponse)
	body, _ := ed.callSpi(w, ar)
	return body
}

// fixme: mock handleSpiResponse without duplicating callSpi
func (ed *MockEndpointsServer) callSpi(w http.ResponseWriter, origRequest *apiRequest) (string, error) {
	var methodConfig *endpoints.ApiMethod
	var params map[string]string
	if origRequest.isRpc() {
		methodConfig = ed.lookupRpcMethod(origRequest)
		params = nil
	} else {
		methodConfig, params = ed.lookupRestMethod(origRequest)
	}

	spiRequest, _ := ed.transformRequest(origRequest, params, methodConfig)

	discovery := newDiscoveryService(ed.configManager)
	discoveryResponse, ok := discovery.handleDiscoveryRequest(
		spiRequest.URL.Path, spiRequest, w)
	if ok {
		return discoveryResponse, nil
	}

	url := fmt.Sprintf(spiRootFormat, spiRequest.URL.Path)
	req, _ := http.NewRequest("POST", url, spiRequest.Body)
	req.Header.Add("Content-Type", "application/json")
	req.RemoteAddr = spiRequest.RemoteAddr
	client := &http.Client{}
	resp, _ := client.Do(req)
	return ed.handleSpiResponse(origRequest, spiRequest, resp, methodConfig, w)
}

func (ed *MockEndpointsServer) handleSpiResponse(origRequest, spiRequest *apiRequest,
	response *http.Response, methodConfig *endpoints.ApiMethod, w http.ResponseWriter) (string, error) {
	args := ed.Mock.Called(origRequest, spiRequest, methodConfig, response, w)
	return args.String(0), args.Error(1)
}

type MockEndpointsServerSpi struct {
	mock.Mock
	*EndpointsServer
}

func newMockEndpointsServerSpi() *MockEndpointsServerSpi {
	return &MockEndpointsServerSpi{
		EndpointsServer: newEndpointsServer(),
	}
}

// fixme: mock callSpi without duplicating serveHTTP
func (ed *MockEndpointsServerSpi) serveHTTP(w http.ResponseWriter, ar *apiRequest) string {
	// Get API configuration first.  We need this so we know how to
	// call the back end.
	apiConfigResponse, err := ed.getApiConfigs()
	if err != nil {
		return ed.failRequest(w, ar.Request, "BackendService.getApiConfigs Error: "+err.Error())
	}
	err = ed.handleApiConfigResponse(apiConfigResponse)
	if err != nil {
		return ed.failRequest(w, ar.Request, "BackendService.getApiConfigs Error: "+err.Error())
	}

	// Call the service.
	body, err := ed.callSpi(w, ar)
	if err != nil {
		reqErr, ok := err.(requestError)
		if ok {
			return ed.handleRequestError(w, ar, reqErr)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return body
		}
	}
	return body
}

func (ed *MockEndpointsServerSpi) callSpi(w http.ResponseWriter, origRequest *apiRequest) (string, error) {
	args := ed.Mock.Called(w, origRequest)
	return args.String(0), args.Error(1)
}
