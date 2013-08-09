
package endpoints

import (
	"testing"
	"encoding/json"
	"fmt"
)

func prepare_dispatch(config) {
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
func assert_dispatch_to_spi(t *testing.T, request, config, spi_path,
		expected_spi_body_json) {
	prepare_dispatch(config)

	spi_headers := [("Content-Type", "application/json")]
	spi_body_json := expected_spi_body_json //or {}
	spi_response = dispatcher.ResponseTuple("200 OK", [], "Test")
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
	config, _ = json.Marshal(map[string]interface{}{
		"name": "guestbook_api",
		"version": "v1",
		"methods": {
			"guestbook.get": {
				"httpMethod": "GET",
				"path": "greetings/{gid}",
				"rosyMethod": "MyApi.greetings_get",
			}
		}
	})
	request = test_utils.build_request("/_ah/api/foo")
	prepare_dispatch(config)
	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
	mox.VerifyAll()

	assert_http_match(t, response, 404,
		[("Content-Type", "text/plain"),
			("Content-Length", "9")], "Not Found")
}

func test_dispatch_invalid_enum(t *testing.T) {
	config = json.Marshal(map[string]interface{}{
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

	request := build_request("/_ah/api/guestbook_api/v1/greetings/invalid_enum")
	prepare_dispatch(config)
	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
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
	config, _ := json.Marshal(map[string]interface{}{
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
	request = build_request("/_ah/api/foo")
	prepare_dispatch(config)
	mox.StubOutWithMock(self.server, "call_spi")
	// The application chose to throw a 404 error.
	response = dispatcher.ResponseTuple("404 Not Found", [],
		(`{"state": "APPLICATION_ERROR", "error_message": "Test error"}`))
	server.call_spi(request, mox.IgnoreArg()).AndRaise(NewBackendError(response))

	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
	self.mox.VerifyAll()

	expected_response = `
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
	response = "".join(response)
	assert_http_match(response, "404 Not Found",
		[("Content-Length", fmt.Sprintf("%d", len(expected_response))),
		("Content-Type", "application/json")],
		expected_response)
}

// Test than an RPC call that returns an error is handled properly.
func test_dispatch_rpc_error(t *testing.T) {
	config, _ = json.Marshal(map[string]interface{}{
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
	request = test_utils.build_request(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X", "id": "gapiRpc"}`)
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

	expected_response := map[string]interface{}{
		"error": {
			"code": 404,
			"message": "Test error",
			"data": []map[string]string{
				map[string]string{
					"domain": "global",
					"reason": "notFound",
					"message": "Test error",
				}
			},
		},
		"id": "gapiRpc",
	}
	response = "".join(response)
	if "200 OK" != self.response_status {
		t.Fail()
	}
	if expected_response != json.loads(response) {
		t.Fail()
	}
}

func test_dispatch_json_rpc(t *testing.T) {
	config, _ = json.Marshal(map[string]interface{}{
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
	request = test_utils.build_request(
		"/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X"}`)
		assert_dispatch_to_spi(request, config, "/_ah/spi/baz.bim")
}

func test_dispatch_rest(t *testing.T) {
	config = json.Marshal(map[string]interface{}{
		"name": "myapi",
		"version": "v1",
		"methods": {
			"bar": {
				"httpMethod": "GET",
				"path": "foo/{id}",
				"rosyMethod": "baz.bim"
			}
		}
	})
	request = build_request("/_ah/api/myapi/v1/foo/testId")
	assert_dispatch_to_spi(request, config,
		"/_ah/spi/baz.bim",
		map[string]string{"id": "testId"})
}

func test_explorer_redirect(t *testing.T) {
	request = build_request("/_ah/api/explorer")
	response = server.dispatch(request, self.start_response)
	assert_http_match(response, 302,
		[("Content-Length", "0"),
		("Location", ("https://developers.google.com/apis-explorer/?base=http://localhost:42/_ah/api"))])
}

func test_static_existing_file(t *testing.T) {
	relative_url = "/_ah/api/static/proxy.html"

	// Set up mocks for the call to DiscoveryApiProxy.get_static_file.
	discovery_api = mox.CreateMock(DiscoveryApiProxy)
	mox.StubOutWithMock(discovery_api_proxy, "DiscoveryApiProxy")
	DiscoveryApiProxy().AndReturn(discovery_api)
	static_response = mox.CreateMock(httplib.HTTPResponse)
	static_response.status = 200
	static_response.reason = "OK"
	static_response.getheader("Content-Type").AndReturn("test/type")
	test_body = "test body"
	get_static_file(relative_url).AndReturn((static_response, test_body))

	// Make sure the dispatch works as expected.
	request = build_request(relative_url)
	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
	mox.VerifyAll()

	response = "".join(response)
	assert_http_match(response, "200 OK",
		[("Content-Length", "%d" % len(test_body)),
			("Content-Type", "test/type")],
		test_body)
}

func test_static_non_existing_file(t *testing.T) {
	relative_url = "/_ah/api/static/blah.html"

	// Set up mocks for the call to DiscoveryApiProxy.get_static_file.
	discovery_api = self.mox.CreateMock(DiscoveryApiProxy)
	self.mox.StubOutWithMock(discovery_api_proxy, "DiscoveryApiProxy")
	discovery_api_proxy.DiscoveryApiProxy().AndReturn(discovery_api)
	static_response = mox.CreateMock(httplib.HTTPResponse)
	static_response.status = 404
	static_response.reason = "Not Found"
	static_response.getheaders().AndReturn([("Content-Type", "test/type")])
	test_body = "No Body"
	get_static_file(relative_url).AndReturn((static_response, test_body))

	// Make sure the dispatch works as expected.
	request = build_request(relative_url)
	mox.ReplayAll()
	response = server.dispatch(request, self.start_response)
	mox.VerifyAll()

	response = "".join(response)
	assert_http_match(response, "404 Not Found",
		[("Content-Length", fmt.Sprintf("%d", len(test_body))),
			("Content-Type", "test/type")],
		test_body)
}

func test_handle_non_json_spi_response(t *testing.T) {
	orig_request = build_request("/_ah/api/fake/path")
	spi_request = orig_request.copy()
	spi_response = dispatcher.ResponseTuple(
		200, [("Content-type", "text/plain")], "This is an invalid response.")
	response = server.handle_spi_response(orig_request, spi_request,
		spi_response,
		self.start_response)
	error_json = map[string]interface{}{
		"error": {
			"message": "Non-JSON reply: This is an invalid response."
		},
	}
	body = json.dumps(error_json)
	assert_http_match(response, "500",
		[("Content-Type", "application/json"),
			("Content-Length", "%d" % len(body))],
		body)
}

// Test that an error response still handles CORS headers.
func test_handle_non_json_spi_response_cors(t *testing.T) {
	server_response = dispatcher.ResponseTuple(
		"200 OK", [("Content-type", "text/plain")],
		"This is an invalid response.")
	response = check_cors([("origin", "test.com")], True, "test.com",
		/*server_response=*/server_response)
	if map[string]interface{}{"error": {"message": "Non-JSON reply: This is an invalid response."}} != json.loads(response) {
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
func check_cors(t *testing.T, request_headers, expect_response, /*expected_origin=*/None, /*expected_allow_headers=*/None, /*server_response=*/None) {
	orig_request = build_request("/_ah/api/fake/path",
		/*http_headers=*/request_headers)
	spi_request = orig_request.copy()

	if server_response == nil {
		server_response = dispatcher.ResponseTuple(
			"200 OK", [("Content-type", "application/json")], "{}")
	}

	response = server.handle_spi_response(orig_request, spi_request,
		server_response, self.start_response)

	headers = dict(self.response_headers)
	if expect_response {
		assertIn(t, _CORS_HEADER_ALLOW_ORIGIN, headers)
		if headers[_CORS_HEADER_ALLOW_ORIGIN] != expected_origin {
			t.Fail()
		}

		assertIn(t, endpoints_server._CORS_HEADER_ALLOW_METHODS, headers)
		if set(headers[
			_CORS_HEADER_ALLOW_METHODS].split(",")) != _CORS_ALLOWED_METHODS {
			t.Fail()
		}

		if expected_allow_headers != nil {
			assertIn(t, _CORS_HEADER_ALLOW_HEADERS, headers)
			if headers[_CORS_HEADER_ALLOW_HEADERS] != expected_allow_headers {
				t.Fail()
			}
		} else {
			assertNotIn(t, _CORS_HEADER_ALLOW_HEADERS, headers)
		}
	} else {
		assertNotIn(t, _CORS_HEADER_ALLOW_ORIGIN, headers)
		assertNotIn(t, _CORS_HEADER_ALLOW_METHODS, headers)
		assertNotIn(t, _CORS_HEADER_ALLOW_HEADERS, headers)
	}
	return "".join(response)
}

// Test CORS support on a regular request.
func test_handle_cors(t *testing.T) {
	check_cors(t, [("origin", "test.com")], true, "test.com")
}

// Test a CORS preflight request.
func test_handle_cors_preflight(t *testing.T) {
	check_cors([("origin", "http://example.com"),
		("Access-control-request-method", "GET")], true,
		"http://example.com")
}

// Test a CORS preflight request for an unaccepted OPTIONS request.
func test_handle_cors_preflight_invalid(t *testing.T) {
	check_cors([("origin", "http://example.com"),
		("Access-control-request-method", "OPTIONS")], false)
}

// Test a CORS preflight request.
func test_handle_cors_preflight_request_headers(t *testing.T) {
	check_cors([("origin", "http://example.com"),
		("Access-control-request-method", "GET"),
		("Access-Control-Request-Headers", "Date,Expires")], true,
		"http://example.com", "Date,Expires")
}

// Verify Lily protocol correctly uses python method name.
//
// This test verifies the fix to http://b/7189819
func test_lily_uses_python_method_name(t *testing.T) {
	config, _ := json.Marshal(map[string]interface{}{
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
	request = test_utils.build_request("/_ah/api/rpc",
		`{"method": "author.greeting.info.get", "apiVersion": "X"}`)
	assert_dispatch_to_spi(request, config, "/_ah/spi/InfoService.get", {})
}

// Verify headers transformed, JsonRpc response transformed, written.
func test_handle_spi_response_json_rpc(t *testing.T) {
	orig_request = build_request("/_ah/api/rpc",
		`{"method": "foo.bar", "apiVersion": "X"}`)
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
	assertIn(t, ("a", "b"), self.response_headers)
	if map[string]interface{} {"id": "Z", "result": {"some": "response"}} != json.loads(response) {
		t.Fail()
	}
}

// Verify that batch requests have an appropriate batch response.
func test_handle_spi_response_batch_json_rpc(t *testing.T) {
	orig_request = build_request("/_ah/api/rpc",
		`[{"method": "foo.bar", "apiVersion": "X"}]`)
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
	assertIn(("a", "b"), self.response_headers)
	if [{"id": "Z", "result": {"some": "response"}}] != json.loads(response) {
		t.Fail()
	}
}

func test_handle_spi_response_rest(t *testing.T) {
	orig_request = build_request("/_ah/api/test", "{}")
	spi_request = orig_request.copy()
	body = json.dumps({"some": "response"}, indent=1)
	spi_response = dispatcher.ResponseTuple("200 OK", [("a", "b")], body)
	response = server.handle_spi_response(orig_request, spi_request,
		spi_response, self.start_response)
	assert_http_match(response, "200 OK",
		[("a", "b"), ("Content-Length", fmt.Sprintf("%d" % len(body)))],
		body)
}

// Verify the response is reformatted correctly.
func test_transform_rest_response(t *testing.T) {
	orig_response = `{"sample": "test", "value1": {"value2": 2}}`
	expected_response = (`{\n"
			  "sample": "test", \n"
			  "value1": {\n"
			   "value2": 2\n"
			  }\n"
			 }`)
	if expected_response != server.transform_rest_response(orig_response) {
		t.Fail()
	}
}

// Verify request_id inserted into the body, and body into body.result.
func test_transform_json_rpc_response_batch(t *testing.T) {
	orig_request = test_utils.build_request("/_ah/api/rpc",
		`[{"params": {"sample": "body"}, "id": "42"}]`)
	request = orig_request.copy()
	request.request_id = "42"
	orig_response = `{"sample": "body"}`
	response = server.transform_jsonrpc_response(request, orig_response)
	if [{"result": {"sample": "body"}, "id": "42"}] != json.loads(response) {
		t.Fail()
	}
}

func test_lookup_rpc_method_no_body(t *testing.T) {
	orig_request = build_request("/_ah/api/rpc", "")
	if server.lookup_rpc_method(orig_request) != nil {
		t.Fail()
	}
}

func test_lookup_rpc_method(t *testing.T) {
	mox.StubOutWithMock(server.config_manager, "lookup_rpc_method")
	server.config_manager.lookup_rpc_method("foo", "v1").AndReturn("bar")

	mox.ReplayAll()
	orig_request = build_request("/_ah/api/rpc",
		`{"method": "foo", "apiVersion": "v1"}`)
	if "bar" != server.lookup_rpc_method(orig_request) {
		t.Fail()
	}
	mox.VerifyAll()
}

func test_verify_response(t *testing.T) {
	response = dispatcher.ResponseTuple("200", [("Content-Type", "a")], "")
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

	response = dispatcher.ResponseTuple("200", [("Content-Length", "10")], "")
	// Any content type accepted
	if !server.verify_response(response, 200, None) {
		t.Fail()
	}
	// Specified content type not matched
	if server.verify_response(response, 200, "a") {
		t.Fail()
	}
}

/* Tests that only hit the request transformation functions.*/

func setUpTransformRequestTests() {
	config_manager := &ApiConfigManager{}
	mock_dispatcher = mox.CreateMock(dispatcher.Dispatcher)
	server := NewEndpointsDispatcher(mock_dispatcher, config_manager)
}

// Verify path is method name after a request is transformed.
func test_transform_request(t *testing.T) {
	request = build_request("/_ah/api/test/{gid}", `{"sample": "body"}`)
	method_config = map[string]string{"rosyMethod": "GuestbookApi.greetings_get"}

	new_request = server.transform_request(request, {"gid": "X"}, method_config)
	if {"sample": "body", "gid": "X"} != json.loads(new_request.body) {
		t.Fail()
	}
	if "GuestbookApi.greetings_get" != new_request.path {
		t.Fail()
	}
}

// Verify request_id is extracted and body is scoped to body.params.
func test_transform_json_rpc_request(t *testing.T) {
	orig_request = test_utils.build_request("/_ah/api/rpc",
		`{"params": {"sample": "body"}, "id": "42"}`)

	new_request = server.transform_jsonrpc_request(orig_request)
	if {"sample": "body"} != json.loads(new_request.body) {
		t.Fail()
	}
	if "42" != new_request.request_id {
		t.Fail()
	}
}

// Takes body, query and path values from a rest request for testing.
//
// Args:
//   path_parameters: A dict containing the parameters parsed from the path.
//     For example if the request came through /a/b for the template /a/{x}
//     then we"d have {"x": "b"}.
//   query_parameters: A dict containing the parameters parsed from the query
//     string.
//   body_json: A dict with the JSON object from the request body.
//   expected: A dict with the expected JSON body after being transformed.
//   method_params: Optional dictionary specifying the parameter configuration
//     associated with the method.
func try_transform_rest_request(t *testing.T, path_parameters, query_parameters,
		body_json, expected, method_params map[string]interface{}/*=None*/) {
	if method_params == nil {
		method_params = make(map[string]string)
	}
	test_request = build_request("/_ah/api/test")
	test_request.body_json = body_json
	test_request.body = json.dumps(body_json)
	test_request.parameters = query_parameters

	transformed_request = server.transform_rest_request(test_request,
		path_parameters, method_params)

	if expected != transformed_request.body_json {
		t.Fail()
	}
	if transformed_request.body_json != json.loads(transformed_request.body) {
		t.Fail()
	}
}

/* Path only. */

func test_transform_rest_request_path_only(t *testing.T) {
	path_parameters := map[string]interface{}{"gid": "X"}
	query_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"gid": "X"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_path_only_message_field(t *testing.T) {
	path_parameters := map[string]interface{}{"gid.val": "X"}
	query_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"gid": {"val": "X"}}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_path_only_enum(t *testing.T) {
	query_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{}
	enum_descriptor := map[string]interface{}{"X": {"backendValue": "X"}}
	method_params := map[string]interface{}{"gid": {"enum": enum_descriptor}}

	// Good enum
	path_parameters := map[string]interface{}{"gid": "X"}
	expected := map[string]interface{}{"gid": "X"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)

	// Bad enum
	path_parameters := map[string]interface{}{"gid": "Y"}
	expected := map[string]interface{}{"gid": "Y"}
	err := try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)
	if err == nil {
		t.Fail("Bad enum should have caused failure.")
	} else {
		if error.parameter_name != "gid" {
			t.Fail()
		}
	}
}

/* Query only. */

func test_transform_rest_request_query_only(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{"foo": ["bar"]}
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"foo": "bar"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_query_only_message_field(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{"gid.val": ["X"]}
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"gid": {"val": "X"}}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_query_only_multiple_values_not_repeated(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{"foo": ["bar", "baz"]}
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"foo": "bar"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_query_only_multiple_values_repeated(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{"foo": ["bar", "baz"]}
	body_object := map[string]interface{}{}
	method_params := map[string]interface{}{"foo": {"repeated": true}}
	expected := map[string]interface{}{"foo": ["bar", "baz"]}
	try_transform_rest_request(path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)
}

func test_transform_rest_request_query_only_enum(t *testing.T) {
	path_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{}
	enum_descriptor := map[string]interface{}{"X": {"backendValue": "X"}}
	method_params := map[string]interface{}{"gid": {"enum": enum_descriptor}}

	// Good enum
	query_parameters := map[string]interface{}{"gid": ["X"]}
	expected := map[string]interface{}{"gid": "X"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)

	// Bad enum
	query_parameters := map[string]interface{}{"gid": ["Y"]}
	expected := map[string]interface{}{"gid": "Y"}
	err := try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)
	if err == nil {
		t.Fail("Bad enum should have caused failure.")
	} else {
		if err.(EnumRejectionError).parameter_name != "gid" {
			t.Fail()
		}
	}
}

func test_transform_rest_request_query_only_repeated_enum(t *testing.T) {
	path_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{}
	enum_descriptor := map[string]interface{}{"X": {"backendValue": "X"}, "Y": {"backendValue": "Y"}}
	method_params := map[string]interface{}{"gid": {"enum": enum_descriptor, "repeated": True}}

	// Good enum
	query_parameters := map[string]interface{}{"gid": ["X", "Y"]}
	expected := map[string]interface{}{"gid": ["X", "Y"]}
	try_transform_rest_request(path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)

	// Bad enum
	query_parameters := map[string]interface{}{"gid": ["X", "Y", "Z"]}
	expected := map[string]interface{}{"gid": ["X", "Y", "Z"]}
	err := try_transform_rest_request(path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)
	if err == nil {
		t.Fail("Bad enum should have caused failure.")
	} else {
		if err.(EnumRejectionError).parameter_name != "gid[2]" {
			t.Fail()
		}
	}
}

/* Body only. */

func test_transform_rest_request_body_only(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{"sample": "body"}
	expected := map[string]interface{}{"sample": "body"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)

func test_transform_rest_request_body_only_any_old_value(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{"sample": {"body": ["can", "be", "anything"]}}
	expected := map[string]interface{}{"sample": {"body": ["can", "be", "anything"]}}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_body_only_message_field(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{"gid": {"val": "X"}}
	expected := map[string]interface{}{"gid": {"val": "X"}}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_body_only_enum(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{}
	enum_descriptor := map[string]interface{}{"X": {"backendValue": "X"}}
	method_params := map[string]interface{}{"gid": {"enum": enum_descriptor}}

	// Good enum
	body_object := map[string]interface{}{"gid": "X"}
	expected := map[string]interface{}{"gid": "X"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)

	// Bad enum
	body_object := map[string]interface{}{"gid": "Y"}
	expected := map[string]interface{}{"gid": "Y"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)
}

/* Path and query only */

func test_transform_rest_request_path_query_no_collision(t *testing.T) {
	path_parameters := map[string]interface{}{"a": "b"}
	query_parameters := map[string]interface{}{"c": ["d"]}
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_path_query_collision(t *testing.T) {
	path_parameters := map[string]interface{}{"a": "b"}
	query_parameters := map[string]interface{}{"a": ["d"]}
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"a": "d"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_path_query_collision_in_repeated_param(t *testing.T) {
	path_parameters := map[string]interface{}{"a": "b"}
	query_parameters := map[string]interface{}{"a": ["d", "c"]}
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"a": ["d", "c", "b"]}
	method_params := map[string]interface{}{"a": {"repeated": true}}
	try_transform_rest_request(path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)
}

/* Path and body only. */

func test_transform_rest_request_path_body_no_collision(t *testing.T) {
	path_parameters := map[string]interface{}{"a": "b"}
	query_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{"c": "d"}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_path_body_collision(t *testing.T):
	path_parameters := map[string]interface{}{"a": "b"}
	query_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{"a": "d"}
	expected := map[string]interface{}{"a": "d"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_path_body_collision_in_repeated_param(t *testing.T) {
	path_parameters := map[string]interface{}{"a": "b"}
	query_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{"a": ["d"]}
	expected := map[string]interface{}{"a": ["d"]}
	method_params := map[string]interface{}{"a": {"repeated": true}}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)
}

func test_transform_rest_request_path_body_message_field_cooperative(t *testing.T) {
	path_parameters := map[string]interface{}{"gid.val1": "X"}
	query_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{"gid": {"val2": "Y"}}
	expected := map[string]interface{}{"gid": {"val1": "X", "val2": "Y"}}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_path_body_message_field_collision(t *testing.T) {
	path_parameters := map[string]interface{}{"gid.val": "X"}
	query_parameters := map[string]interface{}{}
	body_object := map[string]interface{}{"gid": {"val": "Y"}}
	expected := map[string]interface{}{"gid": {"val": "Y"}}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

/* Query and body only */

func test_transform_rest_request_query_body_no_collision(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{"a": ["b"]}
	body_object := map[string]interface{}{"c": "d"}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_query_body_collision(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{"a": ["b"]}
	body_object := map[string]interface{}{"a": "d"}
	expected := map[string]interface{}{"a": "d"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_query_body_collision_in_repeated_param(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{"a": ["b"]}
	body_object := map[string]interface{}{"a": ["d"]}
	expected := map[string]interface{}{"a": ["d"]}
	method_params := map[string]interface{}{"a": {"repeated": true}}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)
}

func test_transform_rest_request_query_body_message_field_cooperative(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{"gid.val1": ["X"]}
	body_object := map[string]interface{}{"gid": {"val2": "Y"}}
	expected := map[string]interface{}{"gid": {"val1": "X", "val2": "Y"}}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_query_body_message_field_collision(t *testing.T) {
	path_parameters := map[string]interface{}{}
	query_parameters := map[string]interface{}{"gid.val": ["X"]}
	body_object := map[string]interface{}{"gid": {"val": "Y"}}
	expected := map[string]interface{}{"gid": {"val": "Y"}}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

/* Path, body and query. */

func test_transform_rest_request_path_query_body_no_collision(t *testing.T) {
	path_parameters := map[string]interface{}{"a": "b"}
	query_parameters := map[string]interface{}{"c": ["d"]}
	body_object := map[string]interface{}{"e": "f"}
	expected := map[string]interface{}{"a": "b", "c": "d", "e": "f"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_path_query_body_collision(t *testing.T) {
	path_parameters := map[string]interface{}{"a": "b"}
	query_parameters := map[string]interface{}{"a": ["d"]}
	body_object := map[string]interface{}{"a": "f"}
	expected := map[string]interface{}{"a": "f"}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, nil)
}

func test_transform_rest_request_unknown_parameters(t *testing.T) {
	path_parameters := map[string]interface{}{"a": "b"}
	query_parameters := map[string]interface{}{"c": ["d"]}
	body_object := map[string]interface{}{"e": "f"}
	expected := map[string]interface{}{"a": "b", "c": "d", "e": "f"}
	method_params := map[string]interface{}{"X": {}, "Y": {}}
	try_transform_rest_request(t, path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)
}

/* Test utilities. */

// Build an ApiRequest for the given path and body.
//
// Args:
//   path: A string containing the URL for the proposed request.
//   body: A string containing the body of the proposed request.
//   http_headers: A list of (header, value) headers to add to the request.
//
// Returns:
//   An ApiRequest object built based on the incoming parameters.
func build_request(path, body, http_headers string) *http.Request {
	unused_scheme, unused_netloc, path, query, unused_fragment := urlparse.urlsplit(path)
	env := map[string]interface{}{"SERVER_PORT": 42, "REQUEST_METHOD": "GET",
		"SERVER_NAME": "localhost", "HTTP_CONTENT_TYPE": "application/json",
		"PATH_INFO": path, "wsgi.input": cStringIO.StringIO(body)}
	if query {
		env["QUERY_STRING"] = query
	}

	if http_headers {
		for header := range http_headers {
			header = fmt.Sprintf("HTTP_%s", header.upper().replace("-", "_"))
			env[header] = value
		}
	}

	cgi_request = api_request.ApiRequest(env)
	return cgi_request
}

// Test that the headers and body match.
func assert_http_match(t *testing.T, response, response_status, expected_status,
		response_headers, expected_headers, response_body, expected_body) {
	if string(expected_status) != response_status {
		t.Fail()
	}

	// Verify that headers match.  Order shouldn't matter.
	if len(response_headers) != len(expected_headers) {
		t.Fail()
	}
	if set(self.response_headers) != set(expected_headers) {
		t.Fail()
	}
	// Make sure there are no duplicate headers in the response.
//	self.assertEqual(len(self.response_headers),
//	len(set(header for header, _ in self.response_headers)))

	// Convert the body from an iterator to a string.
	body = "".join(response)
	if expected_body != body {
		t.Fail()
	}
}
