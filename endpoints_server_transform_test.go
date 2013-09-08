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

/*func setUpTransformRequestTests() *EndpointsServer {
	config_manager := NewApiConfigManager()
	mock_dispatcher := &MockDispatcher{}
	return NewEndpointsServerConfig(mock_dispatcher, config_manager)
}*/

// Verify path is method name after a request is transformed.
func Test_transform_request(t *testing.T) {
	server := NewEndpointsServer()

	request := build_api_request("/_ah/api/test/{gid}", `{"sample": "body"}`, nil)
	method_config := &endpoints.ApiMethod{
		RosyMethod: "GuestbookApi.greetings_get",
	}

	params := map[string]string{"gid": "X"}
	new_request, err := server.transform_request(request, params, method_config)
	expected_body := map[string]interface{}{
		"sample": "body",
		"gid": "X",
	}
	assert.NoError(t, err)
	body, err := ioutil.ReadAll(new_request.Body)
	assert.NoError(t, err)
	var body_json map[string]interface{}
	err = json.Unmarshal(body, &body_json)
	assert.NoError(t, err)
	assert.Equal(t, expected_body, body_json)
	assert.Equal(t, "GuestbookApi.greetings_get", new_request.URL.Path)
}

// Verify request_id is extracted and body is scoped to body.params.
func Test_transform_json_rpc_request(t *testing.T) {
	server := NewEndpointsServer()

	orig_request := build_api_request(
		"/_ah/api/rpc",
		`{"params": {"sample": "body"}, "id": "42"}`,
		nil,
	)

	new_request, err := server.transform_jsonrpc_request(orig_request)
	assert.NoError(t, err)
	expected_body := map[string]interface{}{"sample": "body"}
	body, err := ioutil.ReadAll(new_request.Body)
	var body_json map[string]interface{}
	err = json.Unmarshal(body, &body_json)
	assert.NoError(t, err)
	assert.Equal(t, expected_body, body_json)
	assert.Equal(t, "42", new_request.request_id)
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
func transform_rest_request(server *EndpointsServer, path_parameters map[string]string,
	query_parameters string, body_json map[string]interface{},
	expected map[string]interface{}, method_params map[string]*endpoints.ApiRequestParamSpec) error {

	if method_params == nil {
		method_params = make(map[string]*endpoints.ApiRequestParamSpec)
	}

	test_request := build_api_request("/_ah/api/test", "", nil)
	test_request.body_json = body_json
	body, err := json.Marshal(body_json)
	if err != nil {
		return err
	}
	test_request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	test_request.URL.RawQuery = query_parameters

	transformed_request, err := server.transform_rest_request(test_request,
		path_parameters, method_params)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(expected, transformed_request.body_json) {
		return fmt.Errorf("JSON bodies do not match: %#v != %#v", expected,
			transformed_request.body_json)
	}
	var tr_body_json map[string]interface{}
	body, err = ioutil.ReadAll(transformed_request.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, &tr_body_json)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(tr_body_json, transformed_request.body_json) {
		return fmt.Errorf("Transformed JSON bodies do not match: %#v != %#v",
			transformed_request.body_json, tr_body_json)
	}
	return nil
}

/* Path only. */

func Test_transform_rest_request_path_only(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"gid": "X"}
	query_parameters := ""
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"gid": "X"}
	err := transform_rest_request(server, path_parameters,
		query_parameters, body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_path_only_message_field(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"gid.val": "X"}
	query_parameters := ""
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"gid": map[string]interface{}{"val": "X"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_path_only_enum(t *testing.T) {
	server := NewEndpointsServer()
	query_parameters := ""
	body_object := map[string]interface{}{}
	enum_descriptor := map[string]*endpoints.ApiEnumParamSpec{
		"X": &endpoints.ApiEnumParamSpec{BackendVal: "X"},
	}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"gid": &endpoints.ApiRequestParamSpec{
			Enum: enum_descriptor,
		},
	}

	// Good enum
	path_parameters := map[string]string{"gid": "X"}
	expected := map[string]interface{}{"gid": "X"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected /*method_params=*/, method_params)
	assert.NoError(t, err)

	// Bad enum
	expected_path_parameters := map[string]string{"gid": "Y"}
	expected_body := map[string]interface{}{"gid": "Y"}
	err = transform_rest_request(server, expected_path_parameters, query_parameters,
		body_object, expected_body /*method_params=*/, method_params)

	if assert.Error(t, err, "Bad enum should have caused failure.") {
		enum_error, ok := err.(*EnumRejectionError)
		if assert.True(t, ok, "Bad enum should have caused failure.") {
			assert.Equal(t, enum_error.parameter_name, "gid")
		}
	}
}

/* Query only. */

func Test_transform_rest_request_query_only(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := "foo=bar"
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"foo": "bar"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_query_only_message_field(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := "gid.val=X"
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"gid": map[string]interface{}{"val": "X"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)}

func Test_transform_rest_request_query_only_multiple_values_not_repeated(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := "foo=bar&foo=baz"
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"foo": "bar"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_query_only_multiple_values_repeated(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := "foo=bar&foo=baz"
	body_object := map[string]interface{}{}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"foo": &endpoints.ApiRequestParamSpec{Repeated: true},
	}
	expected := map[string]interface{}{"foo": []interface{}{"bar", "baz"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	assert.NoError(t, err)
}

func Test_transform_rest_request_query_only_enum(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	body_object := map[string]interface{}{}
	enum_descriptor := map[string]*endpoints.ApiEnumParamSpec{
		"X": &endpoints.ApiEnumParamSpec{BackendVal: "X"},
	}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"gid": &endpoints.ApiRequestParamSpec{
			Enum: enum_descriptor,
		},
	}

	// Good enum
	query_parameters := "gid=X"
	expected := map[string]interface{}{"gid": "X"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	assert.NoError(t, err)

	// Bad enum
	query_parameters = "gid=Y"
	expected = map[string]interface{}{"gid": "Y"}
	err = transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected /*method_params=*/, method_params)

	if assert.Error(t, err, "Bad enum should have caused failure.") {
		enum_error, ok := err.(*EnumRejectionError)
		if assert.True(t, ok, "Bad enum should have caused failure.") {
			assert.Equal(t, enum_error.parameter_name, "gid")
		}
	}
}

func Test_transform_rest_request_query_only_repeated_enum(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	body_object := map[string]interface{}{}
	enum_descriptor := map[string]*endpoints.ApiEnumParamSpec{
		"X": &endpoints.ApiEnumParamSpec{BackendVal: "X"},
		"Y": &endpoints.ApiEnumParamSpec{BackendVal: "Y"},
	}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"gid": &endpoints.ApiRequestParamSpec{
			Enum:     enum_descriptor,
			Repeated: true,
		},
	}

	// Good enum
	query_parameters := "gid=X&gid=Y"
	expected := map[string]interface{}{"gid": []interface{}{"X", "Y"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	assert.NoError(t, err)

	// Bad enum
	query_parameters = "gid=X&gid=Y&gid=Z"
	expected = map[string]interface{}{"gid": []interface{}{"X", "Y", "Z"}}
	err = transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)

	if assert.Error(t, err, "Bad enum should have caused failure.") {
		enum_error, ok := err.(*EnumRejectionError)
		if assert.True(t, ok, "Bad enum should have caused failure.") {
			assert.Equal(t, enum_error.parameter_name, "gid[2]")
		}
	}
}

/* Body only. */

func Test_transform_rest_request_body_only(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := ""
	body_object := map[string]interface{}{"sample": "body"}
	expected := map[string]interface{}{"sample": "body"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_body_only_any_old_value(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := ""
	body_object := map[string]interface{}{
		"sample": map[string]interface{}{
			"body": []interface{}{"can", "be", "anything"},
		},
	}
	expected := map[string]interface{}{
		"sample": map[string]interface{}{
			"body": []interface{}{"can", "be", "anything"},
		},
	}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_body_only_message_field(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := ""
	body_object := map[string]interface{}{"gid": map[string]interface{}{"val": "X"}}
	expected := map[string]interface{}{"gid": map[string]interface{}{"val": "X"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_body_only_enum(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := ""
	enum_descriptor := map[string]*endpoints.ApiEnumParamSpec{
		"X": &endpoints.ApiEnumParamSpec{BackendVal: "X"},
	}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"gid": &endpoints.ApiRequestParamSpec{
			Enum: enum_descriptor,
		},
	}

	// Good enum
	body_object := map[string]interface{}{"gid": "X"}
	expected := map[string]interface{}{"gid": "X"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	assert.NoError(t, err)

	// Bad enum
	body_object = map[string]interface{}{"gid": "Y"}
	expected = map[string]interface{}{"gid": "Y"}
	err = transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	assert.NoError(t, err)
}

/* Path and query only */

func Test_transform_rest_request_path_query_no_collision(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "c=d"
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_path_query_collision(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "a=d"
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"a": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_path_query_collision_in_repeated_param(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "a=d&a=c"
	body_object := map[string]interface{}{}
	expected := map[string]interface{}{"a": []interface{}{"d", "c", "b"}}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"a": &endpoints.ApiRequestParamSpec{Repeated: true},
	}

	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	assert.NoError(t, err)
}

/* Path and body only. */

func Test_transform_rest_request_path_body_no_collision(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := ""
	body_object := map[string]interface{}{"c": "d"}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_path_body_collision(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := ""
	body_object := map[string]interface{}{"a": "d"}
	expected := map[string]interface{}{"a": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_path_body_collision_in_repeated_param(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := ""
	body_object := map[string]interface{}{"a": []interface{}{"d"}}
	expected := map[string]interface{}{"a": []interface{}{"d"}}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"a": &endpoints.ApiRequestParamSpec{Repeated: true},
	}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	assert.NoError(t, err)
}

func Test_transform_rest_request_path_body_message_field_cooperative(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"gid.val1": "X"}
	query_parameters := ""
	body_object := map[string]interface{}{
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
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_path_body_message_field_collision(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"gid.val": "X"}
	query_parameters := ""
	body_object := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "Y",
		},
	}
	expected := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "Y",
		},
	}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

/* Query and body only */

func Test_transform_rest_request_query_body_no_collision(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := "a=b"
	body_object := map[string]interface{}{"c": "d"}
	expected := map[string]interface{}{"a": "b", "c": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_query_body_collision(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := "a=b"
	body_object := map[string]interface{}{"a": "d"}
	expected := map[string]interface{}{"a": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_query_body_collision_in_repeated_param(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := "a=b"
	body_object := map[string]interface{}{"a": []interface{}{"d"}}
	expected := map[string]interface{}{"a": []interface{}{"d"}}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"a": &endpoints.ApiRequestParamSpec{Repeated: true},
	}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	assert.NoError(t, err)
}

func Test_transform_rest_request_query_body_message_field_cooperative(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := "gid.val1=X"
	body_object := map[string]interface{}{
		"gid": map[string]interface{}{
			"val2": "Y",
		},
	}
	expected := map[string]interface{}{
		"gid": map[string]interface{}{
			"val1": "X", "val2": "Y",
		},
	}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_query_body_message_field_collision(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := make(map[string]string)
	query_parameters := "gid.val=X"
	body_object := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "Y",
		},
	}
	expected := map[string]interface{}{
		"gid": map[string]interface{}{
			"val": "Y",
		},
	}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

/* Path, body and query. */

func Test_transform_rest_request_path_query_body_no_collision(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "c=d"
	body_object := map[string]interface{}{"e": "f"}
	expected := map[string]interface{}{"a": "b", "c": "d", "e": "f"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_path_query_body_collision(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "a=d"
	body_object := map[string]interface{}{"a": "f"}
	expected := map[string]interface{}{"a": "f"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	assert.NoError(t, err)
}

func Test_transform_rest_request_unknown_parameters(t *testing.T) {
	server := NewEndpointsServer()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "c=d"
	body_object := map[string]interface{}{"e": "f"}
	expected := map[string]interface{}{"a": "b", "c": "d", "e": "f"}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"X": &endpoints.ApiRequestParamSpec{},
		"Y": &endpoints.ApiRequestParamSpec{},
	}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	assert.NoError(t, err)
}
