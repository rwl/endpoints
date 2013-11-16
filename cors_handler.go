// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package endpoint

import (
	"net/http"
	"strings"
)

const corsHeaderOrigin = "Origin"
const corsHeaderRequestMethod = "Access-Control-Request-Method"
const corsHeaderRequestHeaders = "Access-Control-Request-Headers"
const corsHeaderAllowOrigin = "Access-Control-Allow-Origin"
const corsHeaderAllowMethods = "Access-Control-Allow-Methods"
const corsHeaderAllowHeaders = "Access-Control-Allow-Headers"

var corsAllowedMethods = []string{"DELETE", "GET", "PATCH", "POST", "PUT"}

type corsHandler interface {
	updateHeaders(http.Header)
}

// Track information about CORS headers and our response to them.
type checkCorsHeaders struct {
	allowCorsRequest   bool
	origin             string
	corsRequestMethod  string
	corsRequestHeaders string
}

func newCheckCorsHeaders(request *http.Request) *checkCorsHeaders {
	c := &checkCorsHeaders{allowCorsRequest: false}
	c.checkCorsRequest(request)
	return c
}

// Check for a CORS request, and see if it gets a CORS response.
func (c *checkCorsHeaders) checkCorsRequest(request *http.Request) {
	// Check for incoming CORS headers.
	c.origin = request.Header.Get(corsHeaderOrigin)
	c.corsRequestMethod = request.Header.Get(corsHeaderRequestMethod)
	c.corsRequestHeaders = request.Header.Get(corsHeaderRequestHeaders)

	// Check if the request should get a CORS response.
	in := false
	for _, method := range corsAllowedMethods {
		if method == strings.ToUpper(c.corsRequestMethod) {
			in = true
			break
		}
	}
	if len(c.origin) > 0 && ((len(c.corsRequestMethod) == 0) || in) {
		c.allowCorsRequest = true
	}
}

// Add CORS headers to the response, if needed.
func (c *checkCorsHeaders) updateHeaders(headers http.Header) {
	if !c.allowCorsRequest {
		return
	}

	// Add CORS headers.
	headers.Set(corsHeaderAllowOrigin, c.origin)
	headers.Set(corsHeaderAllowMethods,
		strings.Join(corsAllowedMethods, ","))
	if len(c.corsRequestHeaders) != 0 {
		headers.Set(corsHeaderAllowHeaders, c.corsRequestHeaders)
	}
}
