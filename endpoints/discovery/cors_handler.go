
package discovery

import (
	"net/http"
	"strings"
)

const _CORS_HEADER_ORIGIN = "Origin"
const _CORS_HEADER_REQUEST_METHOD = "Access-Control-Request-Method"
const _CORS_HEADER_REQUEST_HEADERS = "Access-Control-Request-Headers"
const _CORS_HEADER_ALLOW_ORIGIN = "Access-Control-Allow-Origin"
const _CORS_HEADER_ALLOW_METHODS = "Access-Control-Allow-Methods"
const _CORS_HEADER_ALLOW_HEADERS = "Access-Control-Allow-Headers"

var _CORS_ALLOWED_METHODS = []string{"DELETE", "GET", "PATCH", "POST", "PUT"}

type CorsHandler interface {
	UpdateHeaders(http.Header)
}

// Track information about CORS headers and our response to them.
type checkCorsHeaders struct {
	allow_cors_request bool
	origin string
	cors_request_method string
	cors_request_headers string
}

func newCheckCorsHeaders(request *http.Request) *checkCorsHeaders {
	c := &checkCorsHeaders{false}
	c.check_cors_request(request)
	return c
}

// Check for a CORS request, and see if it gets a CORS response.
func (c *checkCorsHeaders) check_cors_request(request *http.Request) {
	// Check for incoming CORS headers.
	c.origin = request.Header.Get(_CORS_HEADER_ORIGIN)
	c.cors_request_method = request.Header.Get(_CORS_HEADER_REQUEST_METHOD)
	c.cors_request_headers = request.Header.Get(_CORS_HEADER_REQUEST_HEADERS)

	// Check if the request should get a CORS response.
	in := false
	for _, method := range _CORS_ALLOWED_METHODS {
		if method == strings.ToUpper(c.cors_request_method) {
			in = true
			break
		}
	}
	if len(c.origin) > 0 && ((len(c.cors_request_method) == 0) || in) {
		c.allow_cors_request = true
	}
}

// Add CORS headers to the response, if needed.
func (c *checkCorsHeaders) UpdateHeaders(headers http.Header) {
	if !c.allow_cors_request {
		return
	}

	// Add CORS headers.
	headers.Set(_CORS_HEADER_ALLOW_ORIGIN, c.origin)
	headers.Set(_CORS_HEADER_ALLOW_METHODS,
		strings.Join(_CORS_ALLOWED_METHODS, ","))
	if len(c.cors_request_headers) != 0 {
		headers.Set(_CORS_HEADER_ALLOW_HEADERS, c.cors_request_headers)
	}
}
