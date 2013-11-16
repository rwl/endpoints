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

package endpoints_server

// Mapping of error codes.
//
// Provides functionality to convert HTTP error codes from the SPI to
// match the errors that will be returned by the server.
//
// todo: Generate from /google/appengine/tools/devappserver2/endpoints/generated_error_info.py

type errorInfo struct {
	httpStatus, rpcStatus int
	reason, domain        string
}

var unsupportedError = &errorInfo{404, 404, "unsupportedProtocol", "global"}
var backendErrorInfo = &errorInfo{503, -32099, "backendError", "global"}

var errorMap = map[int]*errorInfo{
	400: &errorInfo{400, 400, "badRequest", "global"},
	401: &errorInfo{401, 401, "required", "global"},
	402: &errorInfo{404, 404, "unsupportedProtocol", "global"},
	403: &errorInfo{403, 403, "forbidden", "global"},
	404: &errorInfo{404, 404, "notFound", "global"},
	405: &errorInfo{501, 501, "unsupportedMethod", "global"},
	406: &errorInfo{404, 404, "unsupportedProtocol", "global"},
	407: &errorInfo{404, 404, "unsupportedProtocol", "global"},
	408: &errorInfo{503, -32099, "backendError", "global"},
	409: &errorInfo{409, 409, "conflict", "global"},
	410: &errorInfo{410, 410, "deleted", "global"},
	411: &errorInfo{404, 404, "unsupportedProtocol", "global"},
	412: &errorInfo{412, 412, "conditionNotMet", "global"},
	413: &errorInfo{413, 413, "uploadTooLarge", "global"},
	414: &errorInfo{404, 404, "unsupportedProtocol", "global"},
	415: &errorInfo{404, 404, "unsupportedProtocol", "global"},
	416: &errorInfo{404, 404, "unsupportedProtocol", "global"},
	417: &errorInfo{404, 404, "unsupportedProtocol", "global"},
}

// Get info that would be returned by the server for this HTTP status.
//
// Takes an integer containing the HTTP status returned by the SPI and
// errorInfo containing information that would be returned by the
// live server for the provided lilyStatus.
func getErrorInfo(lilyStatus int) *errorInfo {
	if lilyStatus >= 500 {
		return backendErrorInfo
	}

	errorInfo, ok := errorMap[lilyStatus]
	if !ok {
		return unsupportedError
	}
	return errorInfo
}
