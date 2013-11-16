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
	"log"
	"net/http"
)

// Errors used in the local Cloud Endpoints server.

type requestError interface {
	statusCode() int
	rpcError() map[string]interface{}
	restError() string
}

// Base type for errors that happen while processing a request.
type baseRequestError struct {
	code int // HTTP status code number associated with this error.

	message string // Text message explaining the error.

	// Error reason is a custom string in the Cloud Endpoints server. When
	// possible, this should match the reason that the live server will
	// generate, based on the error's status code.  If this is empty,
	// the error formatter will attempt to generate a reason from the status
	// code.
	reason string

	domain string // The string "global" by default.

	// Some errors have additional information. This provides a way for
	// subclasses to provide that information.
	extraFields map[string]interface{}
}

func (re *baseRequestError) statusCode() int {
	return re.code
}

func (re *baseRequestError) Error() string {
	return re.message
}

// Format this error into a JSON response.
func (err *baseRequestError) FormatError(errorListTag string) map[string]interface{} {
	errorMap := map[string]interface{}{
		"domain":  err.domain,
		"reason":  err.reason,
		"message": err.message,
	}
	for k, v := range err.extraFields {
		errorMap[k] = v
	}
	return map[string]interface{}{
		"error": map[string]interface{}{
			errorListTag: []map[string]interface{}{errorMap},
			"code":       err.statusCode(),
			"message":    err.message,
		},
	}
}

// Format this error into a response to a REST request.
func (err *baseRequestError) restError() string {
	errorJson := err.FormatError("errors")
	rest, e := json.MarshalIndent(errorJson, "", "  ") // todo: sort keys
	if e != nil {
		log.Printf("Problem formatting error as REST response: %s", e.Error())
		return e.Error()
	}
	return string(rest)
}

// Format this error into a response to a JSON RPC request.
func (err *baseRequestError) rpcError() map[string]interface{} {
	return err.FormatError("data")
}

// Base type for invalid parameter errors.
// Embedding types only need to set the message attribute.
type invalidParameterError struct {
	baseRequestError
	parameterName string // The name of the enum parameter which had a value rejected.
	value         string // The actual value passed in for the parameter.
}

func newInvalidParameterError(parameterName, value string) *invalidParameterError {
	return &invalidParameterError{
		baseRequestError: baseRequestError{
			code:    400,
			message: fmt.Sprintf("Invalid value: %s", value),
			reason:  "invalidParameter",
			extraFields: map[string]interface{}{
				"locationType": "parameter",
				"location":     parameterName,
			},
		},
		parameterName: parameterName,
		value:         value,
	}
}

func (err *invalidParameterError) Error() string {
	return err.message
}

// Request rejection exception for basic types (int, bool).
type basicTypeParameterError struct {
	invalidParameterError
	typeName string // Descriptive name of the data type expected.
}

func newBasicTypeParameterError(parameterName, value, typeName string) *basicTypeParameterError {
	paramError := &basicTypeParameterError{
		*newInvalidParameterError(parameterName, value),
		typeName,
	}
	paramError.message = fmt.Sprintf("Invalid %s value: %s", typeName, value)
	return paramError
}

// Request rejection exception for enum values.
type enumRejectionError struct {
	invalidParameterError
	allowedValues []string // Allowed values for the enum.
}

func newEnumRejectionError(parameterName, value string, allowedValues []string) *enumRejectionError {
	enumErr := &enumRejectionError{
		*newInvalidParameterError(parameterName, value),
		allowedValues,
	}
	enumErr.message = fmt.Sprintf("Invalid string value: %s. Allowed values: %s", value, allowedValues)
	return enumErr
}

// Error returned when the backend SPI returns an error code.
type backendError struct {
	baseRequestError
	errorInfo *errorInfo
}

func newBackendError(response *http.Response) *backendError {
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

	return &backendError{
		baseRequestError: baseRequestError{
			code:    errorInfo.httpStatus,
			message: message,
			reason:  errorInfo.reason,
			domain:  errorInfo.domain,
		},
		errorInfo: errorInfo,
	}
}

func (err *backendError) Error() string {
	return err.message
}
