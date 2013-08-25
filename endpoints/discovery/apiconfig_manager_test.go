
package discovery

import (
	"testing"
	"encoding/json"
	"regexp"
	"fmt"
	"github.com/crhym3/go-endpoints/endpoints"
	"github.com/stretchr/testify/assert"
)

func Test_parse_api_config_empty_response(t *testing.T) {
	config_manager := NewApiConfigManager()
	config_manager.parse_api_config_response("")
	actual_method := config_manager.lookup_rpc_method("guestbook_api.get", "v1")
	assert.Nil(t, actual_method)
}

func Test_parse_api_config_invalid_response(t *testing.T) {
	config_manager := NewApiConfigManager()
	config_manager.parse_api_config_response(`{"name": "foo"}`)
	actual_method := config_manager.lookup_rpc_method("guestbook_api.get", "v1")
	assert.Nil(t, actual_method)
}

func Test_parse_api_config(t *testing.T) {
	config_manager := NewApiConfigManager()
	fake_method := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path: "greetings/{gid}",
		RosyMethod: "baz.bim",
	}
	config, _ := json.Marshal(&endpoints.ApiDescriptor{
		Name: "guestbook_api",
		Version: "X",
		Methods: map[string]*endpoints.ApiMethod{
			"guestbook_api.foo.bar": fake_method,
		},
	})
	items, _ := json.Marshal(JsonObject{
		"items": []string{string(config)},
	})
	err := config_manager.parse_api_config_response(string(items))
	assert.NoError(t, err)
	actual_method := config_manager.lookup_rpc_method("guestbook_api.foo.bar", "X")
	assert.Equal(t, fake_method, actual_method)
}

type method_info struct {
	method_name string
	path string
	method string
}

func Test_parse_api_config_order_length(t *testing.T) {
	config_manager := NewApiConfigManager()
	test_method_info := []method_info{
		method_info{"guestbook_api.foo.bar", "greetings/{gid}", "baz.bim"},
		method_info{"guestbook_api.list", "greetings", "greetings.list"},
		method_info{"guestbook_api.f3", "greetings/{gid}/sender/property/blah", "greetings.f3"},
		method_info{"guestbook_api.shortgreet", "greet", "greetings.short_greeting"},
	}
	methods := make(map[string]*endpoints.ApiMethod)
	for _, mi := range test_method_info {
		method := &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path: mi.path,
			RosyMethod: mi.method,
		}
		methods[mi.method_name] = method
	}
	config, _ := json.Marshal(&endpoints.ApiDescriptor{
			Name: "guestbook_api",
			Version: "X",
			Methods: methods,
	})
	items, _ := json.Marshal(JsonObject{
		"items": []string{string(config)},
	})
	config_manager.parse_api_config_response(string(items))
	// Make sure all methods appear in the result.
	for _, mi := range test_method_info {
		assert.NotNil(t, config_manager.lookup_rpc_method(mi.method_name, "X"))

		var mn string
		// Make sure paths and partial paths return the right methods.
		mn, _, _ = config_manager.lookup_rest_method("guestbook_api/X/greetings", "GET")
		assert.Equal(t, mn, "guestbook_api.list")

		mn, _, _ = config_manager.lookup_rest_method("guestbook_api/X/greetings/1", "GET")
		assert.Equal(t, mn, "guestbook_api.foo.bar")

		mn, _, _ = config_manager.lookup_rest_method("guestbook_api/X/greetings/2/sender/property/blah", "GET")
		assert.Equal(t, mn, "guestbook_api.f3")

		mn, _, _ = config_manager.lookup_rest_method("guestbook_api/X/greet", "GET")
		assert.Equal(t, mn, "guestbook_api.shortgreet")
	}
}

func Test_get_sorted_methods1(t *testing.T) {
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

	for _, s := range sorted_methods {
		fmt.Printf("%s : %v\n", s.method_name, s.api_method)
	}

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

//		fmt.Printf("%s : %v\n", mi.method_name, expected_methods[i].api_method)
	}
	assert.Equal(t, expected_methods, sorted_methods)
}

func Test_get_sorted_methods2(t *testing.T) {
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
	assert.Equal(t, expected_methods, sorted_methods)
}

func Test_parse_api_config_invalid_api_config(t *testing.T) {
	config_manager := NewApiConfigManager()
	fake_method := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path: "greetings/{gid}",
		RosyMethod: "baz.bim",
	}
	config, _ := json.Marshal(&endpoints.ApiDescriptor{
		Name: "guestbook_api",
		Version: "X",
		Methods: map[string]*endpoints.ApiMethod{
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
	assert.Equal(t, fake_method, actual_method)
}

// Test that the parsed API config has switched HTTPS to HTTP.
func Test_parse_api_config_convert_https(t *testing.T) {
	config_manager := NewApiConfigManager()

	descriptor := &endpoints.ApiDescriptor{
		Name: "guestbook_api",
		Version: "X",
		Root: "https://localhost/_ah/api",
		Methods: make(map[string]*endpoints.ApiMethod),
	}
	descriptor.Adapter.Bns = "https://localhost/_ah/spi"
	descriptor.Adapter.Type = "lily"

	config, _ := json.Marshal(descriptor)
	items, _ := json.Marshal(JsonObject{
		"items": []string{string(config)},
	})
	config_manager.parse_api_config_response(string(items))

	key := lookupKey{"guestbook_api", "X"}
	assert.Equal(t, "http://localhost/_ah/spi", config_manager.configs[key].Adapter.Bns)
	assert.Equal(t, "http://localhost/_ah/api", config_manager.configs[key].Root)
}

// Test that the _convert_https_to_http function works.
func Test_convert_https_to_http(t *testing.T) {
	config := &endpoints.ApiDescriptor{
		Name: "guestbook_api",
		Version: "X",
		Root: "https://tictactoe.appspot.com/_ah/api",
		Methods: make(map[string]*endpoints.ApiMethod),
	}
	config.Adapter.Bns = "https://tictactoe.appspot.com/_ah/spi"
	config.Adapter.Type = "lily"

	convert_https_to_http(config)

	assert.Equal(t, "http://tictactoe.appspot.com/_ah/spi", config.Adapter.Bns)
	assert.Equal(t, "http://tictactoe.appspot.com/_ah/api", config.Root)
}

// Verify that we don"t change non-HTTPS URLs.
func Test_dont_convert_non_https_to_http(t *testing.T) {
	config := &endpoints.ApiDescriptor{
		Name: "guestbook_api",
		Version: "X",
		Root: "ios://https.appspot.com/_ah/api",
		Methods: make(map[string]*endpoints.ApiMethod),
	}
	config.Adapter.Bns = "http://https.appspot.com/_ah/spi"
	config.Adapter.Type = "lily"

	convert_https_to_http(config)

	assert.Equal(t, "http://https.appspot.com/_ah/spi", config.Adapter.Bns)
	assert.Equal(t, "ios://https.appspot.com/_ah/api", config.Root)
}

func Test_save_lookup_rpc_method(t *testing.T) {
	config_manager := NewApiConfigManager()
	// First attempt, guestbook.get does not exist
	actual_method := config_manager.lookup_rpc_method("guestbook_api.get", "v1")
	assert.Nil(t, actual_method)

	// Now we manually save it, and should find it
	fake_method := &endpoints.ApiMethod{}
	config_manager.save_rpc_method("guestbook_api.get", "v1", fake_method)
	actual_method = config_manager.lookup_rpc_method("guestbook_api.get", "v1")
	assert.Equal(t, fake_method, actual_method)
}

func Test_save_lookup_rest_method(t *testing.T) {
	config_manager := NewApiConfigManager()
	// First attempt, guestbook.get does not exist
	method_name, api_method, params := config_manager.lookup_rest_method("guestbook_api/v1/greetings/i", "GET")
	assert.Empty(t, method_name)
	assert.Nil(t, api_method)
	assert.Nil(t, params)

	// Now we manually save it, and should find it
	fake_method := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path: "greetings/{id}",
	}
	config_manager.save_rest_method("guestbook_api.get", "guestbook_api", "v1", fake_method)
	method_name, api_method, params = config_manager.lookup_rest_method("guestbook_api/v1/greetings/i", "GET")
	assert.Equal(t, "guestbook_api.get", method_name)
	assert.Equal(t, fake_method, api_method)
	assert.Equal(t, map[string]string{"id": "i"}, params)
}

func Test_trailing_slash_optional(t *testing.T) {
	config_manager := NewApiConfigManager()
	// Create a typical get resource URL.
	fake_method := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path: "trailingslash",
	}
	config_manager.save_rest_method("guestbook_api.trailingslash", "guestbook_api", "v1", fake_method)

	// Make sure we get this method when we query without a slash.
	method_name, method_spec, params := config_manager.lookup_rest_method("guestbook_api/v1/trailingslash", "GET")
	assert.Equal(t, "guestbook_api.trailingslash", method_name)
	assert.Equal(t, fake_method, method_spec)
	assert.Empty(t, params)

	// Make sure we get this method when we query with a slash.
	method_name, method_spec, params = config_manager.lookup_rest_method("guestbook_api/v1/trailingslash/", "GET")
	assert.Equal(t, "guestbook_api.trailingslash", method_name)
	assert.Equal(t, fake_method, method_spec)
	assert.Empty(t, params)
}


/* Parameterized path tests. */

func Test_invalid_variable_name_leading_digit(t *testing.T) {
	matched, _ := regexp.MatchString(_PATH_VARIABLE_PATTERN, "1abc")
	assert.False(t, matched)
}

// Ensure users can not add variables starting with !
// This is used for reserved variables (e.g. !name and !version)
func Test_invalid_var_name_leading_exclamation(t *testing.T) {
	matched, _ := regexp.MatchString(_PATH_VARIABLE_PATTERN, "!abc")
	assert.False(t, matched)
}

func Test_valid_variable_name(t *testing.T) {
	re := regexp.MustCompile(_PATH_VARIABLE_PATTERN)
	assert.Equal(t, re.FindString("AbC1"), "AbC1")
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
	re, err := compile_path_pattern(param_path)
	assert.NoError(t, err)
	params := re.MatchString(path)
	assert.False(t, params)
}

func Test_prefix_no_match(t *testing.T) {
	assert_no_match(t, "/xyz/123", "/abc/{x}")
}

func Test_suffix_no_match(t *testing.T) {
	assert_no_match(t, "/abc/123", "/abc/{x}/456")
}

func Test_suffix_no_match_with_more_variables(t *testing.T) {
	assert_no_match(t, "/abc/456/123/789", "/abc/{x}/123/{y}/xyz")
}

func Test_no_match_collection_with_item(t *testing.T) {
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
	re, err := compile_path_pattern(param_path)
	assert.Nil(t, err)
	match := re.MatchString(path)
	assert.True(t, match) // Will be None if path was not matched.
	names := re.SubexpNames()
	submatch := re.FindStringSubmatch(path)
	params, err := get_path_params(names, submatch)
	assert.NoError(t, err)
	assert.Equal(t, param_count, len(params))
	return params
}

func Test_one_variable_match(t *testing.T) {
	params := assert_match(t, "/abc/123", "/abc/{x}", 1)
	x, ok := params["x"]
	assert.True(t, ok)
	assert.Equal(t, "123", x)
}

func Test_two_variable_match(t *testing.T) {
	params := assert_match(t, "/abc/456/123/789", "/abc/{x}/123/{y}", 2)
	x, ok := params["x"]
	assert.True(t, ok)
	assert.Equal(t, x, "456")
	y, ok := params["y"]
	assert.True(t, ok)
	assert.Equal(t, y, "789")
}

func Test_message_variable_match(t *testing.T) {
	params := assert_match(t, "/abc/123", "/abc/{x.y}", 1)
	xy, ok := params["x.y"]
	assert.True(t, ok)
	assert.Equal(t, xy, "123")
}

func Test_message_and_simple_variable_match(t *testing.T) {
	params := assert_match(t, "/abc/123/456", "/abc/{x.y.z}/{t}", 2)
	xyz, ok := params["x.y.z"]
	assert.True(t, ok)
	assert.Equal(t, xyz, "123")
	_t, ok := params["t"]
	assert.True(t, ok)
	assert.Equal(t, _t, "456")
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
	re, err := compile_path_pattern(param_path)
	assert.NoError(t, err)
	params := re.MatchString(path)
	assert.False(t, params)
}

func Test_invalid_values(t *testing.T) {
	reserved := []string{":", "?", "#", "[", "]", "{", "}"}
	for _, r := range reserved {
		assert_invalid_value(t, fmt.Sprintf("123%s", r))
	}
}
