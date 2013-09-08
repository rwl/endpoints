package endpoint

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

// Test that an error response still handles CORS headers.
func Test_handle_non_json_spi_response_cors(t *testing.T) {
	server_response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString("This is an invalid response.")),
	}
	response := check_cors(
		t,
		http.Header{"origin": []string{"test.com"}},
		true,
		"test.com",
		"",
		server_response,
	)
	error_json := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Non-JSON reply: This is an invalid response.",
		},
	}
	var response_json interface{}
	err := json.Unmarshal([]byte(response), &response_json)
	assert.NoError(t, err)
	assert.Equal(t, error_json, response_json)
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
	orig_request := build_api_request("/_ah/api/fake/path", "", request_headers)
	spi_request, err := orig_request.copy()
	assert.NoError(t, err)

	if server_response == nil {
		server_response = &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Header:     http.Header{"Content-type": []string{"application/json"}},
			Body:       ioutil.NopCloser(bytes.NewBufferString("{}")),
		}
	}

	server := NewEndpointsDispatcher()
	w := httptest.NewRecorder()

	response, err := server.handle_spi_response(orig_request, spi_request,
		server_response, w)
	assert.NoError(t, err)

	headers := w.Header()
	if expect_response {
		assert.Equal(t, headers.Get(_CORS_HEADER_ALLOW_ORIGIN), expected_origin)
		allow_methods := strings.Split(headers.Get(_CORS_HEADER_ALLOW_METHODS), ",")
		sort.Strings(allow_methods)
		assert.Equal(t, allow_methods, _CORS_ALLOWED_METHODS)
		assert.Equal(t, headers.Get(_CORS_HEADER_ALLOW_HEADERS), expected_allow_headers)
	} else {
		assert.Empty(t, headers.Get(_CORS_HEADER_ALLOW_ORIGIN))
		assert.Empty(t, headers.Get(_CORS_HEADER_ALLOW_METHODS))
		assert.Empty(t, headers.Get(_CORS_HEADER_ALLOW_HEADERS))
	}
	return response
}

// Test CORS support on a regular request.
func Test_handle_cors(t *testing.T) {
	header := http.Header{"origin": []string{"test.com"}}
	check_cors(t, header, true, "test.com", "", nil)
}

// Test a CORS preflight request.
func Test_handle_cors_preflight(t *testing.T) {
	header := http.Header{
		"origin":                        []string{"http://example.com"},
		"Access-Control-Request-Method": []string{"GET"},
	}
	check_cors(t, header, true, "http://example.com", "", nil)
}

// Test a CORS preflight request for an unaccepted OPTIONS request.
func Test_handle_cors_preflight_invalid(t *testing.T) {
	header := http.Header{
		"origin":                        []string{"http://example.com"},
		"Access-Control-Request-Method": []string{"OPTIONS"},
	}
	check_cors(t, header, false, "", "", nil)
}

// Test a CORS preflight request.
func Test_handle_cors_preflight_request_headers(t *testing.T) {
	header := http.Header{
		"origin":                         []string{"http://example.com"},
		"Access-Control-Request-Method":  []string{"GET"},
		"Access-Control-Request-Headers": []string{"Date,Expires"},
	}
	check_cors(t, header, true, "http://example.com", "Date,Expires", nil)
}
