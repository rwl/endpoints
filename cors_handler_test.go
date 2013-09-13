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
func TestHandleNonJsonSpiResponseCors(t *testing.T) {
	serverResponse := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString("This is an invalid response.")),
	}
	response := checkCors(
		t,
		http.Header{"origin": []string{"test.com"}},
		true,
		"test.com",
		"",
		serverResponse,
	)
	errorJson := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Non-JSON reply: This is an invalid response.",
		},
	}
	var responseJson interface{}
	err := json.Unmarshal([]byte(response), &responseJson)
	assert.NoError(t, err)
	assert.Equal(t, errorJson, responseJson)
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
func checkCors(t *testing.T, requestHeaders http.Header, expectResponse bool,
		expectedOrigin, expectedAllowHeaders string, serverResponse *http.Response) string {
	origRequest := buildApiRequest("/_ah/api/fake/path", "", requestHeaders)
	spiRequest, err := origRequest.Copy()
	assert.NoError(t, err)

	if serverResponse == nil {
		serverResponse = &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Header:     http.Header{"Content-type": []string{"application/json"}},
			Body:       ioutil.NopCloser(bytes.NewBufferString("{}")),
		}
	}

	server := NewEndpointsServer()
	w := httptest.NewRecorder()

	response, err := server.handleSpiResponse(origRequest, spiRequest,
		serverResponse, w)
	assert.NoError(t, err)

	headers := w.Header()
	if expectResponse {
		assert.Equal(t, headers.Get(CORS_HEADER_ALLOW_ORIGIN), expectedOrigin)
		allowMethods := strings.Split(headers.Get(CORS_HEADER_ALLOW_METHODS), ",")
		sort.Strings(allowMethods)
		assert.Equal(t, allowMethods, CorsAllowedMethods)
		assert.Equal(t, headers.Get(CORS_HEADER_ALLOW_HEADERS), expectedAllowHeaders)
	} else {
		assert.Empty(t, headers.Get(CORS_HEADER_ALLOW_ORIGIN))
		assert.Empty(t, headers.Get(CORS_HEADER_ALLOW_METHODS))
		assert.Empty(t, headers.Get(CORS_HEADER_ALLOW_HEADERS))
	}
	return response
}

// Test CORS support on a regular request.
func TestHandleCors(t *testing.T) {
	header := http.Header{"origin": []string{"test.com"}}
	checkCors(t, header, true, "test.com", "", nil)
}

// Test a CORS preflight request.
func TestHandleCorsPreflight(t *testing.T) {
	header := http.Header{
		"origin":                        []string{"http://example.com"},
		"Access-Control-Request-Method": []string{"GET"},
	}
	checkCors(t, header, true, "http://example.com", "", nil)
}

// Test a CORS preflight request for an unaccepted OPTIONS request.
func TestHandleCorsPreflightInvalid(t *testing.T) {
	header := http.Header{
		"origin":                        []string{"http://example.com"},
		"Access-Control-Request-Method": []string{"OPTIONS"},
	}
	checkCors(t, header, false, "", "", nil)
}

// Test a CORS preflight request.
func TestHandleCorsPreflightRequestHeaders(t *testing.T) {
	header := http.Header{
		"origin":                         []string{"http://example.com"},
		"Access-Control-Request-Method":  []string{"GET"},
		"Access-Control-Request-Headers": []string{"Date,Expires"},
	}
	checkCors(t, header, true, "http://example.com", "Date,Expires", nil)
}
