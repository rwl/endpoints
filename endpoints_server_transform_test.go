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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"reflect"
	"testing"
)

// Tests that only hit the request transformation functions.

// Verify path is method name after a request is transformed.
func TestTransformRequest(t *testing.T) {
	server := newEndpointsServer()

	request := buildApiRequest("/_ah/api/test/{gid}", `{"sample": "body"}`,
		nil)
	methodConfig := &endpoints.ApiMethod{
		RosyMethod: "GuestbookApi.greetings_get",
	}

	params := map[string]string{"gid": "X"}
	newRequest, err := server.transformRequest(request, params, methodConfig)
	expectedBody := map[string]interface{}{
		"sample": "body",
		"gid":    "X",
	}
	assert.NoError(t, err)
	body, err := ioutil.ReadAll(newRequest.Body)
	assert.NoError(t, err)
	var bodyJson map[string]interface{}
	err = json.Unmarshal(body, &bodyJson)
	assert.NoError(t, err)
	assert.Equal(t, expectedBody, bodyJson)
	assert.Equal(t, "GuestbookApi.greetings_get", newRequest.URL.Path)
}

// Verify requestId is extracted and body is scoped to body.params.
func TestTransformJsonRpcRequest(t *testing.T) {
	server := newEndpointsServer()

	origRequest := buildApiRequest(
		"/_ah/api/rpc",
		`{"params": {"sample": "body"}, "id": "42"}`,
		nil,
	)

	newRequest, err := server.transformJsonrpcRequest(origRequest)
	assert.NoError(t, err)
	expectedBody := map[string]interface{}{"sample": "body"}
	body, err := ioutil.ReadAll(newRequest.Body)
	var bodyJson map[string]interface{}
	err = json.Unmarshal(body, &bodyJson)
	assert.NoError(t, err)
	assert.Equal(t, expectedBody, bodyJson)
	assert.Equal(t, "42", newRequest.requestId)
}

// Takes body, query and path values from a rest request for testing.
func transformRestRequest(server *EndpointsServer, pathParameters map[string]string,
	queryParameters string, bodyJson map[string]interface{},
	expected map[string]interface{}, methodParams map[string]*endpoints.ApiRequestParamSpec) error {

	if methodParams == nil {
		methodParams = make(map[string]*endpoints.ApiRequestParamSpec)
	}

	testRequest := buildApiRequest("/_ah/api/test", "", nil)
	testRequest.bodyJson = bodyJson
	body, err := json.Marshal(bodyJson)
	if err != nil {
		return err
	}
	testRequest.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	testRequest.URL.RawQuery = queryParameters

	transformedRequest, err := server.transformRestRequest(testRequest,
		pathParameters, methodParams)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(expected, transformedRequest.bodyJson) {
		return fmt.Errorf("JSON bodies do not match: %#v != %#v", expected,
			transformedRequest.bodyJson)
	}
	var trBodyJson map[string]interface{}
	body, err = ioutil.ReadAll(transformedRequest.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, &trBodyJson)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(trBodyJson, transformedRequest.bodyJson) {
		for k, v := range trBodyJson {
			vv := transformedRequest.bodyJson[k]
			fmt.Printf("%#v (%T)  -  %#v (%T)\n", v, v, vv, vv)
		}
		return fmt.Errorf("Transformed JSON bodies do not match: %#v != %#v",
			transformedRequest.bodyJson, trBodyJson)
	}
	return nil
}

// Path only.

func TestTransformRestRequestPathOnly(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"gid": "X"}
	queryParameters := ""
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{"gid": "X"}
	err := transformRestRequest(server, pathParameters,
		queryParameters, bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathOnlyMessageField(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"gid.val": "X"}
	queryParameters := ""
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "X",
		},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathOnlyEnum(t *testing.T) {
	server := newEndpointsServer()
	queryParameters := ""
	bodyObject := map[string]interface{}{}
	enumDescriptor := map[string]*endpoints.ApiEnumParamSpec{
		"X": &endpoints.ApiEnumParamSpec{BackendVal: "X"},
	}
	methodParams := map[string]*endpoints.ApiRequestParamSpec{
		"gid": &endpoints.ApiRequestParamSpec{
			Enum: enumDescriptor,
		},
	}

	// Good enum
	pathParameters := map[string]string{"gid": "X"}
	expected := map[string]interface{}{"gid": "X"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)
	assert.NoError(t, err)

	// Bad enum
	expectedPathParameters := map[string]string{"gid": "Y"}
	expectedBody := map[string]interface{}{"gid": "Y"}
	err = transformRestRequest(server, expectedPathParameters,
		queryParameters, bodyObject, expectedBody, methodParams)

	if assert.Error(t, err, "Bad enum should have caused failure.") {
		enumError, ok := err.(*enumRejectionError)
		if assert.True(t, ok, "Bad enum should have caused failure.") {
			assert.Equal(t, enumError.parameterName, "gid")
		}
	}
}

// Query only.

func TestTransformRestRequestQueryOnly(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "foo=bar"
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{"foo": "bar"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryOnlyMessageField(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "gid.val=X"
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "X",
		},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryOnlyMultipleValuesNotRepeated(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "foo=bar&foo=baz"
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{"foo": "bar"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryOnlyMultipleValuesRepeated(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "foo=bar&foo=baz"
	bodyObject := map[string]interface{}{}
	methodParams := map[string]*endpoints.ApiRequestParamSpec{
		"foo": &endpoints.ApiRequestParamSpec{Repeated: true},
	}
	expected := map[string]interface{}{"foo": []interface{}{"bar", "baz"}}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryOnlyEnum(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	bodyObject := map[string]interface{}{}
	enumDescriptor := map[string]*endpoints.ApiEnumParamSpec{
		"X": &endpoints.ApiEnumParamSpec{BackendVal: "X"},
	}
	methodParams := map[string]*endpoints.ApiRequestParamSpec{
		"gid": &endpoints.ApiRequestParamSpec{
			Enum: enumDescriptor,
		},
	}

	// Good enum
	queryParameters := "gid=X"
	expected := map[string]interface{}{"gid": "X"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)
	assert.NoError(t, err)

	// Bad enum
	queryParameters = "gid=Y"
	expected = map[string]interface{}{"gid": "Y"}
	err = transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)

	if assert.Error(t, err, "Bad enum should have caused failure.") {
		enumError, ok := err.(*enumRejectionError)
		if assert.True(t, ok, "Bad enum should have caused failure.") {
			assert.Equal(t, enumError.parameterName, "gid")
		}
	}
}

func TestTransformRestRequestQueryOnlyRepeatedEnum(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	bodyObject := map[string]interface{}{}
	enumDescriptor := map[string]*endpoints.ApiEnumParamSpec{
		"X": &endpoints.ApiEnumParamSpec{BackendVal: "X"},
		"Y": &endpoints.ApiEnumParamSpec{BackendVal: "Y"},
	}
	methodParams := map[string]*endpoints.ApiRequestParamSpec{
		"gid": &endpoints.ApiRequestParamSpec{
			Enum:     enumDescriptor,
			Repeated: true,
		},
	}

	// Good enum
	queryParameters := "gid=X&gid=Y"
	expected := map[string]interface{}{"gid": []interface{}{"X", "Y"}}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)
	assert.NoError(t, err)

	// Bad enum
	queryParameters = "gid=X&gid=Y&gid=Z"
	expected = map[string]interface{}{"gid": []interface{}{"X", "Y", "Z"}}
	err = transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)

	if assert.Error(t, err, "Bad enum should have caused failure.") {
		enumError, ok := err.(*enumRejectionError)
		if assert.True(t, ok, "Bad enum should have caused failure.") {
			assert.Equal(t, enumError.parameterName, "gid[2]")
		}
	}
}

// Body only.

func TestTransformRestRequestBodyOnly(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := ""
	bodyObject := map[string]interface{}{"sample": "body"}
	expected := map[string]interface{}{"sample": "body"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestBodyOnlyAnyOldValue(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := ""
	bodyObject := map[string]interface{}{
		"sample": map[string]interface{}{
			"body": []interface{}{"can", "be", "anything"},
		},
	}
	expected := map[string]interface{}{
		"sample": map[string]interface{}{
			"body": []interface{}{"can", "be", "anything"},
		},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestBodyOnlyMessageField(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := ""
	bodyObject := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "X",
		},
	}
	expected := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "X",
		},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestBodyOnlyEnum(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := ""
	enumDescriptor := map[string]*endpoints.ApiEnumParamSpec{
		"X": &endpoints.ApiEnumParamSpec{BackendVal: "X"},
	}
	methodParams := map[string]*endpoints.ApiRequestParamSpec{
		"gid": &endpoints.ApiRequestParamSpec{
			Enum: enumDescriptor,
		},
	}

	// Good enum
	bodyObject := map[string]interface{}{"gid": "X"}
	expected := map[string]interface{}{"gid": "X"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)
	assert.NoError(t, err)

	// Bad enum
	bodyObject = map[string]interface{}{"gid": "Y"}
	expected = map[string]interface{}{"gid": "Y"}
	err = transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)
	assert.NoError(t, err)
}

// Path and query only.

func TestTransformRestRequestPathQueryNoCollision(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := "c=d"
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathQueryCollision(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := "a=d"
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{"a": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathQueryCollisionInRepeatedParam(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := "a=d&a=c"
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{"a": []interface{}{"d", "c", "b"}}
	methodParams := map[string]*endpoints.ApiRequestParamSpec{
		"a": &endpoints.ApiRequestParamSpec{Repeated: true},
	}

	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)
	assert.NoError(t, err)
}

// Path and body only.

func TestTransformRestRequestPathBodyNoCollision(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := ""
	bodyObject := map[string]interface{}{"c": "d"}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathBodyCollision(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := ""
	bodyObject := map[string]interface{}{"a": "d"}
	expected := map[string]interface{}{"a": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathBodyCollisionInRepeatedParam(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := ""
	bodyObject := map[string]interface{}{"a": []interface{}{"d"}}
	expected := map[string]interface{}{"a": []interface{}{"d"}}
	methodParams := map[string]*endpoints.ApiRequestParamSpec{
		"a": &endpoints.ApiRequestParamSpec{Repeated: true},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathBodyMessageFieldCooperative(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"gid.val1": "X"}
	queryParameters := ""
	bodyObject := map[string]interface{}{
		"gid": map[string]interface{}{
			"val2": "Y",
		},
	}
	expected := map[string]interface{}{
		"gid": map[string]interface{}{
			"val1": "X",
			"val2": "Y",
		},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathBodyMessageFieldCollision(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"gid.val": "X"}
	queryParameters := ""
	bodyObject := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "Y",
		},
	}
	expected := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "Y",
		},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

// Query and body only.

func TestTransformRestRequestQueryBodyNoCollision(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "a=b"
	bodyObject := map[string]interface{}{"c": "d"}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryBodyCollision(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "a=b"
	bodyObject := map[string]interface{}{"a": "d"}
	expected := map[string]interface{}{"a": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryBodyCollisionInRepeatedParam(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "a=b"
	bodyObject := map[string]interface{}{"a": []interface{}{"d"}}
	expected := map[string]interface{}{"a": []interface{}{"d"}}
	methodParams := map[string]*endpoints.ApiRequestParamSpec{
		"a": &endpoints.ApiRequestParamSpec{Repeated: true},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryBodyMessageFieldCooperative(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "gid.val1=X"
	bodyObject := map[string]interface{}{
		"gid": map[string]interface{}{
			"val2": "Y",
		},
	}
	expected := map[string]interface{}{
		"gid": map[string]interface{}{
			"val1": "X", "val2": "Y",
		},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryBodyMessageFieldCollision(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "gid.val=X"
	bodyObject := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "Y",
		},
	}
	expected := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "Y",
		},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

// Path, body and query.

func TestTransformRestRequestPathQueryBodyNoCollision(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := "c=d"
	bodyObject := map[string]interface{}{"e": "f"}
	expected := map[string]interface{}{"a": "b", "c": "d", "e": "f"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathQueryBodyCollision(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := "a=d"
	bodyObject := map[string]interface{}{"a": "f"}
	expected := map[string]interface{}{"a": "f"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestUnknownParameters(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	//queryParameters := url.Values{}
	//queryParameters.Set("c", "d")
	queryParameters := "c=d"
	bodyObject := map[string]interface{}{"e": "f"}
	expected := map[string]interface{}{"a": "b", "c": "d", "e": "f"}
	methodParams := map[string]*endpoints.ApiRequestParamSpec{
		"X": &endpoints.ApiRequestParamSpec{},
		"Y": &endpoints.ApiRequestParamSpec{},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)
	assert.NoError(t, err)
}

// Other tests.

// FIXME: Verify that type conversion matches prod.
/*func TestTypeConversions(t *testing.T) {
	server := newEndpointsServer()
	pathParameters := map[string]string{
		"int32_val":      "1",
		"uint32_val":     "2",
		"int64_val":      "3",
		"uint64_val":     "4",
		"true_bool_val":  "true",
		"false_bool_val": "FALSE",
	}
	//queryParameters := url.Values{}
	//queryParameters.Set("float_val", "5.25")
	//queryParameters.Set("double_val", "6.5")
	queryParameters := "float_val=5.25&double_val=6.5"
	bodyObject := map[string]interface{}{"int_body_val": "7"}
	expected := map[string]interface{}{
		"int32_val":      1,
		"uint32_val":     uint(2),
		"int64_val":      "3",
		"uint64_val":     "4",
		"true_bool_val":  true,
		"false_bool_val": false,
		"float_val":      float32(5.25),
		"double_val":     float64(6.5),
		"int_body_val":   "7",
	}
	methodParams := map[string]*endpoints.ApiRequestParamSpec{
		"int32_val":      &endpoints.ApiRequestParamSpec{Type: "int32"},
		"uint32_val":     &endpoints.ApiRequestParamSpec{Type: "uint32"},
		"int64_val":      &endpoints.ApiRequestParamSpec{Type: "int64"},
		"uint64_val":     &endpoints.ApiRequestParamSpec{Type: "uint64"},
		"true_bool_val":  &endpoints.ApiRequestParamSpec{Type: "boolean"},
		"false_bool_val": &endpoints.ApiRequestParamSpec{Type: "boolean"},
		"float_val":      &endpoints.ApiRequestParamSpec{Type: "float"},
		"double_val":     &endpoints.ApiRequestParamSpec{Type: "double"},
		"int_body_val":   &endpoints.ApiRequestParamSpec{Type: "int32"},
	}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, methodParams)
	assert.NoError(t, err)
}*/

// Verify that invalid parameter values for basic types raise errors.
func TestInvalidConversions(t *testing.T) {
	server := newEndpointsServer()
	types := []string{"int32", "uint32", "boolean", "float", "double"}
	for _, typeName := range types {
		paramName := fmt.Sprintf("%s_val", typeName)
		pathParameters := map[string]string{paramName: "invalid"}
		//queryParameters := url.Values{}
		queryParameters := ""
		bodyObject := make(map[string]interface{})
		expected := make(map[string]interface{})
		methodParams := map[string]*endpoints.ApiRequestParamSpec{
			paramName: &endpoints.ApiRequestParamSpec{Type: typeName},
		}
		err := transformRestRequest(server, pathParameters, queryParameters,
			bodyObject, expected, methodParams)
		e, ok := err.(*basicTypeParameterError)
		if !ok {
			t.Fatalf("Bad %s value should have caused failure.", typeName)
		} else {
			assert.Equal(t, e.parameterName, paramName)
		}
	}
}
