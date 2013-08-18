
package endpoints

import (
	"testing"
	"encoding/json"
	"fmt"
	"net/http"
	"io/ioutil"
	"bytes"
)

var _config JsonObject

type MockDispatcher struct {}

func (md *MockDispatcher) Do(request *http.Request) (*http.Response, error) {
	if request.Method == "POST" &&
		request.URL.Path == "/_ah/spi/BackendService.getApiConfigs" &&
		request.Header.Get("Content-Type") == "application/json" {
		response_body, _ := json.Marshal(JsonObject{"items": []JsonObject{_config}})
		header := new(http.Header)
		header.Set("Content-Type", "application/json")
		header.Set("Content-Length", string(len(response_body)))
		return &http.Response{
			Status: "200 OK",
			StatusCode: 200,
			Header: header,
			Body: response_body,
		}
	}
	return nil, fmt.Errorf("Unexpected request: %v", request)
}

func set_up() *EndpointsDispatcher {
	config_manager := NewApiConfigManager()
	mock_dispatcher := new(MockDispatcher)
	return NewEndpointsDispatcherConfig(mock_dispatcher, config_manager)
}

/*func prepare_dispatch(config) {
	// The dispatch call will make a call to get_api_configs, making a
	// dispatcher request.  Set up that request.
	request_method = "POST"
	request_path = "/_ah/spi/BackendService.getApiConfigs"
	request_headers = [("Content-Type", "application/json")]
	request_body = "{}"
	response_body = json.dumps({"items": [config]})
	mock_dispatcher.add_request(
		request_method, request_path, request_headers, request_body,
		_SERVER_SOURCE_IP).AndReturn(
			dispatcher.ResponseTuple("200 OK",
									[("Content-Type", "application/json"),
									("Content-Length", string(len(response_body)))],
									response_body))
}*/

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
//	prepare_dispatch(config)
	_config = config

	spi_headers := [("Content-Type", "application/json")]
	spi_body_json := expected_spi_body_json //or {}
	spi_response := dispatcher.ResponseTuple("200 OK", [], "Test")
	mock_dispatcher.add_request(
		"POST", spi_path, spi_headers, JsonMatches(spi_body_json),
		request.source_ip).AndReturn(spi_response)

	mox.StubOutWithMock(self.server, "handle_spi_response")
	server.handle_spi_response(
		mox.IsA(api_request.ApiRequest), mox.IsA(api_request.ApiRequest),
		spi_response, self.start_response).AndReturn("Test")

	// Run the test.
	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
	mox.VerifyAll()

	if "Test" != response {
		t.Fail()
	}
}

func test_dispatch_invalid_path(t *testing.T) {
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
	prepare_dispatch(config)
	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
	mox.VerifyAll()

	header := new(http.Header)
	header.Set("Content-Type", "text/plain")
	header.Set("Content-Length", "9")
	assert_http_match(t, response, 404, header, "Not Found")
}

func test_dispatch_invalid_enum(t *testing.T) {
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

	request := build_request("/_ah/api/guestbook_api/v1/greetings/invalid_enum", "", nil)
	prepare_dispatch(config)
	mox.ReplayAll()
	response := server.dispatch(request, self.start_response)
	mox.VerifyAll()

	t.Logf("Config %s", server.config_manager.configs)

	if self.response_status != "400 Bad Request" {
		t.Fail()
	}
	body := "".join(response)
	body_json := json.loads(body)
	if 1 != len(body_json["error"]["errors"]) {
		t.Fail()
	}
	if "gid" != body_json["error"]["errors"][0]["location"] {
		t.Fail()
	}
	if "invalidParameter" != body_json["error"]["errors"][0]["reason"] {
		t.Fail()
	}
}

// Check the error response if the SPI returns an error.
func test_dispatch_spi_error(t *testing.T) {
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
	prepare_dispatch(config)
	mox.StubOutWithMock(self.server, "call_spi")
	// The application chose to throw a 404 error.
	response := dispatcher.ResponseTuple("404 Not Found", [],
		(`{"state": "APPLICATION_ERROR", "error_message": "Test error"}`))
	server.call_spi(request, mox.IgnoreArg()).AndRaise(NewBackendError(response))

	mox.ReplayAll()
	response := server.dispatch(request, self.start_response)
	self.mox.VerifyAll()

	expected_response := `
		 {\n
		  "error": {\n
		   "code": 404, \n
		   "errors": [\n
		   {\n
		    "domain": "global", \n
		    "message": "Test error", \n
		    "reason": "notFound"\n
		   }\n
		  ], \n
		  "message": "Test error"\n
		 }\n
		}`)
	response := "".join(response)
	header := new(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Content-Length", fmt.Sprintf("%d", len(expected_response)))
	assert_http_match(response, 404, header, expected_response, "")
}

// Test than an RPC call that returns an error is handled properly.
func test_dispatch_rpc_error(t *testing.T) {
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
	prepare_dispatch(config)
	mox.StubOutWithMock(server, "call_spi")
	// The application chose to throw a 404 error.
	response = dispatcher.ResponseTuple("404 Not Found", [],
	(`{"state": "APPLICATION_ERROR","
	  "error_message": "Test error"}`))
	server.call_spi(request, mox.IgnoreArg()).AndRaise(NewBackendError(response))

	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
	mox.VerifyAll()

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
	response = "".join(response)
	if "200 OK" != self.response_status {
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
	request := build_request("/_ah/api/explorer", "", nil)
	response := server.dispatch(request, self.start_response)
	header := new(http.Header)
	header.Set("Content-Length", "0")
	header.Set("Location", "https://developers.google.com/apis-explorer/?base=http://localhost:42/_ah/api")
	assert_http_match(t, response, 302, header, "")
}

func test_static_existing_file(t *testing.T) {
	relative_url := "/_ah/api/static/proxy.html"

	// Set up mocks for the call to DiscoveryApiProxy.get_static_file.
	discovery_api = mox.CreateMock(DiscoveryApiProxy)
	mox.StubOutWithMock(discovery_api_proxy, "DiscoveryApiProxy")
	DiscoveryApiProxy().AndReturn(discovery_api)
	static_response = mox.CreateMock(httplib.HTTPResponse)
	static_response.status = 200
	static_response.reason = "OK"
	static_response.getheader("Content-Type").AndReturn("test/type")
	test_body = "test body"
	get_static_file(relative_url).AndReturn(static_response, test_body)

	// Make sure the dispatch works as expected.
	request = build_request(relative_url, "", nil)
	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
	mox.VerifyAll()

	response = "".join(response)
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

// Test that an error response still handles CORS headers.
func test_handle_non_json_spi_response_cors(t *testing.T) {
	server_response = dispatcher.ResponseTuple(
		"200 OK", [("Content-type", "text/plain")],
		"This is an invalid response.")
	response = check_cors([("origin", "test.com")], True, "test.com", /*server_response=*/server_response)
	error_json := JsonObject{
		"error": JsonObject{
			"message": "Non-JSON reply: This is an invalid response.",
		},
	}
	var repsonse_json interface{}
	err := json.Unmarshal(response, &response_json)
	if err != nil {
		t.Fail()
	}
	if error_json != response_json {
		t.Fail()
	}
}

// Check that CORS headers are handled correctly.
//
// Args:
//   request_headers: A list of (header, value), to be used as headers in the
//     request.
//   expect_response: A boolean, whether or not CORS headers are expected in
//     the response.
//   expected_origin: A string or None.  If this is a string, this is the value
//     that"s expected in the response"s allow origin header.  This can be
//     None if expect_response is False.
//   expected_allow_headers: A string or None.  If this is a string, this is
//     the value that"s expected in the response"s allow headers header.  If
//     this is None, then the response shouldn"t have any allow headers
//     headers.
//   server_response: A dispatcher.ResponseTuple or None.  The backend"s
//     response, to be wrapped and returned as the server"s response.  If
//     this is None, a generic response will be generated.
//
// Returns:
//   A string containing the body of the response that would be sent.
func check_cors(t *testing.T, request_headers http.Header, expect_response bool, expected_origin, expected_allow_headers string, server_response *http.Response) {
	orig_request := build_request("/_ah/api/fake/path", "", request_headers)
	spi_request := orig_request.copy()

	if server_response == nil {
		server_response = dispatcher.ResponseTuple(
			"200 OK", [("Content-type", "application/json")], "{}")
	}

	response = server.handle_spi_response(orig_request, spi_request,
		server_response, self.start_response)

	headers = dict(self.response_headers)
	if expect_response {
		if _, ok := headers.Get(_CORS_HEADER_ALLOW_ORIGIN); !ok {
			t.Fail()
		} else if headers[_CORS_HEADER_ALLOW_ORIGIN] != expected_origin {
			t.Fail()
		}

		if _, ok := headers.Get(_CORS_HEADER_ALLOW_METHODS); !ok {
			t.Fail()
		}
		allow_methods := strings.Split(headers[_CORS_HEADER_ALLOW_METHODS], ",")
		sort.Strings(allow_methods)
		if allow_methods != _CORS_ALLOWED_METHODS {
			t.Fail()
		}

		if expected_allow_headers != nil {
			if _, ok := headers.Get(_CORS_HEADER_ALLOW_HEADERS); !ok {
				t.Fail()
			} else if headers.Get(_CORS_HEADER_ALLOW_HEADERS) != expected_allow_headers {
				t.Fail()
			}
		} else {
			if _, ok := headers.Get(_CORS_HEADER_ALLOW_HEADERS); ok {
				t.Fail()
			}
		}
	} else {
		if _, ok := headers.Get(_CORS_HEADER_ALLOW_ORIGIN); ok {
			t.Fail()
		}
		if _, ok := headers.Get(_CORS_HEADER_ALLOW_METHODS); ok {
			t.Fail()
		}
		if _, ok != headers.Get(_CORS_HEADER_ALLOW_HEADERS); ok {
			t.Fail()
		}
	}
	return strings.Join(response, "")
}

// Test CORS support on a regular request.
func test_handle_cors(t *testing.T) {
	header := http.Header{"origin": "test.com"}
	check_cors(t, header, true, "test.com", "", nil)
}

// Test a CORS preflight request.
func test_handle_cors_preflight(t *testing.T) {
	header := http.Header{
		"origin": "http://example.com",
		"Access-Control-Request-Method": "GET",
	}
	check_cors(t, header, true, "http://example.com", "", nil)
}

// Test a CORS preflight request for an unaccepted OPTIONS request.
func test_handle_cors_preflight_invalid(t *testing.T) {
	header := http.Header{
		"origin", "http://example.com",
		"Access-Control-Request-Method", "OPTIONS",
	}
	check_cors(t, header, false, "", "", nil)
}

// Test a CORS preflight request.
func test_handle_cors_preflight_request_headers(t *testing.T) {
	header := http.Header{
		"origin": "http://example.com",
		"Access-Control-Request-Method": "GET",
		"Access-Control-Request-Headers": "Date,Expires",
	}
	check_cors(t, header, true, "http://example.com", "Date,Expires", nil)
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
