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

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func prepareTestServer(t *testing.T, config *endpoints.ApiDescriptor) *httptest.Server {
	config_bytes, err := json.Marshal(config)
	if err != nil {
		panic("Invalid config")
	}
	response_body, _ := json.Marshal(map[string]interface{}{
		"items": []string{string(config_bytes)},
	})

	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST")
		assert.Equal(t, r.URL.Path, "/_ah/spi/BackendService.getApiConfigs")
		body, _ := ioutil.ReadAll(r.Body)
		assert.Equal(t, string(body), "{}")
		assert.Equal(t, r.Header.Get("Content-Type"), "application/json")

		w.Header().Set("Content-Type", "application/json")
		//w.Header().Set("Content-Length", string(len(response_body)))
		fmt.Fprintln(w, string(response_body))
	})
	ts := httptest.NewServer(hf)
	return ts
}

// Assert that dispatching a request to the SPI works.
func assertDispatchToSpi(t *testing.T, request *apiRequest, config *endpoints.ApiDescriptor, spiPath string,
	expectedSpiBodyJson map[string]interface{}) {
	server := newEndpointsServer()
	orig := handleSpiResponse
	defer func() {
		handleSpiResponse = orig
	}()
	handleSpiResponse = func(ed *EndpointsServer, origRequest, spiRequest *apiRequest, response *http.Response, methodConfig *endpoints.ApiMethod, w http.ResponseWriter) (string, error) {
		fmt.Fprint(w, "Test")
		return "Test", nil
	}
	ts := prepareTestServer(t, config)
	server.url = ts.URL
	defer ts.Close()

	w := httptest.NewRecorder()

	var spiBodyJson map[string]interface{}
	if expectedSpiBodyJson != nil {
		spiBodyJson = expectedSpiBodyJson
	} else {
		spiBodyJson = make(map[string]interface{})
	}

	// todo: compare a string of a JSON object to a JSON object
	spiBody, err := json.Marshal(spiBodyJson)
	assert.NoError(t, err)

	// todo: build a valid response
	//	spi_response := &http.Response{
	//		StatusCode: 200,
	//		Status:     "200 OK",
	//		Body:       ioutil.NopCloser(bytes.NewBufferString("Test")),
	//	}

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST")
		assert.Equal(t, r.URL.Path, spiPath)
		body, _ := ioutil.ReadAll(r.Body)
		assert.Equal(t, string(body), string(spiBody))
		assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
	}))
	defer ts2.Close()

	save := buildSpiUrl
	buildSpiUrl = func(ed *EndpointsServer, spiRequest *apiRequest) string {
		return ts2.URL + fmt.Sprintf(spiRootFormat, spiRequest.URL.Path)
	}
	defer func() {
		buildSpiUrl = save
	}()

	server.serveHTTP(w, request)
	response, err := ioutil.ReadAll(w.Body)
	assert.NoError(t, err)
	assert.Equal(t, "Test", string(response))
}

func TestDispatchInvalidPath(t *testing.T) {
	server := newEndpointsServer()
	config := &endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "v1",
		Methods: map[string]*endpoints.ApiMethod{
			"guestbook.get": &endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "greetings/{gid}",
				RosyMethod: "MyApi.greetings_get",
			},
		},
	}
	request := buildRequest("/_ah/api/foo", "", nil)
	ts := prepareTestServer(t, config)
	server.url = ts.URL
	defer ts.Close()

	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	server.HandleHttp(mux)
	mux.ServeHTTP(w, request)
	server.HandleHttp(nil)

	header := make(http.Header)
	header.Set("Content-Type", "text/plain")
	header.Set("Content-Length", "9")
	assertHttpMatchRecorder(t, w, 404, header, "Not Found")
}

func TestDispatchInvalidEnum(t *testing.T) {
	server := newEndpointsServer()
	config := &endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "v1",
		Methods: map[string]*endpoints.ApiMethod{
			"guestbook.get": &endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "greetings/{gid}",
				RosyMethod: "MyApi.greetings_get",
				Request: endpoints.ApiReqRespDescriptor{
					Body: "empty",
					Params: map[string]*endpoints.ApiRequestParamSpec{
						"gid": &endpoints.ApiRequestParamSpec{
							Enum: map[string]*endpoints.ApiEnumParamSpec{
								"X": &endpoints.ApiEnumParamSpec{
									BackendVal: "X",
								},
							},
							Type: "string",
						},
					},
				},
			},
		},
	}
	w := httptest.NewRecorder()

	request := buildRequest("/_ah/api/guestbook_api/v1/greetings/invalid_enum", "", nil)
	ts := prepareTestServer(t, config)
	server.url = ts.URL
	defer ts.Close()

	server.ServeHTTP(w, request)

	assert.Equal(t, w.Code, 400)
	body := w.Body.Bytes()
	var bodyJson map[string]interface{}
	err := json.Unmarshal(body, &bodyJson)
	assert.NoError(t, err, "Body: %s", string(body))
	errorVal, ok := bodyJson["error"]
	assert.True(t, ok)
	errorJson, ok := errorVal.(map[string]interface{})
	assert.True(t, ok)
	errors, ok := errorJson["errors"]
	assert.True(t, ok)
	errorsJson, ok := errors.([]interface{})
	assert.True(t, ok)
	if len(errorsJson) > 0 {
		errorsJson0, ok := errorsJson[0].(map[string]interface{})
		assert.True(t, ok)
		ok = assert.Equal(t, 1, len(errorsJson))
		if ok {
			assert.Equal(t, "gid", errorsJson0["location"])
			assert.Equal(t, "invalidParameter", errorsJson0["reason"])
		}
	}
}

// Check the error response if the SPI returns an error.
func TestDispatchSpiError(t *testing.T) {
	server := newEndpointsServer()
	config := &endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "v1",
		Methods: map[string]*endpoints.ApiMethod{
			"guestbook.get": &endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "greetings/{gid}",
				RosyMethod: "MyApi.greetings_get",
			},
		},
	}
	request := buildApiRequest("/_ah/api/foo", "", nil)
	ts := prepareTestServer(t, config)
	server.url = ts.URL
	defer ts.Close()

	w := httptest.NewRecorder()

	// The application chose to throw a 404 error.
	response := &http.Response{
		Status:     "404 Not Found",
		StatusCode: 404,
		Body: ioutil.NopCloser(
			bytes.NewBufferString(
				`{"state": "APPLICATION_ERROR", "error_message": "Test error"}`,
			),
		),
	}
	orig := callSpi
	defer func() {
		callSpi = orig
	}()
	callSpi = func(ed *EndpointsServer, w http.ResponseWriter, origRequest *apiRequest) (string, error) {
		assert.Equal(t, origRequest, request)
		return "", newBackendError(response)
	}

	server.serveHTTP(w, request)

	expectedResponse := `{
  "error": {
    "code": 404,
    "errors": [
      {
        "domain": "global",
        "message": "Test error",
        "reason": "notFound"
      }
    ],
    "message": "Test error"
  }
}`
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Content-Length", fmt.Sprintf("%d", len(expectedResponse)))
	assertHttpMatchRecorder(t, w, 404, header, expectedResponse)
}

// Test than an RPC call that returns an error is handled properly.
func TestDispatchRpcError(t *testing.T) {
	server := newEndpointsServer()
	config := &endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "v1",
		Methods: map[string]*endpoints.ApiMethod{
			"guestbook.get": &endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "greetings/{gid}",
				RosyMethod: "MyApi.greetings_get",
			},
		},
	}
	request := buildApiRequest(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X", "id": "gapiRpc"}`,
		nil,
	)
	ts := prepareTestServer(t, config)
	server.url = ts.URL
	defer ts.Close()

	w := httptest.NewRecorder()

	// The application chose to throw a 404 error.
	response := &http.Response{
		Status:     "404 Not Found",
		StatusCode: 404,
		Body: ioutil.NopCloser(
			bytes.NewBufferString(
				`{"state": "APPLICATION_ERROR", "error_message": "Test error"}`,
			),
		),
	}
	orig := callSpi
	defer func() {
		callSpi = orig
	}()
	callSpi = func(ed *EndpointsServer, w http.ResponseWriter, origRequest *apiRequest) (string, error) {
		assert.Equal(t, origRequest, request)
		return "", newBackendError(response)
	}

	/*responseBody := */ server.serveHTTP(w, request)
	responseBody, err := ioutil.ReadAll(w.Body)
	assert.NoError(t, err)

	expectedResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code": 404,
			"data": []interface{}{
				map[string]interface{}{
					"domain":  "global",
					"message": "Test error",
					"reason":  "notFound",
				},
			},
			"message": "Test error",
		},
		"id": "gapiRpc",
	}
	assert.Equal(t, w.Code, 200)
	var responseJson interface{}
	err = json.Unmarshal(responseBody, &responseJson)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, responseJson)
}

func TestDispatchJsonRpc(t *testing.T) {
	config := &endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "X",
		Methods: map[string]*endpoints.ApiMethod{
			"foo.bar": &endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "greetings/{gid}",
				RosyMethod: "baz.bim",
			},
		},
	}
	request := buildApiRequest(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X"}`,
		nil,
	)
	assertDispatchToSpi(t, request, config, "/_ah/spi/baz.bim", nil)
}

func TestDispatchRest(t *testing.T) {
	config := &endpoints.ApiDescriptor{
		Name:    "myapi",
		Version: "v1",
		Methods: map[string]*endpoints.ApiMethod{
			"bar": &endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "foo/{id}",
				RosyMethod: "baz.bim",
			},
		},
	}
	request := buildApiRequest("/_ah/api/myapi/v1/foo/testId", "", nil)
	assertDispatchToSpi(t, request, config, "/_ah/spi/baz.bim",
		map[string]interface{}{"id": "testId"})
}

func TestExplorerRedirect(t *testing.T) {
	server := newEndpointsServer()
	w := httptest.NewRecorder()
	request := buildRequest("/_ah/api/explorer", "", nil)
	server.HandleApiExplorerRequest(w, request)
	header := make(http.Header)
	header.Set("Content-Length", "0")
	location := "http://apis-explorer.appspot.com/apis-explorer/?base=http://localhost:42/_ah/api/"
	header.Set("Location", location)
	body := fmt.Sprintf(`<a href="%s">Found</a>.

`, location) // todo: check if anchor is a valid response body
	assertHttpMatchRecorder(t, w, 302, header, body)
}

func TestStaticExistingFile(t *testing.T) {
	relativeUrl := "/_ah/api/static/proxy.html"

	w := httptest.NewRecorder()
	server := newEndpointsServer()

	testBody := "test body"
	staticResponse := &http.Response{
		Status:        "200 OK",
		StatusCode:    200,
		Header:        http.Header{"Content-Type": []string{"test/type"}},
		ContentLength: int64(len(testBody)),
		Body:          ioutil.NopCloser(bytes.NewBufferString(testBody)),
	}
	orig := getStaticFile
	defer func() {
		getStaticFile = orig
	}()
	getStaticFile = func(path string) (*http.Response, string, error) {
		assert.Equal(t, path, relativeUrl)
		return staticResponse, testBody, nil
	}

	// Make sure the dispatch works as expected.
	request := buildRequest(relativeUrl, "", nil)

	mux := http.NewServeMux()
	server.HandleHttp(mux)
	mux.ServeHTTP(w, request)

	header := make(http.Header)
	header.Set("Content-Length", fmt.Sprintf("%d", len(testBody)))
	header.Set("Content-Type", "test/type")
	assertHttpMatchRecorder(t, w, 200, header, testBody)
}

func TestStaticNonExistingFile(t *testing.T) {
	relativeUrl := "/_ah/api/static/blah.html"

	w := httptest.NewRecorder()
	server := newEndpointsServer()

	testBody := "No Body"
	staticResponse := &http.Response{
		Status:        "404 Not Found",
		StatusCode:    404,
		Header:        http.Header{"Content-Type": []string{"test/type"}},
		ContentLength: int64(len(testBody)),
		Body:          ioutil.NopCloser(bytes.NewBufferString(testBody)),
	}

	orig := getStaticFile
	defer func() {
		getStaticFile = orig
	}()
	getStaticFile = func(path string) (*http.Response, string, error) {
		assert.Equal(t, path, relativeUrl)
		return staticResponse, testBody, nil
	}

	// Make sure the dispatch works as expected.
	request := buildRequest(relativeUrl, "", nil)

	mux := http.NewServeMux()
	server.HandleHttp(mux)
	mux.ServeHTTP(w, request)

	header := make(http.Header)
	header.Set("Content-Length", fmt.Sprintf("%d", len(testBody)))
	header.Set("Content-Type", "test/type")
	assertHttpMatchRecorder(t, w, 404, header, testBody)
}

func TestHandleNonJsonSpiResponse(t *testing.T) {
	server := newEndpointsServer()
	w := httptest.NewRecorder()
	origRequest := buildApiRequest("/_ah/api/fake/path", "", nil)
	spiRequest, err := origRequest.copy()
	assert.NoError(t, err)
	header := make(http.Header)
	header.Set("Content-type", "text/plain")
	spiResponse := &http.Response{
		Header:     header,
		Body:       ioutil.NopCloser(bytes.NewBufferString("This is an invalid response.")),
		StatusCode: 200,
		Status:     "200 OK",
	}
	handleSpiResponse(server, origRequest, spiRequest, spiResponse,
		&endpoints.ApiMethod{}, w)
	errorJson := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Non-JSON reply: This is an invalid response.",
		},
	}
	bodyBytes, _ := json.Marshal(errorJson)
	body := string(bodyBytes)
	expectedHeader := http.Header{
		"Content-Type":   []string{"application/json"},
		"Content-Length": []string{fmt.Sprintf("%d", len(body))},
	}
	assertHttpMatchRecorder(t, w, 500, expectedHeader, body)
}

// Verify Lily protocol correctly uses method name.
//
// This test verifies the fix to http://b/7189819
func TestLilyUsesMethodName(t *testing.T) {
	config := &endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "X",
		Methods: map[string]*endpoints.ApiMethod{
			"author.greeting.info.get": &endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "authors/{aid}/greetings/{gid}/infos/{iid}",
				RosyMethod: "InfoService.get",
			},
		},
	}
	request := buildApiRequest(
		"/_ah/api/rpc",
		`{"method": "author.greeting.info.get", "apiVersion": "X"}`,
		nil,
	)
	assertDispatchToSpi(t, request, config, "/_ah/spi/InfoService.get",
		map[string]interface{}{})
}

// Verify headers transformed, JsonRpc response transformed, written.
func TestHandleSpiResponseJsonRpc(t *testing.T) {
	server := newEndpointsServer()
	w := httptest.NewRecorder()
	origRequest := buildApiRequest(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X"}`,
		nil,
	)
	assert.True(t, origRequest.isRpc())
	origRequest.requestId = "Z"
	spiRequest, err := origRequest.copy()
	assert.NoError(t, err)
	spiResponse := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"a": []string{"b"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"some": "response"}`)),
	}

	response, err := handleSpiResponse(server, origRequest, spiRequest,
		spiResponse, &endpoints.ApiMethod{}, w)
	assert.NoError(t, err)

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Header()["a"][0], "b")
	expectedResponse := map[string]interface{}{
		"id":     "Z",
		"result": map[string]interface{}{"some": "response"},
	}
	var responseJson interface{}
	err = json.Unmarshal([]byte(response), &responseJson)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, responseJson)
}

// Verify that batch requests have an appropriate batch response.
func TestHandleSpiResponseBatchJsonRpc(t *testing.T) {
	server := newEndpointsServer()
	w := httptest.NewRecorder()
	origRequest := buildApiRequest(
		"/_ah/api/rpc",
		`[{"method": "foo.bar", "apiVersion": "X"}]`,
		nil,
	)
	assert.True(t, origRequest.isBatch)
	assert.True(t, origRequest.isRpc())
	origRequest.requestId = "Z"
	spiRequest, err := origRequest.copy()
	assert.NoError(t, err)
	spiResponse := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"a": []string{"b"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"some": "response"}`)),
	}

	response, err := handleSpiResponse(server, origRequest, spiRequest,
		spiResponse, &endpoints.ApiMethod{}, w)
	assert.NoError(t, err)

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Header()["a"][0], "b")
	expectedResponse := []interface{}{
		map[string]interface{}{
			"id":     "Z",
			"result": map[string]interface{}{"some": "response"},
		},
	}
	var responseJson interface{}
	err = json.Unmarshal([]byte(response), &responseJson)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, responseJson)
}

func TestHandleSpiResponseRest(t *testing.T) {
	server := newEndpointsServer()
	w := httptest.NewRecorder()
	origRequest := buildApiRequest("/_ah/api/test", "{}", nil)
	spiRequest, err := origRequest.copy()
	assert.NoError(t, err)
	body, _ := json.MarshalIndent(map[string]interface{}{
		"some": "response",
	}, "", "  ")
	spiResponse := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"a": []string{"b"}},
		Body:       ioutil.NopCloser(bytes.NewBuffer(body)),
	}
	_, err = handleSpiResponse(server, origRequest, spiRequest,
		spiResponse, &endpoints.ApiMethod{}, w)
	assert.NoError(t, err)
	header := http.Header{
		"a":              []string{"b"},
		"Content-Length": []string{fmt.Sprintf("%d", len(body))},
	}
	assertHttpMatchRecorder(t, w, 200, header, string(body))
}

// Verify the response is reformatted correctly.
func TestTransformRestResponse(t *testing.T) {
	server := newEndpointsServer()
	origResponse := `{"sample": "test", "value1": {"value2": 2}}`
	expectedResponse := `{
  "sample": "test",
  "value1": {
    "value2": 2
  }
}`
	response, err := server.transformRestResponse(origResponse)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}

// Verify requestId inserted into the body, and body into body.result.
func TestTransformJsonRpcResponseBatch(t *testing.T) {
	server := newEndpointsServer()
	origRequest := buildApiRequest(
		"/_ah/api/rpc",
		`[{"params": {"sample": "body"}, "id": "42"}]`,
		nil,
	)
	request, err := origRequest.copy()
	assert.NoError(t, err)
	request.requestId = "42"
	origResponse := `{"sample": "body"}`
	response, err := server.transformJsonrpcResponse(request, origResponse)
	assert.NoError(t, err)
	expectedResponse := []map[string]interface{}{
		map[string]interface{}{
			"result": map[string]interface{}{"sample": "body"},
			"id":     "42",
		},
	}
	var responseJson []map[string]interface{}
	err = json.Unmarshal([]byte(response), &responseJson)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, responseJson)
}

func TestLookupRpcMethodNoBody(t *testing.T) {
	server := newEndpointsServer()
	origRequest := buildApiRequest("/_ah/api/rpc", "", nil)
	assert.Nil(t, server.lookupRpcMethod(origRequest))
}

/*func TestLookupRpcMethod(t *testing.T) {
	mox.StubOutWithMock(server.configManager, "lookupRpcMethod")
	server.configManager.lookupRpcMethod("foo", "v1").AndReturn("bar")

	mox.ReplayAll()
	origRequest := buildApiRequest(
		"/_ah/api/rpc",
		`{"method": "foo", "apiVersion": "v1"}`,
		nil,
	)
	if "bar" != server.lookupRpcMethod(origRequest) {
		t.Fail()
	}
	mox.VerifyAll()
}*/

func TestVerifyResponse(t *testing.T) {
	response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"a"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString("")),
	}
	// Expected response
	assert.NoError(t, verifyResponse(response, 200, "a"))
	// Any content type accepted
	assert.NoError(t, verifyResponse(response, 200, ""))
	// Status code mismatch
	assert.Error(t, verifyResponse(response, 400, "a"))
	// Content type mismatch
	assert.Error(t, verifyResponse(response, 200, "b"))

	response = &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"Content-Length": []string{"10"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString("")),
	}
	// Any content type accepted
	assert.NoError(t, verifyResponse(response, 200, ""))
	// Specified content type not matched
	assert.Error(t, verifyResponse(response, 200, "a"))
}

// Test that checkEmptyResponse returns 204 for an empty response.
func TestCheckEmptyResponse(t *testing.T) {
	server := newEndpointsServer()
	w := httptest.NewRecorder()
	origRequest := buildApiRequest("/_ah/api/test", "{}", nil)
	methodConfig := &endpoints.ApiMethod{
		Response: endpoints.ApiReqRespDescriptor{
			Body: "empty",
		},
	}
	/*emptyResponse := */ server.checkEmptyResponse(origRequest,
		methodConfig, w)
	header := http.Header{
		"Content-Length": []string{"0"},
	}
	assertHttpMatchRecorder(t, w, 204, header, "")
}

// Test that check_empty_response returns None for a non-empty response.
func TestCheckNonEmptyResponse(t *testing.T) {
	server := newEndpointsServer()
	w := httptest.NewRecorder()
	origRequest := buildApiRequest("/_ah/api/test", "{}", nil)
	methodConfig := &endpoints.ApiMethod{
		Response: endpoints.ApiReqRespDescriptor{
			Body: "autoTemplate(backendResponse)",
		},
	}
	emptyResponse := server.checkEmptyResponse(origRequest,
		methodConfig, w)
	assert.Empty(t, emptyResponse)
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, len(w.Header()), 0)
	//assert.Nil(t, w.responseExcInfo)
}
