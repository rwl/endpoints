
package discovery

import (
	"testing"
	"encoding/json"
	"regexp"
	"fmt"
	"github.com/crhym3/go-endpoints/endpoints"
	"reflect"
)

func test_parse_api_config_empty_response(t *testing.T) {
	config_manager := NewApiConfigManager()
	config_manager.parse_api_config_response("")
	actual_method := config_manager.lookup_rpc_method("guestbook_api.get", "v1")
	if actual_method != nil {
		t.Fail()
	}
}

func test_parse_api_config_invalid_response(t *testing.T) {
	config_manager := NewApiConfigManager()
	config_manager.parse_api_config_response(`{"name": "foo"}`)
	actual_method := config_manager.lookup_rpc_method("guestbook_api.get", "v1")
	if actual_method != nil {
		t.Fail()
	}
}

func test_parse_api_config(t *testing.T) {
	config_manager := NewApiConfigManager()
	fake_method := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path: "greetings/{gid}",
		RosyMethod: "baz.bim",
	}
	config, _ := json.Marshal(JsonObject{
		"name": "guestbook_api",
		"version": "X",
		"methods": JsonObject{
			"guestbook_api.foo.bar": fake_method,
		},
	})
	items, _ := json.Marshal(JsonObject{
		"items": []string{string(config)},
	})
	config_manager.parse_api_config_response(string(items))
	actual_method := config_manager.lookup_rpc_method("guestbook_api.foo.bar", "X")
	if !reflect.DeepEqual(fake_method, actual_method) {
		t.Fail()
	}
}

type method_info struct {
	method_name string
	path string
	method string
}

func test_parse_api_config_order_length(t *testing.T) {
	config_manager := NewApiConfigManager()
	test_method_info := []method_info{
		method_info{"guestbook_api.foo.bar", "greetings/{gid}", "baz.bim"},
		method_info{"guestbook_api.list", "greetings", "greetings.list"},
		method_info{"guestbook_api.f3", "greetings/{gid}/sender/property/blah", "greetings.f3"},
		method_info{"guestbook_api.shortgreet", "greet", "greetings.short_greeting"},
	}
	methods := make(map[string]map[string]string)
	for _, mi := range test_method_info {
		method := map[string]string{
			"httpMethod": "GET",
			"path": mi.path,
			"rosyMethod": mi.method,
		}
		methods[mi.method_name] = method
	}
	config, _ := json.Marshal(JsonObject{
			"name": "guestbook_api",
			"version": "X",
			"methods": methods,
	})
	items, _ := json.Marshal(JsonObject{
		"items": []string{string(config)},
	})
	config_manager.parse_api_config_response(string(items))
	// Make sure all methods appear in the result.
	for _, mi := range test_method_info {
		if config_manager.lookup_rpc_method(mi.method_name, "X") == nil {
			t.Fail()
		}
		var mn string
		// Make sure paths and partial paths return the right methods.
		mn, _, _ = config_manager.lookup_rest_method("guestbook_api/X/greetings", "GET")
		if mn != "guestbook_api.list" {
			t.Fail()
		}
		mn, _, _ = config_manager.lookup_rest_method("guestbook_api/X/greetings/1", "GET")
		if mn != "guestbook_api.foo.bar" {
			t.Fail()
		}
		mn, _, _ = config_manager.lookup_rest_method("guestbook_api/X/greetings/2/sender/property/blah", "GET")
		if mn != "guestbook_api.f3" {
			t.Fail()
		}
		mn, _, _ = config_manager.lookup_rest_method("guestbook_api/X/greet", "GET")
		if mn != "guestbook_api.shortgreet" {
			t.Fail()
		}
	}
}

func test_get_sorted_methods1(t *testing.T) {
	test_method_info := []method_info{
		method_info{"name1", "greetings", "POST"},
		method_info{"name2", "greetings", "GET"},
		method_info{"name3", "short/but/many/constants", "GET"},
		method_info{"name4", "greetings", ""},
		method_info{"name5", "greetings/{gid}", "GET"},
		method_info{"name6", "greetings/{gid}", "PUT"},
		method_info{"name7", "a/b/{var}/{var2}", "GET"},
	}
	methods := make(map[string]*endpoints.ApiMethod)
	for _, mi := range test_method_info {
		method := &endpoints.ApiMethod{
			HttpMethod: mi.method,
			Path: mi.path,
		}
		methods[mi.method_name] = method
	}
	sorted_methods := get_sorted_methods(methods)

	expected_data := []method_info{
		method_info{"name3", "short/but/many/constants", "GET"},
		method_info{"name7", "a/b/{var}/{var2}", "GET"},
		method_info{"name4", "greetings", ""},
		method_info{"name2", "greetings", "GET"},
		method_info{"name1", "greetings", "POST"},
		method_info{"name5", "greetings/{gid}", "GET"},
		method_info{"name6", "greetings/{gid}", "PUT"},
	}
	expected_methods := make([]*methodInfo, len(expected_data))
	for i, mi := range expected_data {
		expected_methods[i] = &methodInfo{
			mi.method_name,
			&endpoints.ApiMethod{
				HttpMethod: mi.method,
				Path: mi.path,
			},
		}
	}
	if !reflect.DeepEqual(expected_methods, sorted_methods) {
		t.Fail()
	}
}

func test_get_sorted_methods2(t *testing.T) {
	test_method_info := []method_info{
		method_info{"name1", "abcdefghi", "GET"},
		method_info{"name2", "foo", "GET"},
		method_info{"name3", "greetings", "GET"},
		method_info{"name4", "bar", "POST"},
		method_info{"name5", "baz", "GET"},
		method_info{"name6", "baz", "PUT"},
		method_info{"name7", "baz", "DELETE"},
	}
	methods := make(map[string]*endpoints.ApiMethod)
	for _, mi := range test_method_info {
		method := &endpoints.ApiMethod{
			HttpMethod: mi.method,
			Path: mi.path,
		}
		methods[mi.method_name] = method
	}
	sorted_methods := get_sorted_methods(methods)

	// Single-part paths should be sorted by path name, http_method.
	expected_data := []method_info{
		method_info{"name1", "abcdefghi", "GET"},
		method_info{"name4", "bar", "POST"},
		method_info{"name7", "baz", "DELETE"},
		method_info{"name5", "baz", "GET"},
		method_info{"name6", "baz", "PUT"},
		method_info{"name2", "foo", "GET"},
		method_info{"name3", "greetings", "GET"},
	}
	expected_methods := make([]*methodInfo, len(expected_data))
	for i, mi := range expected_data {
		expected_methods[i] = &methodInfo{
			mi.method_name,
			&endpoints.ApiMethod{
				HttpMethod: mi.method,
				Path: mi.path,
			},
		}
	}
	if !reflect.DeepEqual(expected_methods, sorted_methods) {
		t.Fail()
	}
}

func test_parse_api_config_invalid_api_config(t *testing.T) {
	config_manager := NewApiConfigManager()
	fake_method := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path: "greetings/{gid}",
		RosyMethod: "baz.bim",
	}
	config, _ := json.Marshal(JsonObject{
		"name": "guestbook_api",
		"version": "X",
		"methods": JsonObject{
			"guestbook_api.foo.bar": fake_method,
		},
	})
	// Invalid Json.
	config2 := "{"
	items, _ := json.Marshal(JsonObject{
		"items": []string{string(config), string(config2)},
	})
	config_manager.parse_api_config_response(string(items))
	actual_method := config_manager.lookup_rpc_method("guestbook_api.foo.bar", "X")
	if fake_method != actual_method {
		t.Fail()
	}
}

// Test that the parsed API config has switched HTTPS to HTTP.
func test_parse_api_config_convert_https(t *testing.T) {
	config_manager := NewApiConfigManager()
	config, _ := json.Marshal(JsonObject{
		"name": "guestbook_api",
		"version": "X",
		"adapter": JsonObject{
			"bns": "https://localhost/_ah/spi",
			"type": "lily",
		},
		"root": "https://localhost/_ah/api",
		"methods": JsonObject{},
	})
	items, _ := json.Marshal(JsonObject{
		"items": []string{string(config)},
	})
	config_manager.parse_api_config_response(string(items))

	key := lookupKey{"guestbook_api", "X"}
	if "http://localhost/_ah/spi" != config_manager.configs[key].Adapter.Bns {
		t.Fail()
	}
	if "http://localhost/_ah/api" != config_manager.configs[key].Root {
		t.Fail()
	}
}

// Test that the _convert_https_to_http function works.
func test_convert_https_to_http(t *testing.T) {
	config := &endpoints.ApiDescriptor{
		Name: "guestbook_api",
		Version: "X",
		Root: "https://tictactoe.appspot.com/_ah/api",
		Methods: make(map[string]*endpoints.ApiMethod),
	}
	config.Adapter.Bns = "https://tictactoe.appspot.com/_ah/spi"
	config.Adapter.Type = "lily"

	convert_https_to_http(config)

	if "http://tictactoe.appspot.com/_ah/spi" != config.Adapter.Bns {
		t.Fail()
	}
	if "http://tictactoe.appspot.com/_ah/api" != config.Root {
		t.Fail()
	}
}

// Verify that we don"t change non-HTTPS URLs.
func test_dont_convert_non_https_to_http(t *testing.T) {
	config := &endpoints.ApiDescriptor{
		Name: "guestbook_api",
		Version: "X",
		Root: "ios://https.appspot.com/_ah/api",
		Methods: make(map[string]*endpoints.ApiMethod),
	}
	config.Adapter.Bns = "http://https.appspot.com/_ah/spi"
	config.Adapter.Type = "lily"

	convert_https_to_http(config)

	if "http://https.appspot.com/_ah/spi" != config.Adapter.Bns {
		t.Fail()
	}
	if "ios://https.appspot.com/_ah/api" != config.Root {
		t.Fail()
	}
}

func test_save_lookup_rpc_method(t *testing.T) {
	config_manager := NewApiConfigManager()
	// First attempt, guestbook.get does not exist
	actual_method := config_manager.lookup_rpc_method("guestbook_api.get", "v1")
	if actual_method != nil {
		t.Fail()
	}

	// Now we manually save it, and should find it
	fake_method := &endpoints.ApiMethod{}
	config_manager.save_rpc_method("guestbook_api.get", "v1", fake_method)
	actual_method = config_manager.lookup_rpc_method("guestbook_api.get", "v1")
	if fake_method != actual_method {
		t.Fail()
	}
}

func test_save_lookup_rest_method(t *testing.T) {
	config_manager := NewApiConfigManager()
	// First attempt, guestbook.get does not exist
	method_name, api_method, params := config_manager.lookup_rest_method("guestbook_api/v1/greetings/i", "GET")
	if len(method_name) > 0 || api_method != nil || params != nil {
		t.Fail()
	}

	// Now we manually save it, and should find it
	fake_method := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path: "greetings/{id}",
	}
	config_manager.save_rest_method("guestbook_api.get", "guestbook_api", "v1", fake_method)
	method_name, api_method, params = config_manager.lookup_rest_method("guestbook_api/v1/greetings/i", "GET")
	if "guestbook_api.get" != method_name {
		t.Fail()
	}
	if fake_method != api_method {
		t.Fail()
	}
	if !reflect.DeepEqual(map[string]string{"id": "i"}, params) {
		t.Fail()
	}
}

func test_trailing_slash_optional(t *testing.T) {
	config_manager := NewApiConfigManager()
	// Create a typical get resource URL.
	fake_method := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path: "trailingslash",
	}
	config_manager.save_rest_method("guestbook_api.trailingslash", "guestbook_api", "v1", fake_method)

	// Make sure we get this method when we query without a slash.
	method_name, method_spec, params := config_manager.lookup_rest_method("guestbook_api/v1/trailingslash", "GET")
	if "guestbook_api.trailingslash" != method_name {
		t.Fail()
	}
	if fake_method != method_spec {
		t.Fail()
	}
	if len(params) > 0 {
		t.Fail()
	}

	// Make sure we get this method when we query with a slash.
	method_name, method_spec, params = config_manager.lookup_rest_method("guestbook_api/v1/trailingslash/", "GET")
	if "guestbook_api.trailingslash" != method_name {
		t.Fail()
	}
	if fake_method != method_spec {
		t.Fail()
	}
	if len(params) > 0 {
		t.Fail()
	}
}


/* Parameterized path tests. */

func test_invalid_variable_name_leading_digit(t *testing.T) {
	matched, _ := regexp.MatchString(_PATH_VARIABLE_PATTERN, "1abc")
	if matched {
		t.Fail()
	}
}

// Ensure users can not add variables starting with !
// This is used for reserved variables (e.g. !name and !version)
func test_invalid_var_name_leading_exclamation(t *testing.T) {
	matched, _ := regexp.MatchString(_PATH_VARIABLE_PATTERN, "!abc")
	if matched {
		t.Fail()
	}
}

func test_valid_variable_name(t *testing.T) {
	re := regexp.MustCompile(_PATH_VARIABLE_PATTERN)
	if re.FindString("AbC1") != "AbC1" {
		t.Fail()
	}
}

// Assert that the given path does not match param_path pattern.
//
// For example, /xyz/123 does not match /abc/{x}.
//
// Args:
//   path: A string, the inbound request path.
//   param_path: A string, the parameterized path pattern to match against
//     this path.
func assert_no_match(t *testing.T, path, param_path string) {
	config_manager := NewApiConfigManager()
	re, err := compile_path_pattern(param_path)
	if err != nil {
		t.Fail()
	}
	params := re.MatchString(path)
	if params {
		t.Fail()
	}
}

func test_prefix_no_match(t *testing.T) {
	assert_no_match(t, "/xyz/123", "/abc/{x}")
}

func test_suffix_no_match(t *testing.T) {
	assert_no_match(t, "/abc/123", "/abc/{x}/456")
}

func test_suffix_no_match_with_more_variables(t *testing.T) {
	assert_no_match(t, "/abc/456/123/789", "/abc/{x}/123/{y}/xyz")
}

func test_no_match_collection_with_item(t *testing.T) {
	assert_no_match(t, "/api/v1/resources/123", "/{name}/{version}/resources")
}

// Assert that the given path does match param_path pattern.
//
// For example, /abc/123 does not match /abc/{x}.
//
// Args:
//   path: A string, the inbound request path.
//   param_path: A string, the parameterized path pattern to match against
//     this path.
//   param_count: An int, the expected number of parameters to match in
//     pattern.
//
// Returns:
//   Dict mapping path variable name to path variable value.
func assert_match(t *testing.T, path, param_path string, param_count int) map[string]string {
	config_manager := NewApiConfigManager()
	re, err := compile_path_pattern(param_path)
	if err != nil {
		t.Fail()
	}
	match := re.MatchString(path)
	if !match {  // Will be None if path was not matched.
		t.Fail()
	}
	names := re.SubexpNames()
	submatch := re.FindStringSubmatch(path)
	params, err := get_path_params(names, submatch)
	if err != nil {
		t.Fail()
	}
	if param_count != len(params) {
		t.Fail()
	}
	return params
}

func test_one_variable_match(t *testing.T) {
	params := assert_match(t, "/abc/123", "/abc/{x}", 1)
	if x, ok := params["x"]; !ok || "123" != x {
		t.Fail()
	}
}

func test_two_variable_match(t *testing.T) {
	params := assert_match(t, "/abc/456/123/789", "/abc/{x}/123/{y}", 2)
	if x, ok := params["x"]; !ok || x != "456" {
		t.Fail()
	}
	if y, ok := params["y"]; !ok || y != "789" {
		t.Fail()
	}
}

func test_message_variable_match(t *testing.T) {
	params := assert_match(t, "/abc/123", "/abc/{x.y}", 1)
	if xy, ok := params["x.y"]; !ok || xy != "123" {
		t.Fail()
	}
}

func test_message_and_simple_variable_match(t *testing.T) {
	params := assert_match(t, "/abc/123/456", "/abc/{x.y.z}/{t}", 2)
	if xyz, ok := params["x.y.z"]; !ok || xyz != "123" {
		t.Fail()
	}
	if _t, ok := params["t"]; !ok || _t != "456" {
		t.Fail()
	}
}

// Assert that the path parameter value is not valid.
//
// For example, /abc/3!:2 is invalid for /abc/{x}.
//
// Args:
//   value: A string containing a variable value to check for validity.
func assert_invalid_value(t *testing.T, value string) {
	param_path := "/abc/{x}"
	path := fmt.Sprintf("/abc/%s", value)
	config_manager := NewApiConfigManager()
	re, err := compile_path_pattern(param_path)
	if err != nil {
		t.Fail()
	}
	params := re.MatchString(path)
	if params {
		t.Fail()
	}
}

func test_invalid_values(t *testing.T) {
	reserved := []string{":", "?", "#", "[", "]", "{", "}"}
	for _, r := range reserved {
		assert_invalid_value(t, fmt.Sprintf("123%s", reserved))
	}
}
