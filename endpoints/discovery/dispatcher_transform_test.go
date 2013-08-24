
package discovery

import (
	"testing"
	"encoding/json"
	"io/ioutil"
	"bytes"
	"errors"
	"github.com/crhym3/go-endpoints/endpoints"
	"reflect"
)


/* Tests that only hit the request transformation functions.*/

func setUpTransformRequestTests() *EndpointsDispatcher {
	config_manager := NewApiConfigManager()
	mock_dispatcher := &MockDispatcher{}
	return NewEndpointsDispatcherConfig(mock_dispatcher, config_manager)
}

// Verify path is method name after a request is transformed.
func test_transform_request(t *testing.T) {
	server := setUpTransformRequestTests()

	request := build_request("/_ah/api/test/{gid}", `{"sample": "body"}`, nil)
	method_config := &endpoints.ApiMethod{
		RosyMethod: "GuestbookApi.greetings_get",
	}

	params := map[string]string{"gid": "X"}
	new_request := server.transform_request(request, params, method_config)
	expected_body := JsonObject{"sample": "body", "gid": "X"}
	body, err := ioutil.ReadAll(new_request.Body)
	if err != nil {
		t.Fail()
	}
	var body_json interface{}
	err = json.Unmarshal(body, body_json)
	if err != nil {
		t.Fail()
	}
	if !reflect.DeepEqual(expected_body, body_json) {
		t.Fail()
	}
	if "GuestbookApi.greetings_get" != new_request.URL.Path {
		t.Fail()
	}
}

// Verify request_id is extracted and body is scoped to body.params.
func test_transform_json_rpc_request(t *testing.T) {
	server := setUpTransformRequestTests()

	orig_request := build_request(
		"/_ah/api/rpc",
		`{"params": {"sample": "body"}, "id": "42"}`,
		nil,
	)

	new_request, err := server.transform_jsonrpc_request(orig_request)
	if err != nil {
		t.Fail()
	}
	expected_body := JsonObject{"sample": "body"}
	body, err := ioutil.ReadAll(new_request.Body)
	var body_json interface{}
	err = json.Unmarshal(body, body_json)
	if err != nil {
		t.Fail()
	}
	if !reflect.DeepEqual(expected_body, body_json) {
		t.Fail()
	}
	if "42" != new_request.request_id {
		t.Fail()
	}
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
func transform_rest_request(server *EndpointsDispatcher, path_parameters map[string]string,
		query_parameters string, body_json JsonObject,
		expected JsonObject, method_params map[string]*endpoints.ApiRequestParamSpec) error {

	if method_params == nil {
		method_params = make(map[string]*endpoints.ApiRequestParamSpec)
	}

	test_request := build_request("/_ah/api/test", "", nil)
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
		return errors.New("JSON bodies do not match")
	}
	var tr_body_json interface{}
	body, err = ioutil.ReadAll(transformed_request.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, &tr_body_json)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(transformed_request.body_json, tr_body_json) {
		return errors.New("Transformed JSON bodies do not match")
	}
	return nil
}

/* Path only. */

func test_transform_rest_request_path_only(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"gid": "X"}
	query_parameters := ""
	body_object := JsonObject{}
	expected := JsonObject{"gid": "X"}
	err := transform_rest_request(server, path_parameters,
		query_parameters, body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_path_only_message_field(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"gid.val": "X"}
	query_parameters := ""
	body_object := JsonObject{}
	expected := JsonObject{"gid": JsonObject{"val": "X"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_path_only_enum(t *testing.T) {
	server := setUpTransformRequestTests()
	query_parameters := ""
	body_object := JsonObject{}
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
	expected := JsonObject{"gid": "X"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)
	if err != nil {
		t.Fail()
	}

	// Bad enum
	expected_path_parameters := map[string]string{"gid": "Y"}
	expected_body := JsonObject{"gid": "Y"}
	err = transform_rest_request(server, expected_path_parameters, query_parameters,
		body_object, expected_body, /*method_params=*/method_params)
	if err == nil {
		t.Error("Bad enum should have caused failure.")
	} else if _, ok := err.(*EnumRejectionError); !ok {
		t.Error("Bad enum should have caused failure.")
	} else {
		enum_error := err.(*EnumRejectionError)
		if enum_error.parameter_name != "gid" {
			t.Fail()
		}
	}
}

/* Query only. */

func test_transform_rest_request_query_only(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := "foo=bar"
	body_object := JsonObject{}
	expected := JsonObject{"foo": "bar"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_query_only_message_field(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := "gid.val=X"
	body_object := JsonObject{}
	expected := JsonObject{"gid": JsonObject{"val": "X"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_query_only_multiple_values_not_repeated(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := "foo=bar,baz" // todo: check query form
	body_object := JsonObject{}
	expected := JsonObject{"foo": "bar"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_query_only_multiple_values_repeated(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := "foo=bar,baz"
	body_object := JsonObject{}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"foo": &endpoints.ApiRequestParamSpec{Repeated: true},
	}
	expected := JsonObject{"foo": []string{"bar", "baz"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_query_only_enum(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	body_object := JsonObject{}
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
	expected := JsonObject{"gid": "X"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	if err != nil {
		t.Fail()
	}

	// Bad enum
	query_parameters = "gid=Y"
	expected = JsonObject{"gid": "Y"}
	err = transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, /*method_params=*/method_params)
	if err == nil {
		t.Error("Bad enum should have caused failure.")
	} else if _, ok := err.(*EnumRejectionError); !ok {
		t.Error("Bad enum should have caused failure.")
	} else {
		enum_err := err.(*EnumRejectionError)
		if enum_err.parameter_name != "gid" {
			t.Fail()
		}
	}
}

func test_transform_rest_request_query_only_repeated_enum(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	body_object := JsonObject{}
	enum_descriptor := map[string]*endpoints.ApiEnumParamSpec{
		"X": &endpoints.ApiEnumParamSpec{BackendVal: "X"},
		"Y": &endpoints.ApiEnumParamSpec{BackendVal: "Y"},
	}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"gid": &endpoints.ApiRequestParamSpec{
			Enum: enum_descriptor,
			Repeated: true,
		},
	}

	// Good enum
	query_parameters := "gid=X,Y"
	expected := JsonObject{"gid": []string{"X", "Y"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	if err != nil {
		t.Fail()
	}

	// Bad enum
	query_parameters = "gid=X,Y,Z"
	expected = JsonObject{"gid": []string{"X", "Y", "Z"}}
	err = transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	if err == nil {
		t.Error("Bad enum should have caused failure.")
	} else if _, ok := err.(*EnumRejectionError); !ok {
		t.Error("Bad enum should have caused failure.")
	} else {
		enum_err := err.(*EnumRejectionError)
		if enum_err.parameter_name != "gid[2]" {
			t.Fail()
		}
	}
}

/* Body only. */

func test_transform_rest_request_body_only(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := ""
	body_object := JsonObject{"sample": "body"}
	expected := JsonObject{"sample": "body"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_body_only_any_old_value(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := ""
	body_object := JsonObject{
		"sample": JsonObject{
			"body": []string{"can", "be", "anything"},
		},
	}
	expected := JsonObject{
		"sample": JsonObject{
			"body": []string{"can", "be", "anything"},
		},
	}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_body_only_message_field(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := ""
	body_object := JsonObject{"gid": JsonObject{"val": "X"}}
	expected := JsonObject{"gid": JsonObject{"val": "X"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_body_only_enum(t *testing.T) {
	server := setUpTransformRequestTests()
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
	body_object := JsonObject{"gid": "X"}
	expected := JsonObject{"gid": "X"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	if err != nil {
		t.Fail()
	}

	// Bad enum
	body_object = JsonObject{"gid": "Y"}
	expected = JsonObject{"gid": "Y"}
	err = transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	if err != nil {
		t.Fail()
	}
}

/* Path and query only */

func test_transform_rest_request_path_query_no_collision(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "c=d"
	body_object := JsonObject{}
	expected := JsonObject{"a": "b", "c": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_path_query_collision(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "a=d"
	body_object := JsonObject{}
	expected := JsonObject{"a": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_path_query_collision_in_repeated_param(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "a=d,c"
	body_object := JsonObject{}
	expected := JsonObject{"a": []string{"d", "c", "b"}}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"a": &endpoints.ApiRequestParamSpec{Repeated: true},
	}

	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	if err != nil {
		t.Fail()
	}
}

/* Path and body only. */

func test_transform_rest_request_path_body_no_collision(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := ""
	body_object := JsonObject{"c": "d"}
	expected := JsonObject{"a": "b", "c": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_path_body_collision(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := ""
	body_object := JsonObject{"a": "d"}
	expected := JsonObject{"a": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_path_body_collision_in_repeated_param(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := ""
	body_object := JsonObject{"a": []string{"d"}}
	expected := JsonObject{"a": []string{"d"}}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"a": &endpoints.ApiRequestParamSpec{Repeated: true},
	}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_path_body_message_field_cooperative(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"gid.val1": "X"}
	query_parameters := ""
	body_object := JsonObject{"gid": JsonObject{"val2": "Y"}}
	expected := JsonObject{"gid": JsonObject{"val1": "X", "val2": "Y"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_path_body_message_field_collision(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"gid.val": "X"}
	query_parameters := ""
	body_object := JsonObject{"gid": JsonObject{"val": "Y"}}
	expected := JsonObject{"gid": JsonObject{"val": "Y"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

/* Query and body only */

func test_transform_rest_request_query_body_no_collision(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := "a=b"
	body_object := JsonObject{"c": "d"}
	expected := JsonObject{"a": "b", "c": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_query_body_collision(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := "a=b"
	body_object := JsonObject{"a": "d"}
	expected := JsonObject{"a": "d"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_query_body_collision_in_repeated_param(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := "a=b"
	body_object := JsonObject{"a": []string{"d"}}
	expected := JsonObject{"a": []string{"d"}}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"a": &endpoints.ApiRequestParamSpec{Repeated: true},
	}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_query_body_message_field_cooperative(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := "gid.val1=X"
	body_object := JsonObject{"gid": JsonObject{"val2": "Y"}}
	expected := JsonObject{"gid": JsonObject{"val1": "X", "val2": "Y"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_query_body_message_field_collision(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := make(map[string]string)
	query_parameters := "gid.val=X"
	body_object := JsonObject{"gid": JsonObject{"val": "Y"}}
	expected := JsonObject{"gid": JsonObject{"val": "Y"}}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

/* Path, body and query. */

func test_transform_rest_request_path_query_body_no_collision(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "c=d"
	body_object := JsonObject{"e": "f"}
	expected := JsonObject{"a": "b", "c": "d", "e": "f"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_path_query_body_collision(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "a=d"
	body_object := JsonObject{"a": "f"}
	expected := JsonObject{"a": "f"}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, nil)
	if err != nil {
		t.Fail()
	}
}

func test_transform_rest_request_unknown_parameters(t *testing.T) {
	server := setUpTransformRequestTests()
	path_parameters := map[string]string{"a": "b"}
	query_parameters := "c=d"
	body_object := JsonObject{"e": "f"}
	expected := JsonObject{"a": "b", "c": "d", "e": "f"}
	method_params := map[string]*endpoints.ApiRequestParamSpec{
		"X": &endpoints.ApiRequestParamSpec{},
		"Y": &endpoints.ApiRequestParamSpec{},
	}
	err := transform_rest_request(server, path_parameters, query_parameters,
		body_object, expected, method_params)
	if err != nil {
		t.Fail()
	}
}
