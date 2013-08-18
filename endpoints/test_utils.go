// Test utilities.

package endpoints

import (
	"io/ioutil"
	"net/http"
	"bytes"
	"testing"
)

// Build an ApiRequest for the given path and body.
//
// Args:
//   url: A string containing the URL for the proposed request.
//   body: A string containing the body of the proposed request.
//   http_headers: A list of (header, value) headers to add to the request.
//
// Returns:
//   An ApiRequest object built based on the incoming parameters.
func build_request(url, body string, http_headers map[string]string) *ApiRequest {
	//	unused_scheme, unused_netloc, path, query, unused_fragment := urlparse.urlsplit(path)
	req, _ := http.NewRequest("GET", url,//fmt.Sprintf("localhost:%d/%s", 42, path),
		ioutil.NopCloser(bytes.NewBufferString(body)))
	req.Header.Set("Content-Type", "application/json")

	if http_headers != nil {
		for key, value := range http_headers {
			req.Header.Set(key, value)
		}
	}

	api_request, _ := newApiRequest(req)
	return api_request
}

// Test that the headers and body match.
func assert_http_match(t *testing.T, response *http.Response, expected_status int,
expected_headers http.Header, expected_body string) {
	if expected_status != response.StatusCode {
		t.Fail()
	}

	// Verify that headers match. Order shouldn't matter.
	if len(response.Header) != len(expected_headers) {
		t.Fail()
	}
	for key, value := range response.Header {
		expected_value, ok := expected_headers[key]
		if !ok {
			t.Fail()
		}
		if value != expected_value {
			t.Fail()
		}
	}

	// Convert the body to a string.
	body, _ := ioutil.ReadAll(response.Body)
	if expected_body != string(body) {
		t.Fail()
	}
}
