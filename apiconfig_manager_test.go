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
	"github.com/rwl/go-endpoints/endpoints"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseApiConfigEmptyResponse(t *testing.T) {
	configManager := NewApiConfigManager()
	configManager.parseApiConfigResponse("")
	actualMethod := configManager.lookupRpcMethod("guestbook_api.get", "v1")
	assert.Nil(t, actualMethod)
}

func TestParseApiConfigInvalidResponse(t *testing.T) {
	configManager := NewApiConfigManager()
	configManager.parseApiConfigResponse(`{"name": "foo"}`)
	actualMethod := configManager.lookupRpcMethod("guestbook_api.get", "v1")
	assert.Nil(t, actualMethod)
}

func TestParseApiConfig(t *testing.T) {
	configManager := NewApiConfigManager()
	fakeMethod := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path:       "greetings/{gid}",
		RosyMethod: "baz.bim",
	}
	config, _ := json.Marshal(&endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "X",
		Methods: map[string]*endpoints.ApiMethod{
			"guestbook_api.foo.bar": fakeMethod,
		},
	})
	items, _ := json.Marshal(map[string]interface{}{
		"items": []string{string(config)},
	})
	err := configManager.parseApiConfigResponse(string(items))
	assert.NoError(t, err)
	actualMethod := configManager.lookupRpcMethod("guestbook_api.foo.bar", "X")
	assert.Equal(t, fakeMethod, actualMethod)
}

func TestParseApiConfigOrderLength(t *testing.T) {
	configManager := NewApiConfigManager()
	methods := map[string]*endpoints.ApiMethod {
		"guestbook_api.foo.bar": &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "greetings/{gid}",
			RosyMethod: "baz.bim",
		},
		"guestbook_api.list": &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "greetings",
			RosyMethod: "greetings.list",
		},
		"guestbook_api.f3": &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "greetings/{gid}/sender/property/blah",
			RosyMethod: "greetings.f3",
		},
		"guestbook_api.shortgreet": &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "greet",
			RosyMethod: "greetings.short_greeting",
		},
	}
	config, err := json.Marshal(&endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "X",
		Methods: methods,
	})
	assert.NoError(t, err)
	items, err := json.Marshal(map[string]interface{}{
		"items": []string{string(config)},
	})
	assert.NoError(t, err)
	err = configManager.parseApiConfigResponse(string(items))
	assert.NoError(t, err)
	// Make sure all methods appear in the result.
	for methodName, _ := range methods {
		assert.NotNil(t, configManager.lookupRpcMethod(methodName, "X"))
	}

	var mn string
	// Make sure paths and partial paths return the right methods.
	mn, _, _ = configManager.lookupRestMethod("guestbook_api/X/greetings", "GET")
	assert.Equal(t, mn, "guestbook_api.list")

	mn, _, _ = configManager.lookupRestMethod("guestbook_api/X/greetings/1", "GET")
	assert.Equal(t, mn, "guestbook_api.foo.bar")

	mn, _, _ = configManager.lookupRestMethod("guestbook_api/X/greetings/2/sender/property/blah", "GET")
	assert.Equal(t, mn, "guestbook_api.f3")

	mn, _, _ = configManager.lookupRestMethod("guestbook_api/X/greet", "GET")
	assert.Equal(t, mn, "guestbook_api.shortgreet")
}

func TestSortMethods1(t *testing.T) {
	methods := map[string]*endpoints.ApiMethod {
		"name1" : &endpoints.ApiMethod{
			HttpMethod: "POST",
			Path:       "greetings",
		},
		"name2" : &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "greetings",
		},
		"name3" : &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "short/but/many/constants",
		},
		"name4" : &endpoints.ApiMethod{
			HttpMethod: "",
			Path:       "greetings",
		},
		"name5" : &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "greetings/{gid}",
		},
		"name6" : &endpoints.ApiMethod{
			HttpMethod: "PUT",
			Path:       "greetings/{gid}",
		},
		"name7" : &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "a/b/{var}/{var2}",
		},
	}
	sortedMethods := sortMethods(methods)

	expectedMethods := []*methodInfo{
		&methodInfo{
			"name3",
			&endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "short/but/many/constants",
			},
		},
		&methodInfo{
			"name7",
			&endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "a/b/{var}/{var2}",
			},
		},
		&methodInfo{
			"name4",
			&endpoints.ApiMethod{
				HttpMethod: "",
				Path:       "greetings",
			},
		},
		&methodInfo{
			"name2",
			&endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "greetings",
			},
		},
		&methodInfo{
			"name1",
			&endpoints.ApiMethod{
				HttpMethod: "POST",
				Path:       "greetings",
			},
		},
		&methodInfo{
			"name5",
			&endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "greetings/{gid}",
			},
		},
		&methodInfo{
			"name6",
			&endpoints.ApiMethod{
				HttpMethod: "PUT",
				Path:       "greetings/{gid}",
			},
		},
	}
	assert.Equal(t, expectedMethods, sortedMethods)
}

func TestSortMethods2(t *testing.T) {
	methods := map[string]*endpoints.ApiMethod {
		"name1": &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "abcdefghi",
		},
		"name2": &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "foo",
		},
		"name3": &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "greetings",
		},
		"name4": &endpoints.ApiMethod{
			HttpMethod: "POST",
			Path:       "bar",
		},
		"name5": &endpoints.ApiMethod{
			HttpMethod: "GET",
			Path:       "baz",
		},
		"name6": &endpoints.ApiMethod{
			HttpMethod: "PUT",
			Path:       "baz",
		},
		"name7": &endpoints.ApiMethod{
			HttpMethod: "DELETE",
			Path:       "baz",
		},
	}
	sortedMethods := sortMethods(methods)

	// Single-part paths should be sorted by path name, http_method.
	expectedMethods := []*methodInfo {
		&methodInfo{
			"name1",
			&endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "abcdefghi",
			},
		},
		&methodInfo{
			"name4",
			&endpoints.ApiMethod{
				HttpMethod: "POST",
				Path:       "bar",
			},
		},
		&methodInfo{
			"name7",
			&endpoints.ApiMethod{
				HttpMethod: "DELETE",
				Path:       "baz",
			},
		},
		&methodInfo{
			"name5",
			&endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "baz",
			},
		},
		&methodInfo{
			"name6",
			&endpoints.ApiMethod{
				HttpMethod: "PUT",
				Path:       "baz",
			},
		},
		&methodInfo{
			"name2",
			&endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "foo",
			},
		},
		&methodInfo{
			"name3",
			&endpoints.ApiMethod{
				HttpMethod: "GET",
				Path:       "greetings",
			},
		},
	}
	assert.Equal(t, expectedMethods, sortedMethods)
}

func TestParseApiConfigInvalidApiConfig(t *testing.T) {
	configManager := NewApiConfigManager()
	fakeMethod := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path:       "greetings/{gid}",
		RosyMethod: "baz.bim",
	}
	config, _ := json.Marshal(&endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "X",
		Methods: map[string]*endpoints.ApiMethod{
			"guestbook_api.foo.bar": fakeMethod,
		},
	})
	config2 := "{" // Invalid Json.
	items, _ := json.Marshal(map[string]interface{}{
		"items": []string{string(config), string(config2)},
	})
	configManager.parseApiConfigResponse(string(items))
	actualMethod := configManager.lookupRpcMethod("guestbook_api.foo.bar", "X")
	assert.Equal(t, fakeMethod, actualMethod)
}

// Test that the parsed API config has switched HTTPS to HTTP.
func TestParseApiConfigConvertHttps(t *testing.T) {
	configManager := NewApiConfigManager()

	descriptor := &endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "X",
		Root:    "https://localhost/_ah/api",
		Methods: make(map[string]*endpoints.ApiMethod),
	}
	descriptor.Adapter.Bns = "https://localhost/_ah/spi"
	descriptor.Adapter.Type = "lily"

	config, _ := json.Marshal(descriptor)
	items, _ := json.Marshal(map[string]interface{}{
		"items": []string{string(config)},
	})
	configManager.parseApiConfigResponse(string(items))

	key := lookupKey{"guestbook_api", "X"}
	assert.Equal(t, "http://localhost/_ah/spi", configManager.configs[key].Adapter.Bns)
	assert.Equal(t, "http://localhost/_ah/api", configManager.configs[key].Root)
}

// Test that the convertHttpsToHttp function works.
func TestConvertHttpsToHttp(t *testing.T) {
	config := &endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "X",
		Root:    "https://tictactoe.appspot.com/_ah/api",
		Methods: make(map[string]*endpoints.ApiMethod),
	}
	config.Adapter.Bns = "https://tictactoe.appspot.com/_ah/spi"
	config.Adapter.Type = "lily"

	convertHttpsToHttp(config)

	assert.Equal(t, "http://tictactoe.appspot.com/_ah/spi", config.Adapter.Bns)
	assert.Equal(t, "http://tictactoe.appspot.com/_ah/api", config.Root)
}

// Verify that we don't change non-HTTPS URLs.
func TestDontConvertNonHttpsToHttp(t *testing.T) {
	config := &endpoints.ApiDescriptor{
		Name:    "guestbook_api",
		Version: "X",
		Root:    "ios://https.appspot.com/_ah/api",
		Methods: make(map[string]*endpoints.ApiMethod),
	}
	config.Adapter.Bns = "http://https.appspot.com/_ah/spi"
	config.Adapter.Type = "lily"

	convertHttpsToHttp(config)

	assert.Equal(t, "http://https.appspot.com/_ah/spi", config.Adapter.Bns)
	assert.Equal(t, "ios://https.appspot.com/_ah/api", config.Root)
}

func TestSaveLookupRpcMethod(t *testing.T) {
	configManager := NewApiConfigManager()
	// First attempt, guestbook.get does not exist
	actualMethod := configManager.lookupRpcMethod("guestbook_api.get", "v1")
	assert.Nil(t, actualMethod)

	// Now we manually save it, and should find it
	fakeMethod := &endpoints.ApiMethod{}
	configManager.saveRpcMethod("guestbook_api.get", "v1", fakeMethod)
	actualMethod = configManager.lookupRpcMethod("guestbook_api.get", "v1")
	assert.Equal(t, fakeMethod, actualMethod)
}

func TestSaveLookupRestMethod(t *testing.T) {
	configManager := NewApiConfigManager()
	// First attempt, guestbook.get does not exist
	methodName, apiMethod, params := configManager.lookupRestMethod("guestbook_api/v1/greetings/i", "GET")
	assert.Empty(t, methodName)
	assert.Nil(t, apiMethod)
	assert.Nil(t, params)

	// Now we manually save it, and should find it
	fakeMethod := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path:       "greetings/{id}",
	}
	configManager.saveRestMethod("guestbook_api.get", "guestbook_api", "v1", fakeMethod)
	methodName, apiMethod, params = configManager.lookupRestMethod("guestbook_api/v1/greetings/i", "GET")
	assert.Equal(t, "guestbook_api.get", methodName)
	assert.Equal(t, fakeMethod, apiMethod)
	assert.Equal(t, map[string]string{"id": "i"}, params)
}

func TestTrailingSlashOptional(t *testing.T) {
	configManager := NewApiConfigManager()
	// Create a typical get resource URL.
	fakeMethod := &endpoints.ApiMethod{
		HttpMethod: "GET",
		Path:       "trailingslash",
	}
	configManager.saveRestMethod("guestbook_api.trailingslash", "guestbook_api", "v1", fakeMethod)

	// Make sure we get this method when we query without a slash.
	methodName, methodSpec, params := configManager.lookupRestMethod("guestbook_api/v1/trailingslash", "GET")
	assert.Equal(t, "guestbook_api.trailingslash", methodName)
	assert.Equal(t, fakeMethod, methodSpec)
	assert.Equal(t, params, make(map[string]string))

	// Make sure we get this method when we query with a slash.
	methodName, methodSpec, params = configManager.lookupRestMethod("guestbook_api/v1/trailingslash/", "GET")
	assert.Equal(t, "guestbook_api.trailingslash", methodName)
	assert.Equal(t, fakeMethod, methodSpec)
	assert.Equal(t, params, make(map[string]string))
}

// Parameterized path tests.

func TestInvalidVariableNameLeadingDigit(t *testing.T) {
	matched := PathVariablePattern.MatchString("1abc")
	assert.False(t, matched)
}

// Ensure users can not add variables starting with !
// This is used for reserved variables (e.g. !name and !version)
func TestInvalidVarNameLeadingExclamation(t *testing.T) {
	matched := PathVariablePattern.MatchString("!abc")
	assert.False(t, matched)
}

func TestValidVariableName(t *testing.T) {
	assert.Equal(t, PathVariablePattern.FindString("AbC1"), "AbC1")
}

// Assert that the given inbound request path does not match the
// parameterized path pattern.
//
// For example, /xyz/123 does not match /abc/{x}.
func assertNoMatch(t *testing.T, path, paramPath string) {
	re, err := compilePathPattern(paramPath)
	assert.NoError(t, err)
	params := re.MatchString(path)
	assert.False(t, params)
}

func TestPrefixNoMatch(t *testing.T) {
	assertNoMatch(t, "/xyz/123", "/abc/{x}")
}

func TestSuffixNoMatch(t *testing.T) {
	assertNoMatch(t, "/abc/123", "/abc/{x}/456")
}

func TestSuffixNoMatchWithMoreVariables(t *testing.T) {
	assertNoMatch(t, "/abc/456/123/789", "/abc/{x}/123/{y}/xyz")
}

func TestNoMatchCollectionWithItem(t *testing.T) {
	assertNoMatch(t, "/api/v1/resources/123", "/{name}/{version}/resources")
}

// Assert that the given inbound request path does match the parameterized
// path pattern.
//
// For example, /abc/123 does match /abc/{x}.
//
// Also checks against the expected number of parameters.
func assertMatch(t *testing.T, path, paramPath string, paramCount int) map[string]string {
	re, err := compilePathPattern(paramPath)
	ok := assert.NoError(t, err)
	if ok {
		match := re.MatchString(path)
		assert.True(t, match) // Will be None if path was not matched.
		names := re.SubexpNames()
		submatch := re.FindStringSubmatch(path)
		params, err := pathParams(names, submatch)
		assert.NoError(t, err)
		assert.Equal(t, paramCount, len(params))
		return params
	}
	return make(map[string]string)
}

func TestOneVariableMatch(t *testing.T) {
	params := assertMatch(t, "/abc/123", "/abc/{x}", 1)
	x, ok := params["x"]
	assert.True(t, ok)
	assert.Equal(t, "123", x)
}

func TestTwoVariableMatch(t *testing.T) {
	params := assertMatch(t, "/abc/456/123/789", "/abc/{x}/123/{y}", 2)
	x, ok := params["x"]
	assert.True(t, ok)
	assert.Equal(t, x, "456")
	y, ok := params["y"]
	assert.True(t, ok)
	assert.Equal(t, y, "789")
}

func TestMessageVariableMatch(t *testing.T) {
	params := assertMatch(t, "/abc/123", "/abc/{x.y}", 1)
	xy, ok := params["x.y"]
	assert.True(t, ok)
	assert.Equal(t, xy, "123")
}

func TestMessageAndSimpleVariableMatch(t *testing.T) {
	params := assertMatch(t, "/abc/123/456", "/abc/{x.y.z}/{t}", 2)
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
func assertInvalidValue(t *testing.T, value string) {
	paramPath := "/abc/{x}"
	path := fmt.Sprintf("/abc/%s", value)
	re, err := compilePathPattern(paramPath)
	assert.NoError(t, err)
	params := re.MatchString(path)
	assert.False(t, params)
}

func TestInvalidValues(t *testing.T) {
	reserved := []string{":", "?", "#", "[", "]", "{", "}"}
	for _, r := range reserved {
		assertInvalidValue(t, fmt.Sprintf("123%s", r))
	}
}
