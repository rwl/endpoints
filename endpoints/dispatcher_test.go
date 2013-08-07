
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
	config = json.dumps(map[string]interface{}{
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
