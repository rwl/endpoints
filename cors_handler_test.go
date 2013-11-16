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
	"encoding/json"
	"github.com/rwl/go-endpoints/endpoints"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

// Test that an error response still handles CORS headers.
func TestHandleNonJsonSpiResponseCors(t *testing.T) {
	serverResponse := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString("This is an invalid response.")),
	}
	response := checkCors(
		t,
		http.Header{"origin": []string{"test.com"}},
		true,
		"test.com",
		"",
		serverResponse,
	)
	errorJson := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Non-JSON reply: This is an invalid response.",
		},
	}
	var responseJson interface{}
	err := json.Unmarshal([]byte(response), &responseJson)
	assert.NoError(t, err)
	assert.Equal(t, errorJson, responseJson)
}

// Check that CORS headers are handled correctly.
//
// requestHeaders: A header map to be used in the request.
//
// expectResponse: Whether or not CORS headers are expected in the response.
//
// expectedOrigin: If specified, this is the value that's expected in the
// response's allow origin header.
//
// expectedAllowHeaders: If specified, this is the value that's expected in
// the response's allow headers header. If this is empty, then the response
// shouldn't have any allow headers headers.
//
// serverResponse: The backend's response, to be wrapped and returned as
// the server's response. If this is nil, a generic response will be generated.
//
// Returns the body of the response that would be sent.
func checkCors(t *testing.T, requestHeaders http.Header, expectResponse bool,
	expectedOrigin, expectedAllowHeaders string, serverResponse *http.Response) string {
	origRequest := buildApiRequest("/_ah/api/fake/path", "", requestHeaders)
	spiRequest, err := origRequest.copy()
	assert.NoError(t, err)

	if serverResponse == nil {
		serverResponse = &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Header:     http.Header{"Content-type": []string{"application/json"}},
			Body:       ioutil.NopCloser(bytes.NewBufferString("{}")),
		}
	}

	server := newEndpointsServer()
	w := httptest.NewRecorder()

	response, err := handleSpiResponse(server, origRequest, spiRequest,
		serverResponse, &endpoints.ApiMethod{}, w)
	assert.NoError(t, err)

	headers := w.Header()
	if expectResponse {
		assert.Equal(t, headers.Get(corsHeaderAllowOrigin), expectedOrigin)
		allowMethods := strings.Split(headers.Get(corsHeaderAllowMethods), ",")
		sort.Strings(allowMethods)
		assert.Equal(t, allowMethods, corsAllowedMethods)
		assert.Equal(t, headers.Get(corsHeaderAllowHeaders), expectedAllowHeaders)
	} else {
		assert.Empty(t, headers.Get(corsHeaderAllowOrigin))
		assert.Empty(t, headers.Get(corsHeaderAllowMethods))
		assert.Empty(t, headers.Get(corsHeaderAllowHeaders))
	}
	return response
}

// Test CORS support on a regular request.
func TestHandleCors(t *testing.T) {
	header := http.Header{"origin": []string{"test.com"}}
	checkCors(t, header, true, "test.com", "", nil)
}

// Test a CORS preflight request.
func TestHandleCorsPreflight(t *testing.T) {
	header := http.Header{
		"origin":                        []string{"http://example.com"},
		"Access-Control-Request-Method": []string{"GET"},
	}
	checkCors(t, header, true, "http://example.com", "", nil)
}

// Test a CORS preflight request for an unaccepted OPTIONS request.
func TestHandleCorsPreflightInvalid(t *testing.T) {
	header := http.Header{
		"origin":                        []string{"http://example.com"},
		"Access-Control-Request-Method": []string{"OPTIONS"},
	}
	checkCors(t, header, false, "", "", nil)
}

// Test a CORS preflight request.
func TestHandleCorsPreflightRequestHeaders(t *testing.T) {
	header := http.Header{
		"origin":                         []string{"http://example.com"},
		"Access-Control-Request-Method":  []string{"GET"},
		"Access-Control-Request-Headers": []string{"Date,Expires"},
	}
	checkCors(t, header, true, "http://example.com", "Date,Expires", nil)
}
