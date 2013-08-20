
package discovery

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
