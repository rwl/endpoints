
package discovery

import (
	"testing"
	"encoding/json"
	"fmt"
	"net/http"
	"io/ioutil"
	"bytes"
	"github.com/stretchr/testify/mock"
	"net/http/httptest"
)

type MockDispatcher struct {
	mock.Mock
}

func (md *MockDispatcher) Do(request *http.Request) (*http.Response, error) {
	args := md.Mock.Called(request)
	return args.Get(0).(*http.Response), args.Error(1)
}

//type MockDispatcherSPI struct {
//	mock.Mock
//}
//
//func (md *MockDispatcherSPI) Do(request *http.Request) (*http.Response, error) {
//	args := md.Mock.Called(request)
//	return args.Get(0).(*http.Response), args.Error(1)
//}

type MockEndpointsDispatcher struct {
	mock.Mock
	EndpointsDispatcher
}

func newMockEndpointsDispatcher() *MockEndpointsDispatcher {
	return &MockEndpointsDispatcher{
		EndpointsDispatcher: set_up(),
	}
}

func (ed *MockEndpointsDispatcher) handle_spi_response(orig_request, spi_request *ApiRequest, response *http.Response, w http.ResponseWriter) (string, error) {
	args := ed.Mock.Called(orig_request, spi_request, response, w)
	return args.String(0), args.Error(1)
}

type MockEndpointsDispatcherSPI struct {
	mock.Mock
	EndpointsDispatcher
}

func newMockEndpointsDispatcherSPI() *MockEndpointsDispatcherSPI {
	return &MockEndpointsDispatcherSPI{
		EndpointsDispatcher: set_up(),
	}
}

func (ed *MockEndpointsDispatcher) call_spi(w http.ResponseWriter, orig_request *ApiRequest) (string, error) {
	args := ed.Mock.Called(w, orig_request)
	return args.String(0), args.Error(1)
}

type MockDiscoveryApiProxy struct {
	mock.Mock
	DiscoveryApiProxy
}

func (m *MockDiscoveryApiProxy) get_static_file(path string) (*http.Response, string, error) {
	args := m.Mock.Called(path)
	return args.Get(0).(*http.Response), args.String(1), args.Error(2)
}

func set_up() *EndpointsDispatcher {
	config_manager := NewApiConfigManager()
	mock_dispatcher := new(MockDispatcher)
	discovery_api := NewDiscoveryApiProxy()
	return NewEndpointsDispatcherConfig(mock_dispatcher, config_manager, discovery_api)
}

func prepare_dispatch(mock_dispatcher Dispatcher, config JsonObject) {
	// The dispatch call will make a call to get_api_configs, making a
	// dispatcher request. Set up that request.
	req, err := http.NewRequest("POST",
		_SERVER_SOURCE_IP + "/_ah/spi/BackendService.getApiConfigs",
		ioutil.NopCloser(bytes.NewBufferString("{}")))
	req.Header.Set("Content-Type", "application/json")

	response_body, err := json.Marshal(JsonObject{
		"items": []JsonObject{config},
	})
	header := new(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Content-Length", string(len(response_body)))
	resp := &http.Response{
		Body: ioutil.NopCloser(bytes.NewBuffer(response_body)),
		StatusCode: 200,
		Status: "200 OK",
		Header: header,
	}

	mock_dispatcher.On("Do", req).Return(resp, nil)
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
func assert_dispatch_to_spi(t *testing.T, request *ApiRequest, config *ApiDescriptor, spi_path string,
		expected_spi_body_json JsonObject) {
	server := newMockEndpointsDispatcher()
	prepare_dispatch(server.dispatcher, config)

	w := httptest.NewRecorder()

//	spi_headers := new(http.Header)
//	spi_headers.Set("Content-Type", "application/json")

	var spi_body_json JsonObject
	if expected_spi_body_json != nil {
		spi_body_json = expected_spi_body_json
	} else {
		spi_body_json = make(JsonObject)
	}

	// todo: compare a string of a JSON object to a JSON object
	spi_body, err := json.Marshal(spi_body_json)

	spi_request, err := http.NewRequest(
		"POST",
		request.RemoteAddr + spi_path,
		ioutil.NopCloser(bytes.NewBufferString(spi_body)),
	)
	spi_request.Header.Set("Content-Type", "application/json")

//	spi_response := dispatcher.ResponseTuple("200 OK", [], "Test")
	spi_response := &http.Response{
		StatusCode: 200,
		Status: "200 OK",
		Body: ioutil.NopCloser(bytes.NewBufferString("Test")),
	}

//	mock_dispatcher.add_request(
//		"POST", spi_path, spi_headers, JsonMatches(spi_body_json),
//		request.source_ip).AndReturn(spi_response)
	server.dispatcher.On("Do", spi_request).Return(spi_response, nil)

	server.On(
		"handle_spi_response",
		mock.AnythingOfType("*ApiRequest"),
		mock.AnythingOfType("*ApiRequest"),
		spi_response,
		w,
	).Return("Test", nil)
//	mox.StubOutWithMock(self.server, "handle_spi_response")
//	server.handle_spi_response(
//		mox.IsA(api_request.ApiRequest), mox.IsA(api_request.ApiRequest),
//		spi_response, self.start_response).AndReturn("Test")

	// Run the test.
//	mox.ReplayAll()
	response := server.dispatch(request, w)
//	mox.VerifyAll()
	server.Mock.AssertExpectations(t)

	if "Test" != response {
		t.Fail()
	}
}

func test_dispatch_invalid_path(t *testing.T) {
	server := set_up()
	config, _ := json.Marshal(JsonObject{
		"name": "guestbook_api",
		"version": "v1",
		"methods": {
			"guestbook.get": {
				"httpMethod": "GET",
				"path": "greetings/{gid}",
				"rosyMethod": "MyApi.greetings_get",
			},
		},
	})
	request := build_request("/_ah/api/foo", "", nil)
	prepare_dispatch(server.dispatcher, config)

	w := httptest.NewRecorder()

//	mox.ReplayAll()
	response := server.dispatch(request, w)
//	mox.VerifyAll()
	server.Mock.AssertExpectations(t)

	header := new(http.Header)
	header.Set("Content-Type", "text/plain")
	header.Set("Content-Length", "9")
	assert_http_match(t, response, 404, header, "Not Found")
}

func test_dispatch_invalid_enum(t *testing.T) {
	server := set_up()
	config, _ := json.Marshal(JsonObject{
		"name": "guestbook_api",
		"version": "v1",
		"methods": {
			"guestbook.get": {
				"httpMethod": "GET",
				"path": "greetings/{gid}",
				"rosyMethod": "MyApi.greetings_get",
				"request": {
					"body": "empty",
					"parameters": {
						"gid": {
							"enum": {
								"X": {
									"backendValue": "X",
								},
							},
							"type": "string",
						},
					},
				},
			},
		},
	})

	w := httptest.NewRecorder()

	request := build_request("/_ah/api/guestbook_api/v1/greetings/invalid_enum", "", nil)
	prepare_dispatch(server.dispatcher, config)
//	mox.ReplayAll()
	_ := server.dispatch(request, w)
//	mox.VerifyAll()
	server.Mock.AssertExpectations(t)

	t.Logf("Config %s", server.config_manager.configs)

	if w.Code != 400 {
		t.Fail()
	}
	body := w.Body.Bytes()
	var body_json JsonObject
	err := json.Unmarshal(body, &body_json)
	if err != nil {
		t.Fail()
	}
	error, ok := body_json["error"]
	if !ok {
		t.Fail()
	}
	error_json, ok := error.(JsonObject)
	if !ok {
		t.Fail()
	}
	errors, ok := error_json["errors"]
	if !ok {
		t.Fail()
	}
	errors_json, ok := errors.([]JsonObject)
	if !ok {
		t.Fail()
	}
	if 1 != len(errors_json) {
		t.Fail()
	}
	if "gid" != errors_json["location"] {
		t.Fail()
	}
	if "invalidParameter" != errors_json[0]["reason"] {
		t.Fail()
	}
}

// Check the error response if the SPI returns an error.
func test_dispatch_spi_error(t *testing.T) {
	server := newMockEndpointsDispatcherSPI()
	config, _ := json.Marshal(JsonObject{
		"name": "guestbook_api",
		"version": "v1",
		"methods": JsonObject{
			"guestbook.get": JsonObject{
				"httpMethod": "GET",
				"path": "greetings/{gid}",
				"rosyMethod": "MyApi.greetings_get",
			},
		},
	})
	request := build_request("/_ah/api/foo", "", nil)
	prepare_dispatch(server.dispatcher, config)

	w := httptest.NewRecorder()

//	mox.StubOutWithMock(server, "call_spi")
	// The application chose to throw a 404 error.
	response := &http.Response{
		Status: "404 Not Found",
		StatusCode: 404,
		Body: ioutil.NopCloser(
			bytes.NewBufferString(
				`{"state": "APPLICATION_ERROR", "error_message": "Test error"}`,
			),
		),
	}
//	response := dispatcher.ResponseTuple("404 Not Found", [],
//		(`{"state": "APPLICATION_ERROR", "error_message": "Test error"}`))
	server.On(
		"call_spi",
		request,
		mock.Anything,
	).Return(NewBackendError(response))
//	server.call_spi(request, mox.IgnoreArg()).AndRaise(NewBackendError(response))

//	mox.ReplayAll()
	response := server.dispatch(request, w)
//	self.mox.VerifyAll()
	server.Mock.AssertExpectations(t)

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
}`)
	header := new(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Content-Length", fmt.Sprintf("%d", len(expected_response)))
	assert_http_match(t, response, 404, header, expected_response)
}

// Test than an RPC call that returns an error is handled properly.
func test_dispatch_rpc_error(t *testing.T) {
	server := newMockEndpointsDispatcherSPI()
	config, _ := json.Marshal(JsonObject{
		"name": "guestbook_api",
		"version": "v1",
		"methods": {
			"guestbook.get": {
				"httpMethod": "GET",
				"path": "greetings/{gid}",
				"rosyMethod": "MyApi.greetings_get",
			},
		},
	})
	request := build_request(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X", "id": "gapiRpc"}`,
		nil,
	)
	prepare_dispatch(server.dispatcher, config)

	w := httptest.NewRecorder()

//	mox.StubOutWithMock(server, "call_spi")
	// The application chose to throw a 404 error.
	response := &http.Response{
		Status: "404 Not Found",
		StatusCode: 404,
		Body: ioutil.NopCloser(
		bytes.NewBufferString(
		`{"state": "APPLICATION_ERROR", "error_message": "Test error"}`,
		),
		),
	}
//	response = dispatcher.ResponseTuple("404 Not Found", [],
//	(`{"state": "APPLICATION_ERROR","
//	  "error_message": "Test error"}`))
	server.On(
		"call_spi",
		request,
		mock.Anything,
	).Return(NewBackendError(response))
//	server.call_spi(request, mox.IgnoreArg()).AndRaise(NewBackendError(response))

//	mox.ReplayAll()
	response := server.dispatch(request, w)
//	mox.VerifyAll()
	server.Mock.AssertExpectations(t)

	expected_response := JsonObject{
		"error": {
			"code": 404,
			"message": "Test error",
			"data": []JsonObject{
				JsonObject{
					"domain": "global",
					"reason": "notFound",
					"message": "Test error",
				},
			},
		},
		"id": "gapiRpc",
	}
	if w.Code != 200 {
		t.Fail()
	}
	var response_json interface{}
	err := json.Unmarshal(response, &response_json)
	if err != nil {
		t.Fail()
	}
	if expected_response != response_json {
		t.Fail()
	}
}

func test_dispatch_json_rpc(t *testing.T) {
	config, _ := json.Marshal(JsonObject{
		"name": "guestbook_api",
		"version": "X",
		"methods": {
			"foo.bar": {
				"httpMethod": "GET",
				"path": "greetings/{gid}",
				"rosyMethod": "baz.bim",
			},
		},
	})
	request := build_request(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X"}`,
		nil,
	)
	assert_dispatch_to_spi(t, request, config, "/_ah/spi/baz.bim", nil)
}

func test_dispatch_rest(t *testing.T) {
	config, _ := json.Marshal(JsonObject{
		"name": "myapi",
		"version": "v1",
		"methods": {
			"bar": {
				"httpMethod": "GET",
				"path": "foo/{id}",
				"rosyMethod": "baz.bim",
			},
		},
	})
	request := build_request("/_ah/api/myapi/v1/foo/testId", "", nil)
	assert_dispatch_to_spi(t, request, config, "/_ah/spi/baz.bim",
		JsonObject{"id": "testId"})
}

func test_explorer_redirect(t *testing.T) {
	server := set_up()
	w := httptest.NewRecorder()
	request := build_request("/_ah/api/explorer", "", nil)
	response := server.dispatch(request, w)
	header := new(http.Header)
	header.Set("Content-Length", "0")
	header.Set("Location", "https://developers.google.com/apis-explorer/?base=http://localhost:42/_ah/api")
	assert_http_match(t, response, 302, header, "")
}

func test_static_existing_file(t *testing.T) {
	relative_url := "/_ah/api/static/proxy.html"

	w := httptest.NewRecorder()

	// Set up mocks for the call to DiscoveryApiProxy.get_static_file.
	discovery_api := &MockDiscoveryApiProxy{}
	server := NewEndpointsDispatcherConfig(
		&http.Client{},
		NewApiConfigManager(),
		discovery_api,
	)
//	mox.StubOutWithMock(discovery_api_proxy, "DiscoveryApiProxy")
//	DiscoveryApiProxy().AndReturn(discovery_api)
	/*static_response = mox.CreateMock(httplib.HTTPResponse)
	static_response.status = 200
	static_response.reason = "OK"
	static_response.getheader("Content-Type").AndReturn("test/type")*/
	test_body := "test body"
//	get_static_file(relative_url).AndReturn(static_response, test_body)
	discovery_api.On(
		"get_static_file",
		relative_url,
	).Return(mock.Anything/*static_response*/, test_body, nil)

	// Make sure the dispatch works as expected.
	request := build_request(relative_url, "", nil)
//	mox.ReplayAll()
	response := server.dispatch(request, w)
//	mox.VerifyAll()
	server.Mock.AssertExpectations(t)

	header := new(Header)
	header.Set("Content-Length", fmt.Sprintf("%d", len(test_body)))
	header.Set("Content-Type", "test/type")
	assert_http_match(t, response, 200, header, test_body)
}

func test_static_non_existing_file(t *testing.T) {
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
	request = build_request(relative_url, "", nil)
	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
	mox.VerifyAll()

	response := "".join(response)
	header := new(Header)
	header.Set("Content-Length", fmt.Sprintf("%d", len(test_body)))
	header.Set("Content-Type", "test/type")
	assert_http_match(t, response, 404, header, test_body)
}

func test_handle_non_json_spi_response(t *testing.T) {
	orig_request := build_request("/_ah/api/fake/path", "", nil)
	spi_request := orig_request.copy()
	spi_response = dispatcher.ResponseTuple(
		200, [("Content-type", "text/plain")], "This is an invalid response.")
	response = server.handle_spi_response(orig_request, spi_request,
		spi_response, self.start_response)
	error_json = JsonObject{
		"error": {
			"message": "Non-JSON reply: This is an invalid response.",
		},
	}
	body_bytes, _ = json.Marshal(error_json)
	body = string(body_bytes)
	header := Header{
		"Content-Type", "application/json",
		"Content-Length", fmt.Sprintf("%d", len(body)),
	}
	assert_http_match(t, response, 500, header, body)
}

// Verify Lily protocol correctly uses python method name.
//
// This test verifies the fix to http://b/7189819
func test_lily_uses_python_method_name(t *testing.T) {
	config, _ := json.Marshal(JsonObject{
		"name": "guestbook_api",
		"version": "X",
		"methods": {
			"author.greeting.info.get": {
				"httpMethod": "GET",
				"path": "authors/{aid}/greetings/{gid}/infos/{iid}",
				"rosyMethod": "InfoService.get",
			},
		},
	})
	request := build_request(
		"/_ah/api/rpc",
		`{"method": "author.greeting.info.get", "apiVersion": "X"}`,
		nil,
	)
	assert_dispatch_to_spi(t, request, config, "/_ah/spi/InfoService.get", JsonObject{})
}

// Verify headers transformed, JsonRpc response transformed, written.
func test_handle_spi_response_json_rpc(t *testing.T) {
	orig_request := build_request(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X"}`,
		nil
	)
	if !orig_request.is_rpc() {
		t.Fail()
	}
	orig_request.request_id = "Z"
	spi_request = orig_request.copy()
	spi_response = dispatcher.ResponseTuple("200 OK", [("a", "b")],
		`{"some": "response"}`)

	response = server.handle_spi_response(orig_request, spi_request,
		spi_response, self.start_response)
	response = "".join(response)  // Merge response iterator into single body.

	if self.response_status != "200 OK" {
		t.Fail()
	}
	if _, ok := self.response_headers["a"]; !ok {
		t.Fail()
	}
	if _, ok := self.response_headers["b"]; !ok {
		t.Fail()
	}
	expected_response := JsonObject{
		"id": "Z",
		"result": JsonObject{"some": "response"},
	}
	var response_json interface{}
	err := json.Unmarshal(response, &response_json)
	if err != nil {
		t.Fail()
	}
	if expected_response != response_json {
		t.Fail()
	}
}

// Verify that batch requests have an appropriate batch response.
func test_handle_spi_response_batch_json_rpc(t *testing.T) {
	orig_request = build_request(
		"/_ah/api/rpc",
		`[{"method": "foo.bar", "apiVersion": "X"}]`,
		nil,
	)
	if !orig_request.is_batch() {
		t.Fail()
	}
	if !orig_request.is_rpc() {
		t.Fail()
	}
	orig_request.request_id = "Z"
	spi_request = orig_request.copy()
	spi_response = dispatcher.ResponseTuple("200 OK", [("a", "b")],
		`{"some": "response"}`)

	response = server.handle_spi_response(orig_request, spi_request,
		spi_response, self.start_response)
	response = "".join(response)  // Merge response iterator into single body.

	if self.response_status != "200 OK" {
		t.Fail()
	}
	if _, ok := self.response_headers["a"]; !ok {
		t.Fail()
	}
	if _, ok := self.response_headers["b"]; !ok {
		t.Fail()
	}
	expected_response := JsonObject{
		"id": "Z",
		"result": JsonObject{"some": "response"},
	}
	var response_json interface{}
	err := json.Unmarshal(response, &response_json)
	if err != nil {
		t.Fail()
	}
	if expected_response != response_json {
		t.Fail()
	}
}

func test_handle_spi_response_rest(t *testing.T) {
	orig_request := build_request("/_ah/api/test", "{}", nil)
	spi_request := orig_request.copy()
	body, _ := json.MarshalIndent(JsonObject{"some": "response"}, "", " ")
	spi_response = dispatcher.ResponseTuple("200 OK", [("a", "b")], body)
	response = server.handle_spi_response(orig_request, spi_request,
		spi_response, self.start_response)
	header := http.Header{
		"a": "b",
		"Content-Length": fmt.Sprintf("%d" % len(body)),
	}
	assert_http_match(t, response, 200, header, body)
}

// Verify the response is reformatted correctly.
func test_transform_rest_response(t *testing.T) {
	orig_response = `{"sample": "test", "value1": {"value2": 2}}`
	expected_response = (`{
 "sample": "test",
 "value1": {
  "value2": 2
 }
}`)
	if expected_response != server.transform_rest_response(orig_response) {
		t.Fail()
	}
}

// Verify request_id inserted into the body, and body into body.result.
func test_transform_json_rpc_response_batch(t *testing.T) {
	orig_request := build_request(
		"/_ah/api/rpc",
		`[{"params": {"sample": "body"}, "id": "42"}]`,
		nil
	)
	request = orig_request.copy()
	request.request_id = "42"
	orig_response = `{"sample": "body"}`
	response = server.transform_jsonrpc_response(request, orig_response)
	expected_response := []JsonObject{
		JsonObject{
			"result": JsonObject{"sample": "body"},
			"id": "42",
		},
	}
	var response_json interface{}
	err := json.Unmarshal(response, &response_json)
	if err != nil {
		t.Fail()
	}
	if expected_response != response_json {
		t.Fail()
	}
}

func test_lookup_rpc_method_no_body(t *testing.T) {
	orig_request := build_request("/_ah/api/rpc", "", nil)
	if server.lookup_rpc_method(orig_request) != nil {
		t.Fail()
	}
}

func test_lookup_rpc_method(t *testing.T) {
	mox.StubOutWithMock(server.config_manager, "lookup_rpc_method")
	server.config_manager.lookup_rpc_method("foo", "v1").AndReturn("bar")

	mox.ReplayAll()
	orig_request := build_request(
		"/_ah/api/rpc",
		`{"method": "foo", "apiVersion": "v1"}`,
		nil,
	)
	if "bar" != server.lookup_rpc_method(orig_request) {
		t.Fail()
	}
	mox.VerifyAll()
}

func test_verify_response(t *testing.T) {
	response := dispatcher.ResponseTuple("200", [("Content-Type", "a")], "")
	// Expected response
	if !server.verify_response(response, 200, "a") {
		t.Fail()
	}
	// Any content type accepted
	if !server.verify_response(response, 200, None) {
		t.Fail()
	}
	// Status code mismatch
	if server.verify_response(response, 400, "a") {
		t.Fail()
	}
	// Content type mismatch
	if server.verify_response(response, 200, "b") {
		t.Fail()
	}

	response := dispatcher.ResponseTuple("200", [("Content-Length", "10")], "")
	// Any content type accepted
	if !server.verify_response(response, 200, None) {
		t.Fail()
	}
	// Specified content type not matched
	if server.verify_response(response, 200, "a") {
		t.Fail()
	}
}
