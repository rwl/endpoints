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

func prepare_dispatch(t *testing.T, config *endpoints.ApiDescriptor) *httptest.Server {
	// The dispatch call will make a call to get_api_configs, making a
	// dispatcher request. Set up that request.
	/*req, _ := http.NewRequest("POST",
		_SERVER_SOURCE_IP+"/_ah/spi/BackendService.getApiConfigs",
		ioutil.NopCloser(bytes.NewBufferString("{}")))
	req.Header.Set("Content-Type", "application/json")*/

	config_bytes, err := json.Marshal(config)
	if err != nil {
		panic("Invalid config")
	}
	response_body, _ := json.Marshal(map[string]interface{}{
		"items": []string{string(config_bytes)},
	})
	/*header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Content-Length", string(len(response_body)))
	resp := &http.Response{
		Body:       ioutil.NopCloser(bytes.NewBuffer(response_body)),
		StatusCode: 200,
		Status:     "200 OK",
		Header:     header,
	}*/

	//mock_dispatcher.On("Do", req).Return(resp, nil)

	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST")
		//assert.Equal(t, r.RemoteAddr, _SERVER_SOURCE_IP)
		assert.Equal(t, r.URL.Path, "/_ah/spi/BackendService.getApiConfigs")
		body, _ := ioutil.ReadAll(r.Body)
		assert.Equal(t, string(body), "{}")
		assert.Equal(t, r.Header.Get("Content-Type"), "application/json")

		w.Header().Set("Content-Type", "application/json")
		//w.Header().Set("Content-Length", string(len(response_body)))
		fmt.Fprintln(w, string(response_body))
	})
	ts := httptest.NewServer(hf)
	//ts := httptest.NewServer(nil)
	//http.DefaultServeMux.HandleFunc("/_ah/spi/BackendService.getApiConfigs", hf)

	//defer ts.Close()
	//_SERVER_SOURCE_IP = ts.URL
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
func assert_dispatch_to_spi(t *testing.T, request *ApiRequest, config *endpoints.ApiDescriptor, spi_path string,
	expected_spi_body_json map[string]interface{}) {
	server := newMockEndpointsServer()
	ts := prepare_dispatch(t, config)
	server.URL = ts.URL
	defer ts.Close()

	w := httptest.NewRecorder()

	//spi_headers := make(http.Header)
	//spi_headers.Set("Content-Type", "application/json")

	var spi_body_json map[string]interface{}
	if expected_spi_body_json != nil {
		spi_body_json = expected_spi_body_json
	} else {
		spi_body_json = make(map[string]interface{})
	}

	// todo: compare a string of a JSON object to a JSON object
	spi_body, err := json.Marshal(spi_body_json)
	assert.NoError(t, err)

	/*spi_request, err := http.NewRequest(
		"POST",
		request.RemoteAddr+spi_path,
		ioutil.NopCloser(bytes.NewBuffer(spi_body)),
	)
	assert.NoError(t, err)
	spi_request.Header.Set("Content-Type", "application/json")*/

	//spi_response := dispatcher.ResponseTuple("200 OK", [], "Test")
	// fixme: build a valid response
	//	spi_response := &http.Response{
	//		StatusCode: 200,
	//		Status:     "200 OK",
	//		Body:       ioutil.NopCloser(bytes.NewBufferString("Test")),
	//	}

	//mock_dispatcher.add_request(
	//	"POST", spi_path, spi_headers, JsonMatches(spi_body_json),
	//	request.source_ip).AndReturn(spi_response)
	//dispatcher.On("Do", spi_request).Return(spi_response, nil)
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST")
		assert.Equal(t, r.URL.Path, spi_path)
		body, _ := ioutil.ReadAll(r.Body)
		assert.Equal(t, string(body), string(spi_body))
		assert.Equal(t, r.Header.Get("Content-Type"), "application/json")

		fmt.Fprint(w, "Test")
	}))
	defer ts2.Close()
	orig := _SPI_ROOT_FORMAT
	_SPI_ROOT_FORMAT = ts2.URL + _SPI_ROOT_FORMAT
	defer func() {
		_SPI_ROOT_FORMAT = orig
	}()

	server.On(
		"handle_spi_response",
		mock.Anything, //OfType("*ApiRequest"),
		mock.Anything, //OfType("*ApiRequest"),
		mock.Anything, //spi_response,
		w,
	).Return("Test", nil)
	//mox.StubOutWithMock(self.server, "handle_spi_response")
	//server.handle_spi_response(
	//	mox.IsA(api_request.ApiRequest), mox.IsA(api_request.ApiRequest),
	//	spi_response, self.start_response).AndReturn("Test")

	// Run the test.
	//mox.ReplayAll()
	response := server.dispatch(w, request)
	//mox.VerifyAll()
	server.Mock.AssertExpectations(t)

	assert.Equal(t, "Test", response)
}

func Test_dispatch_invalid_path(t *testing.T) {
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
	request := build_request("/_ah/api/foo", "", nil)
	ts := prepare_dispatch(t, config)
	server.URL = ts.URL
	defer ts.Close()

	w := httptest.NewRecorder()

	//mox.ReplayAll()
	server.HandleHttp(nil)
	http.DefaultServeMux.ServeHTTP(w, request)
	//mox.VerifyAll()
	//	dispatcher.Mock.AssertExpectations(t)

	header := make(http.Header)
	header.Set("Content-Type", "text/plain")
	header.Set("Content-Length", "9")
	assert_http_match_recorder(t, w, 404, header, "Not Found")
}

func Test_dispatch_invalid_enum(t *testing.T) {
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

	request := build_request("/_ah/api/guestbook_api/v1/greetings/invalid_enum", "", nil)
	ts := prepare_dispatch(t, config)
	server.URL = ts.URL
	defer ts.Close()

	//server.HandleHttp(nil)
	//http.DefaultServeMux.ServeHTTP(w, request)
	server.ServeHTTP(w, request)

	//t.Logf("Config %s", server.config_manager.configs)

	assert.Equal(t, w.Code, 400)
	body := w.Body.Bytes()
	var body_json map[string]interface{}
	err := json.Unmarshal(body, &body_json)
	assert.NoError(t, err, "Body: %s", string(body))
	error, ok := body_json["error"]
	assert.True(t, ok)
	error_json, ok := error.(map[string]interface{})
	assert.True(t, ok)
	errors, ok := error_json["errors"]
	assert.True(t, ok)
	errors_json, ok := errors.([]interface{})
	assert.True(t, ok)
	errors_json0, ok := errors_json[0].(map[string]interface{})
	assert.True(t, ok)
	ok = assert.Equal(t, 1, len(errors_json))
	if ok {
		assert.Equal(t, "gid", errors_json0["location"])
		assert.Equal(t, "invalidParameter", errors_json0["reason"])
	}
}

// Check the error response if the SPI returns an error.
func Test_dispatch_spi_error(t *testing.T) {
	server := newMockEndpointsServerSPI()
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
	request := build_api_request("/_ah/api/foo", "", nil)
	ts := prepare_dispatch(t, config)
	server.URL = ts.URL
	defer ts.Close()

	w := httptest.NewRecorder()

	//mox.StubOutWithMock(server, "call_spi")
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
	//response := dispatcher.ResponseTuple("404 Not Found", [],
	//	(`{"state": "APPLICATION_ERROR", "error_message": "Test error"}`))
	server.On(
		"call_spi",
		mock.Anything,
		request,
	).Return("", NewBackendError(response))
	//server.call_spi(request, mox.IgnoreArg()).AndRaise(NewBackendError(response))

	//mox.ReplayAll()
	server.dispatch(w, request)
	//self.mox.VerifyAll()
	server.Mock.AssertExpectations(t)
	//	dispatcher.Mock.AssertExpectations(t)

	expected_response := `{
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
	header.Set("Content-Length", fmt.Sprintf("%d", len(expected_response)))
	assert_http_match_recorder(t, w, 404, header, expected_response)
}

// Test than an RPC call that returns an error is handled properly.
func Test_dispatch_rpc_error(t *testing.T) {
	server := newMockEndpointsServerSPI()
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
	request := build_api_request(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X", "id": "gapiRpc"}`,
		nil,
	)
	ts := prepare_dispatch(t, config)
	server.URL = ts.URL
	defer ts.Close()

	w := httptest.NewRecorder()

	//mox.StubOutWithMock(server, "call_spi")
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
	//response = dispatcher.ResponseTuple("404 Not Found", [],
	//(`{"state": "APPLICATION_ERROR","
	//  "error_message": "Test error"}`))
	server.On(
		"call_spi",
		mock.Anything,
		request,
	).Return("", NewBackendError(response))
	//server.call_spi(request, mox.IgnoreArg()).AndRaise(NewBackendError(response))

	//mox.ReplayAll()
	response_body := server.dispatch(w, request)
	//mox.VerifyAll()
	server.Mock.AssertExpectations(t)

	expected_response := map[string]interface{}{
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
	var response_json interface{}
	err := json.Unmarshal([]byte(response_body), &response_json)
	assert.NoError(t, err)
	assert.Equal(t, expected_response, response_json)
}

func Test_dispatch_json_rpc(t *testing.T) {
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
	request := build_api_request(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X"}`,
		nil,
	)
	assert_dispatch_to_spi(t, request, config, "/_ah/spi/baz.bim", nil)
}

func Test_dispatch_rest(t *testing.T) {
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
	request := build_api_request("/_ah/api/myapi/v1/foo/testId", "", nil)
	assert_dispatch_to_spi(t, request, config, "/_ah/spi/baz.bim",
		map[string]interface{}{"id": "testId"})
}

func Test_explorer_redirect(t *testing.T) {
	server := NewEndpointsServer()
	w := httptest.NewRecorder()
	request := build_request("/_ah/api/explorer", "", nil)
	//server.dispatch(w, request)
	//server.HandleHttp(nil)
	//http.DefaultServeMux.ServeHTTP(w, request)
	server.HandleApiExplorerRequest(w, request)
	header := make(http.Header)
	//	header.Set("Content-Length", "0")
	location := "https://developers.google.com/apis-explorer/?base=http://localhost:42/_ah/api"
	header.Set("Location", location)
	body := fmt.Sprintf(`<a href="%s">Found</a>.

`, location) // todo: check if anchor is a valid response body
	assert_http_match_recorder(t, w, 302, header, body)
}

//func Test_static_existing_file(t *testing.T) {
//	relative_url := "/_ah/api/static/proxy.html"
//
//	w := httptest.NewRecorder()
//
//	// Set up mocks for the call to DiscoveryApiProxy.get_static_file.
//	discovery_api := &MockDiscoveryApiProxy{}
//	server := NewEndpointsServerConfig(
//		&http.Client{},
//		NewApiConfigManager(),
//		discovery_api,
//	)
////	mox.StubOutWithMock(discovery_api_proxy, "DiscoveryApiProxy")
////	DiscoveryApiProxy().AndReturn(discovery_api)
//	/*static_response = mox.CreateMock(httplib.HTTPResponse)
//	static_response.status = 200
//	static_response.reason = "OK"
//	static_response.getheader("Content-Type").AndReturn("test/type")*/
//	test_body := "test body"
////	get_static_file(relative_url).AndReturn(static_response, test_body)
//	discovery_api.On(
//		"get_static_file",
//		relative_url,
//	).Return(mock.Anything/*static_response*/, test_body, nil)
//
//	// Make sure the dispatch works as expected.
//	request := build_api_request(relative_url, "", nil)
////	mox.ReplayAll()
//	response := server.dispatch(request, w)
////	mox.VerifyAll()
//	server.Mock.AssertExpectations(t)
//
//	header := new(Header)
//	header.Set("Content-Length", fmt.Sprintf("%d", len(test_body)))
//	header.Set("Content-Type", "test/type")
//	assert_http_match(t, response, 200, header, test_body)
//}

/*func Test_static_non_existing_file(t *testing.T) {
	relative_url := "/_ah/api/static/blah.html"

	// Set up mocks for the call to DiscoveryApiProxy.get_static_file.
	discovery_api = self.mox.CreateMock(DiscoveryApiProxy)
	self.mox.StubOutWithMock(discovery_api_proxy, "DiscoveryApiProxy")
	discovery_api_proxy.DiscoveryApiProxy().AndReturn(discovery_api)
	static_response = mox.CreateMock(httplib.HTTPResponse)
	static_response.status = 404
	static_response.reason = "Not Found"
	static_response.getheaders().AndReturn(map[string]string{"Content-Type": "test/type"})
	test_body = "No Body"
	get_static_file(relative_url).AndReturn(static_response, test_body)

	// Make sure the dispatch works as expected.
	request = build_api_request(relative_url, "", nil)
	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
	mox.VerifyAll()

	response := "".join(response)
	header := new(Header)
	header.Set("Content-Length", fmt.Sprintf("%d", len(test_body)))
	header.Set("Content-Type", "test/type")
	assert_http_match(t, response, 404, header, test_body)
}*/

func Test_handle_non_json_spi_response(t *testing.T) {
	server := NewEndpointsServer()
	w := httptest.NewRecorder()
	orig_request := build_api_request("/_ah/api/fake/path", "", nil)
	spi_request, err := orig_request.copy()
	assert.NoError(t, err)
	header := make(http.Header)
	header.Set("Content-type", "text/plain")
	spi_response := &http.Response{
		Header:     header,
		Body:       ioutil.NopCloser(bytes.NewBufferString("This is an invalid response.")),
		StatusCode: 200,
		Status:     "200 OK",
	}
	server.handle_spi_response(orig_request, spi_request, spi_response, w)
	error_json := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Non-JSON reply: This is an invalid response.",
		},
	}
	body_bytes, _ := json.Marshal(error_json)
	body := string(body_bytes)
	expected_header := http.Header{
		"Content-Type":   []string{"application/json"},
		"Content-Length": []string{fmt.Sprintf("%d", len(body))},
	}
	assert_http_match_recorder(t, w, 500, expected_header, body)
}

// Verify Lily protocol correctly uses python method name.
//
// This test verifies the fix to http://b/7189819
func Test_lily_uses_python_method_name(t *testing.T) {
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
	request := build_api_request(
		"/_ah/api/rpc",
		`{"method": "author.greeting.info.get", "apiVersion": "X"}`,
		nil,
	)
	assert_dispatch_to_spi(t, request, config, "/_ah/spi/InfoService.get", map[string]interface{}{})
}

// Verify headers transformed, JsonRpc response transformed, written.
func Test_handle_spi_response_json_rpc(t *testing.T) {
	server := NewEndpointsServer()
	w := httptest.NewRecorder()
	orig_request := build_api_request(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X"}`,
		nil,
	)
	assert.True(t, orig_request.is_rpc())
	orig_request.request_id = "Z"
	spi_request, err := orig_request.copy()
	assert.NoError(t, err)
	spi_response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"a": []string{"b"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"some": "response"}`)),
	}

	response, err := server.handle_spi_response(orig_request, spi_request,
		spi_response, w)
	//response = "".join(response)  // Merge response iterator into single body.
	assert.NoError(t, err)

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Header()["a"][0], "b")
	expected_response := map[string]interface{}{
		"id":     "Z",
		"result": map[string]interface{}{"some": "response"},
	}
	var response_json interface{}
	err = json.Unmarshal([]byte(response), &response_json)
	assert.NoError(t, err)
	assert.Equal(t, expected_response, response_json)
}

// Verify that batch requests have an appropriate batch response.
func Test_handle_spi_response_batch_json_rpc(t *testing.T) {
	server := NewEndpointsServer()
	w := httptest.NewRecorder()
	orig_request := build_api_request(
		"/_ah/api/rpc",
		`[{"method": "foo.bar", "apiVersion": "X"}]`,
		nil,
	)
	assert.True(t, orig_request.is_batch)
	assert.True(t, orig_request.is_rpc())
	orig_request.request_id = "Z"
	spi_request, err := orig_request.copy()
	assert.NoError(t, err)
	spi_response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"a": []string{"b"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"some": "response"}`)),
	}

	response, err := server.handle_spi_response(orig_request, spi_request,
		spi_response, w)
	//response = "".join(response)  // Merge response iterator into single body.
	assert.NoError(t, err)

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Header()["a"][0], "b")
	expected_response := []interface{}{
		map[string]interface{}{
			"id":     "Z",
			"result": map[string]interface{}{"some": "response"},
		},
	}
	var response_json interface{}
	err = json.Unmarshal([]byte(response), &response_json)
	assert.NoError(t, err)
	assert.Equal(t, expected_response, response_json)
}

func Test_handle_spi_response_rest(t *testing.T) {
	server := NewEndpointsServer()
	w := httptest.NewRecorder()
	orig_request := build_api_request("/_ah/api/test", "{}", nil)
	spi_request, err := orig_request.copy()
	assert.NoError(t, err)
	body, _ := json.MarshalIndent(map[string]interface{}{"some": "response"}, "", "  ")
	spi_response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"a": []string{"b"}},
		Body:       ioutil.NopCloser(bytes.NewBuffer(body)),
	}
	_, err = server.handle_spi_response(orig_request, spi_request,
		spi_response, w)
	assert.NoError(t, err)
	header := http.Header{
		"a":              []string{"b"},
		"Content-Length": []string{fmt.Sprintf("%d", len(body))},
	}
	assert_http_match_recorder(t, w, 200, header, string(body))
}

// Verify the response is reformatted correctly.
func Test_transform_rest_response(t *testing.T) {
	server := NewEndpointsServer()
	orig_response := `{"sample": "test", "value1": {"value2": 2}}`
	expected_response := `{
  "sample": "test",
  "value1": {
    "value2": 2
  }
}`
	response, err := server.transform_rest_response(orig_response)
	assert.NoError(t, err)
	assert.Equal(t, expected_response, response)
}

// Verify request_id inserted into the body, and body into body.result.
func Test_transform_json_rpc_response_batch(t *testing.T) {
	server := NewEndpointsServer()
	orig_request := build_api_request(
		"/_ah/api/rpc",
		`[{"params": {"sample": "body"}, "id": "42"}]`,
		nil,
	)
	request, err := orig_request.copy()
	assert.NoError(t, err)
	request.request_id = "42"
	orig_response := `{"sample": "body"}`
	response, err := server.transform_jsonrpc_response(request, orig_response)
	assert.NoError(t, err)
	expected_response := []map[string]interface{}{
		map[string]interface{}{
			"result": map[string]interface{}{"sample": "body"},
			"id":     "42",
		},
	}
	var response_json []map[string]interface{}
	err = json.Unmarshal([]byte(response), &response_json)
	assert.NoError(t, err)
	assert.Equal(t, expected_response, response_json)
}

func Test_lookup_rpc_method_no_body(t *testing.T) {
	server := NewEndpointsServer()
	orig_request := build_api_request("/_ah/api/rpc", "", nil)
	assert.Nil(t, server.lookup_rpc_method(orig_request))
}

/*func Test_lookup_rpc_method(t *testing.T) {
	mox.StubOutWithMock(server.config_manager, "lookup_rpc_method")
	server.config_manager.lookup_rpc_method("foo", "v1").AndReturn("bar")

	mox.ReplayAll()
	orig_request := build_api_request(
		"/_ah/api/rpc",
		`{"method": "foo", "apiVersion": "v1"}`,
		nil,
	)
	if "bar" != server.lookup_rpc_method(orig_request) {
		t.Fail()
	}
	mox.VerifyAll()
}*/

func Test_verify_response(t *testing.T) {
	response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"a"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString("")),
	}
	// Expected response
	assert.NoError(t, verify_response(response, 200, "a"))
	// Any content type accepted
	assert.NoError(t, verify_response(response, 200, ""))
	// Status code mismatch
	assert.Error(t, verify_response(response, 400, "a"))
	// Content type mismatch
	assert.Error(t, verify_response(response, 200, "b"))

	response = &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"Content-Length": []string{"10"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString("")),
	}
	// Any content type accepted
	assert.NoError(t, verify_response(response, 200, ""))
	// Specified content type not matched
	assert.Error(t, verify_response(response, 200, "a"))
}
