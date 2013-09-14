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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"github.com/golang/glog"
	"net/http"
)

// Errors used in the local Cloud Endpoints server.

type RequestError interface {
	StatusCode() int
	RpcError() map[string]interface{}
	RestError() string
}

// Base type for errors that happen while processing a request.
type BaseRequestError struct {
	statusCode int // HTTP status code number associated with this error.

	Message string // Text message explaining the error.

	// Error reason is a custom string in the Cloud Endpoints server. When
	// possible, this should match the reason that the live server will
	// generate, based on the error's status code.  If this is empty,
	// the error formatter will attempt to generate a reason from the status
	// code.
	Reason string

	Domain string // The string "global" by default.

	// Some errors have additional information. This provides a way for
	// subclasses to provide that information.
	ExtraFields map[string]interface{}
}

func (re *BaseRequestError) StatusCode() int {
	return re.statusCode
}

func (re *BaseRequestError) Error() string {
	return re.Message
}

// Format this error into a JSON response.
func (err *BaseRequestError) FormatError(errorListTag string) map[string]interface{} {
	errorMap := map[string]interface{}{
		"domain":  err.Domain,
		"reason":  err.Reason,
		"message": err.Message,
	}
	for k, v := range err.ExtraFields {
		errorMap[k] = v
	}
	return map[string]interface{}{
		"error": map[string]interface{}{
			errorListTag: []map[string]interface{}{errorMap},
			"code":       err.StatusCode(),
			"message":    err.Message,
		},
	}
}

// Format this error into a response to a REST request.
func (err *BaseRequestError) RestError() string {
	errorJson := err.FormatError("errors")
	rest, e := json.MarshalIndent(errorJson, "", "  ") // todo: sort keys
	if e != nil {
		glog.Errorf("Problem formatting error as REST response: %s", e.Error())
		return e.Error()
	}
	return string(rest)
}

// Format this error into a response to a JSON RPC request.
func (err *BaseRequestError) RpcError() map[string]interface{} {
	return err.FormatError("data")
}

// Request rejection exception for enum values.
type EnumRejectionError struct {
	BaseRequestError
	ParameterName string   // The name of the enum parameter which had a value rejected.
	Value         string   // The actual value passed in for the enum.
	AllowedValues []string // List of strings allowed for the enum.
}

func NewEnumRejectionError(parameterName, value string, allowedValues []string) *EnumRejectionError {
	return &EnumRejectionError{
		BaseRequestError: BaseRequestError{
			statusCode: 400,
			Message:    fmt.Sprintf("Invalid string value: %s. Allowed values: %v", value, allowedValues),
			Reason:     "invalidParameter",
			ExtraFields: map[string]interface{}{
				"locationType": "parameter",
				"location":     parameterName,
			},
		},
		ParameterName: parameterName,
		Value:         value,
		AllowedValues: allowedValues,
	}
}

func (err *EnumRejectionError) Error() string {
	return err.Message
}

// Error returned when the backend SPI returns an error code.
type BackendError struct {
	BaseRequestError
	ErrorInfo *ErrorInfo
}

func NewBackendError(response *http.Response) *BackendError {
	// Convert backend error status to whatever the live server would return.
	errorInfo := getErrorInfo(response.StatusCode)

	var errorJson map[string]interface{}
	body, _ := ioutil.ReadAll(response.Body)
	err := json.Unmarshal(body, &errorJson)
	var message string
	if err == nil {
		_message, ok := errorJson["error_message"]
		if ok {
			message, ok = _message.(string)
			if !ok {
				message = string(body)
			}
		} else {
			message = string(body)
		}
	} else {
		message = string(body)
	}

	return &BackendError{
		BaseRequestError: BaseRequestError{
			statusCode: errorInfo.HttpStatus,
			Message:    message,
			Reason:     errorInfo.Reason,
			Domain:     errorInfo.Domain,
		},
		ErrorInfo: errorInfo,
	}
}

func (err *BackendError) Error() string {
	return err.Message
}
