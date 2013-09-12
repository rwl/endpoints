package endpoint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"io/ioutil"
	"testing"
	"github.com/stretchr/testify/assert"
	"reflect"
)

/* Tests that only hit the request transformation functions.*/

// Verify path is method name after a request is transformed.
func TestTransformRequest(t *testing.T) {
	server := NewEndpointsServer()

	request := buildApiRequest("/_ah/api/test/{gid}", `{"sample": "body"}`,
		nil)
	methodConfig := &endpoints.ApiMethod{
		RosyMethod: "GuestbookApi.greetings_get",
	}

	params := map[string]string{"gid": "X"}
	newRequest, err := server.transformRequest(request, params, methodConfig)
	expectedBody := map[string]interface{}{
		"sample": "body",
		"gid": "X",
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

// Verify request_id is extracted and body is scoped to body.params.
func TestTransformJsonRpcRequest(t *testing.T) {
	server := NewEndpointsServer()

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
	assert.Equal(t, "42", newRequest.RequestId)
}

// Takes body, query and path values from a rest request for testing.
//
// Args:
//   path_parameters: A dict containing the parameters parsed from the path.
//     For example if the request came through /a/b for the template /a/{x}
//     then we"d have {"x": "b"}.
//   query_parameters: A dict containing the parameters parsed from the query
//     string.
//   body_json: A dict with the JSON object from the request body.
//   expected: A dict with the expected JSON body after being transformed.
//   method_params: Optional dictionary specifying the parameter configuration
//     associated with the method.
func transformRestRequest(server *EndpointsServer, pathParameters map[string]string,
	queryParameters string, bodyJson map[string]interface{},
	expected map[string]interface{}, methodParams map[string]*endpoints.ApiRequestParamSpec) error {

	if methodParams == nil {
		methodParams = make(map[string]*endpoints.ApiRequestParamSpec)
	}

	testRequest := buildApiRequest("/_ah/api/test", "", nil)
	testRequest.BodyJson = bodyJson
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
	if !reflect.DeepEqual(expected, transformedRequest.BodyJson) {
		return fmt.Errorf("JSON bodies do not match: %#v != %#v", expected,
			transformedRequest.BodyJson)
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
	if !reflect.DeepEqual(trBodyJson, transformedRequest.BodyJson) {
		return fmt.Errorf("Transformed JSON bodies do not match: %#v != %#v",
			transformedRequest.BodyJson, trBodyJson)
	}
	return nil
}

/* Path only. */

func TestTransformRestRequestPathOnly(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := map[string]string{"gid": "X"}
	queryParameters := ""
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{"gid": "X"}
	err := transformRestRequest(server, pathParameters,
		queryParameters, bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathOnlyMessageField(t *testing.T) {
	server := NewEndpointsServer()
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
	server := NewEndpointsServer()
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
		enumError, ok := err.(*EnumRejectionError)
		if assert.True(t, ok, "Bad enum should have caused failure.") {
			assert.Equal(t, enumError.ParameterName, "gid")
		}
	}
}

/* Query only. */

func TestTransformRestRequestQueryOnly(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "foo=bar"
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{"foo": "bar"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryOnlyMessageField(t *testing.T) {
	server := NewEndpointsServer()
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
	server := NewEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "foo=bar&foo=baz"
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{"foo": "bar"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryOnlyMultipleValuesRepeated(t *testing.T) {
	server := NewEndpointsServer()
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
	server := NewEndpointsServer()
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
		enumError, ok := err.(*EnumRejectionError)
		if assert.True(t, ok, "Bad enum should have caused failure.") {
			assert.Equal(t, enumError.ParameterName, "gid")
		}
	}
}

func TestTransformRestRequestQueryOnlyRepeatedEnum(t *testing.T) {
	server := NewEndpointsServer()
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
		enumError, ok := err.(*EnumRejectionError)
		if assert.True(t, ok, "Bad enum should have caused failure.") {
			assert.Equal(t, enumError.ParameterName, "gid[2]")
		}
	}
}

/* Body only. */

func TestTransformRestRequestBodyOnly(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := ""
	bodyObject := map[string]interface{}{"sample": "body"}
	expected := map[string]interface{}{"sample": "body"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestBodyOnlyAnyOldValue(t *testing.T) {
	server := NewEndpointsServer()
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
	server := NewEndpointsServer()
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
	server := NewEndpointsServer()
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

/* Path and query only */

func TestTransformRestRequestPathQueryNoCollision(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := "c=d"
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathQueryCollision(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := "a=d"
	bodyObject := map[string]interface{}{}
	expected := map[string]interface{}{"a": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathQueryCollisionInRepeatedParam(t *testing.T) {
	server := NewEndpointsServer()
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

/* Path and body only. */

func TestTransformRestRequestPathBodyNoCollision(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := ""
	bodyObject := map[string]interface{}{"c": "d"}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathBodyCollision(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := ""
	bodyObject := map[string]interface{}{"a": "d"}
	expected := map[string]interface{}{"a": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathBodyCollisionInRepeatedParam(t *testing.T) {
	server := NewEndpointsServer()
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
	server := NewEndpointsServer()
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
	server := NewEndpointsServer()
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

/* Query and body only */

func TestTransformRestRequestQueryBodyNoCollision(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "a=b"
	bodyObject := map[string]interface{}{"c": "d"}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryBodyCollision(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := make(map[string]string)
	queryParameters := "a=b"
	bodyObject := map[string]interface{}{"a": "d"}
	expected := map[string]interface{}{"a": "d"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestQueryBodyCollisionInRepeatedParam(t *testing.T) {
	server := NewEndpointsServer()
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
	server := NewEndpointsServer()
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
	server := NewEndpointsServer()
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

/* Path, body and query. */

func TestTransformRestRequestPathQueryBodyNoCollision(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := "c=d"
	bodyObject := map[string]interface{}{"e": "f"}
	expected := map[string]interface{}{"a": "b", "c": "d", "e": "f"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestPathQueryBodyCollision(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
	queryParameters := "a=d"
	bodyObject := map[string]interface{}{"a": "f"}
	expected := map[string]interface{}{"a": "f"}
	err := transformRestRequest(server, pathParameters, queryParameters,
		bodyObject, expected, nil)
	assert.NoError(t, err)
}

func TestTransformRestRequestUnknownParameters(t *testing.T) {
	server := NewEndpointsServer()
	pathParameters := map[string]string{"a": "b"}
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
