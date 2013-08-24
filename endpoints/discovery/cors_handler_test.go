
package discovery

import (
	"testing"
	"net/http"
	"strings"
	"sort"
	"io/ioutil"
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"reflect"
)

// Test that an error response still handles CORS headers.
func test_handle_non_json_spi_response_cors(t *testing.T) {
	server_response := &http.Response{
		Status: "200 OK",
		StatusCode: 200,
		Header: http.Header{"Content-Type": []string{"text/plain"}},
		Body: ioutil.NopCloser(bytes.NewBufferString("This is an invalid response.")),
	}
	response := check_cors(
		t,
		http.Header{"origin": []string{"test.com"}},
		true,
		"test.com",
		"",
		server_response,
	)
	error_json := JsonObject{
		"error": JsonObject{
			"message": "Non-JSON reply: This is an invalid response.",
		},
	}
	var response_json interface{}
	err := json.Unmarshal([]byte(response), &response_json)
	if err != nil {
		t.Fail()
	}
	if !reflect.DeepEqual(error_json, response_json) {
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
func check_cors(t *testing.T, request_headers http.Header, expect_response bool, expected_origin, expected_allow_headers string, server_response *http.Response) string {
	orig_request := build_request("/_ah/api/fake/path", "", request_headers)
	spi_request := orig_request.copy()

	if server_response == nil {
		server_response = &http.Response{
			Status: "200 OK",
			StatusCode: 200,
			Header: http.Header{"Content-type": []string{"application/json"}},
			Body: ioutil.NopCloser(bytes.NewBufferString("{}")),
		}
	}

	server, _ := set_up()
	w := httptest.NewRecorder()

	response, err := server.handle_spi_response(orig_request, spi_request,
		server_response, w)
	if err != nil {
		t.Fail()
	}

	headers := w.Header()
	if expect_response {
		if headers.Get(_CORS_HEADER_ALLOW_ORIGIN) == "" {
			t.Fail()
		} else if headers.Get(_CORS_HEADER_ALLOW_ORIGIN) != expected_origin {
			t.Fail()
		}

		if headers.Get(_CORS_HEADER_ALLOW_METHODS) == "" {
			t.Fail()
		}
		allow_methods := strings.Split(headers.Get(_CORS_HEADER_ALLOW_METHODS), ",")
		sort.Strings(allow_methods)
		if !reflect.DeepEqual(allow_methods, _CORS_ALLOWED_METHODS) {
			t.Fail()
		}

		if expected_allow_headers != "" {
			if headers.Get(_CORS_HEADER_ALLOW_HEADERS) != "" {
				t.Fail()
			} else if headers.Get(_CORS_HEADER_ALLOW_HEADERS) != expected_allow_headers {
				t.Fail()
			}
		} else {
			if headers.Get(_CORS_HEADER_ALLOW_HEADERS) != "" {
				t.Fail()
			}
		}
	} else {
		if headers.Get(_CORS_HEADER_ALLOW_ORIGIN) != "" {
			t.Fail()
		}
		if headers.Get(_CORS_HEADER_ALLOW_METHODS) != "" {
			t.Fail()
		}
		if headers.Get(_CORS_HEADER_ALLOW_HEADERS) != "" {
			t.Fail()
		}
	}
	return response
}

// Test CORS support on a regular request.
func test_handle_cors(t *testing.T) {
	header := http.Header{"origin": []string{"test.com"}}
	check_cors(t, header, true, "test.com", "", nil)
}

// Test a CORS preflight request.
func test_handle_cors_preflight(t *testing.T) {
	header := http.Header{
		"origin": []string{"http://example.com"},
		"Access-Control-Request-Method": []string{"GET"},
	}
	check_cors(t, header, true, "http://example.com", "", nil)
}

// Test a CORS preflight request for an unaccepted OPTIONS request.
func test_handle_cors_preflight_invalid(t *testing.T) {
	header := http.Header{
		"origin": []string{"http://example.com"},
		"Access-Control-Request-Method": []string{"OPTIONS"},
	}
	check_cors(t, header, false, "", "", nil)
}

// Test a CORS preflight request.
func test_handle_cors_preflight_request_headers(t *testing.T) {
	header := http.Header{
		"origin": []string{"http://example.com"},
		"Access-Control-Request-Method": []string{"GET"},
		"Access-Control-Request-Headers": []string{"Date,Expires"},
	}
	check_cors(t, header, true, "http://example.com", "Date,Expires", nil)
}
