package endpoint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
//
// Mock out the dispatcher.add_request and handle_spi_response, and use these
// to ensure that the correct request is being sent to the back end when
// Dispatch is called.
//
// Args:
//   request: An ApiRequest, the request to dispatch.
//   config: A dict containing the API configuration.
//   spi_path: A string containing the relative path to the SPI.
//   expected_spi_body_json: If not None, this is a JSON object containing
//     the mock response sent by the back end.  If None, this will create an
//     empty response.
func assertDispatchToSpi(t *testing.T, request *ApiRequest, config *endpoints.ApiDescriptor, spiPath string,
	expectedSpiBodyJson map[string]interface{}) {
	server := newMockEndpointsServer()
	ts := prepareTestServer(t, config)
	server.URL = ts.URL
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

		fmt.Fprint(w, "Test")
	}))
	defer ts2.Close()
	orig := SpiRootFormat
	SpiRootFormat = ts2.URL + SpiRootFormat
	defer func() {
		SpiRootFormat = orig
	}()

	server.On(
		"handleSpiResponse",
		mock.Anything, //OfType("*ApiRequest"),
		mock.Anything, //OfType("*ApiRequest"),
		mock.Anything, //spi_response,
		w,
	).Return("Test", nil)

	response := server.serveHTTP(w, request)
	server.Mock.AssertExpectations(t)

	assert.Equal(t, "Test", response)
}

func TestDispatchInvalidPath(t *testing.T) {
	server := NewEndpointsServer()
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
	server.URL = ts.URL
	defer ts.Close()

	w := httptest.NewRecorder()

	server.HandleHttp(nil)
	http.DefaultServeMux.ServeHTTP(w, request)

	header := make(http.Header)
	header.Set("Content-Type", "text/plain")
	header.Set("Content-Length", "9")
	assertHttpMatchRecorder(t, w, 404, header, "Not Found")
}

func TestDispatchInvalidEnum(t *testing.T) {
	server := NewEndpointsServer()
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
	server.URL = ts.URL
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
	server := newMockEndpointsServerSpi()
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
	server.URL = ts.URL
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
	server.On(
		"callSpi",
		mock.Anything,
		request,
	).Return("", NewBackendError(response))

	server.serveHTTP(w, request)
	server.Mock.AssertExpectations(t)

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
	server := newMockEndpointsServerSpi()
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
	server.URL = ts.URL
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
	server.On(
		"callSpi",
		mock.Anything,
		request,
	).Return("", NewBackendError(response))

	responseBody := server.serveHTTP(w, request)
	server.Mock.AssertExpectations(t)

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
	err := json.Unmarshal([]byte(responseBody), &responseJson)
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
	server := NewEndpointsServer()
	w := httptest.NewRecorder()
	request := buildRequest("/_ah/api/explorer", "", nil)
	server.HandleApiExplorerRequest(w, request)
	header := make(http.Header)
	//header.Set("Content-Length", "0")
	location := "https://developers.google.com/apis-explorer/?base=http://localhost:42/_ah/api"
	header.Set("Location", location)
	body := fmt.Sprintf(`<a href="%s">Found</a>.

`, location) // todo: check if anchor is a valid response body
	assertHttpMatchRecorder(t, w, 302, header, body)
}

/*func TestStaticExistingFile(t *testing.T) {
	relativeUrl := "/_ah/api/static/proxy.html"

	w := httptest.NewRecorder()

	// Set up mocks for the call to DiscoveryApiProxy.get_static_file.
	discoveryApi := &MockDiscoveryApiProxy{}
	server := NewEndpointsServerConfig(
		&http.Client{},
		NewApiConfigManager(),
		discoveryApi,
	)
	testBody := "test body"
	discoveryApi.On(
		"getStaticFile",
		relativeUrl,
	).Return(mock.Anything, //staticResponse,
		test_body, nil)

	// Make sure the dispatch works as expected.
	request := buildApiRequest(relativeUrl, "", nil)
	response := server.dispatch(request, w)
	server.Mock.AssertExpectations(t)

	header := new(Header)
	header.Set("Content-Length", fmt.Sprintf("%d", len(testBody)))
	header.Set("Content-Type", "test/type")
	assert_http_match(t, response, 200, header, testBody)
}*/

/*func TestStaticNonExistingFile(t *testing.T) {
	relativeUrl := "/_ah/api/static/blah.html"

	// Set up mocks for the call to getStaticFile.
	discoveryApi = mox.CreateMock(DiscoveryApiProxy)
	mox.StubOutWithMock(discoveryApiProxy, "DiscoveryApiProxy")
	discoveryApiProxy.DiscoveryApiProxy().AndReturn(discoveryApi)
	staticResponse = mox.CreateMock(httplib.HTTPResponse)
	staticResponse.status = 404
	staticResponse.reason = "Not Found"
	staticResponse.Headers().AndReturn(map[string]string{"Content-Type": "test/type"})
	testBody = "No Body"
	getStaticFile(relativeUrl).AndReturn(staticResponse, testBody)

	// Make sure the dispatch works as expected.
	request = buildApiRequest(relativeUrl, "", nil)
	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
	mox.VerifyAll()

	response := "".join(response)
	header := new(Header)
	header.Set("Content-Length", fmt.Sprintf("%d", len(testBody)))
	header.Set("Content-Type", "test/type")
	assertHttpMatch(t, response, 404, header, testBody)
}*/

func TestHandleNonJsonSpiResponse(t *testing.T) {
	server := NewEndpointsServer()
	w := httptest.NewRecorder()
	origRequest := buildApiRequest("/_ah/api/fake/path", "", nil)
	spiRequest, err := origRequest.Copy()
	assert.NoError(t, err)
	header := make(http.Header)
	header.Set("Content-type", "text/plain")
	spiResponse := &http.Response{
		Header:     header,
		Body:       ioutil.NopCloser(bytes.NewBufferString("This is an invalid response.")),
		StatusCode: 200,
		Status:     "200 OK",
	}
	server.handleSpiResponse(origRequest, spiRequest, spiResponse, w)
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
	server := NewEndpointsServer()
	w := httptest.NewRecorder()
	origRequest := buildApiRequest(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X"}`,
		nil,
	)
	assert.True(t, origRequest.IsRpc())
	origRequest.RequestId = "Z"
	spiRequest, err := origRequest.Copy()
	assert.NoError(t, err)
	spiResponse := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"a": []string{"b"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"some": "response"}`)),
	}

	response, err := server.handleSpiResponse(origRequest, spiRequest,
		spiResponse, w)
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
	server := NewEndpointsServer()
	w := httptest.NewRecorder()
	origRequest := buildApiRequest(
		"/_ah/api/rpc",
		`[{"method": "foo.bar", "apiVersion": "X"}]`,
		nil,
	)
	assert.True(t, origRequest.IsBatch)
	assert.True(t, origRequest.IsRpc())
	origRequest.RequestId = "Z"
	spiRequest, err := origRequest.Copy()
	assert.NoError(t, err)
	spiResponse := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"a": []string{"b"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"some": "response"}`)),
	}

	response, err := server.handleSpiResponse(origRequest, spiRequest,
		spiResponse, w)
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
	server := NewEndpointsServer()
	w := httptest.NewRecorder()
	origRequest := buildApiRequest("/_ah/api/test", "{}", nil)
	spiRequest, err := origRequest.Copy()
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
	_, err = server.handleSpiResponse(origRequest, spiRequest,
		spiResponse, w)
	assert.NoError(t, err)
	header := http.Header{
		"a":              []string{"b"},
		"Content-Length": []string{fmt.Sprintf("%d", len(body))},
	}
	assertHttpMatchRecorder(t, w, 200, header, string(body))
}

// Verify the response is reformatted correctly.
func TestTransformRestResponse(t *testing.T) {
	server := NewEndpointsServer()
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

// Verify request_id inserted into the body, and body into body.result.
func TestTransformJsonRpcResponseBatch(t *testing.T) {
	server := NewEndpointsServer()
	origRequest := buildApiRequest(
		"/_ah/api/rpc",
		`[{"params": {"sample": "body"}, "id": "42"}]`,
		nil,
	)
	request, err := origRequest.Copy()
	assert.NoError(t, err)
	request.RequestId = "42"
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
	server := NewEndpointsServer()
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
