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
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
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
