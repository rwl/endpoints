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
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"github.com/golang/glog"
	"regexp"
	"strings"
	"sync"
)

const pathValuePattern = `[^:/?#\[\]{}]*`

var pathVariablePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z_.\d]*`)

// Configuration manager to store API configurations.
//
// Manages loading api configs and method lookup.
type apiConfigManager struct {
	rpcMethods map[lookupKey]*endpoints.ApiMethod
	restMethods    []*restMethod
	configs         map[lookupKey]*endpoints.ApiDescriptor
	configLock     sync.Mutex
}

func newApiConfigManager() *apiConfigManager {
	return &apiConfigManager{
		rpcMethods: make(map[lookupKey]*endpoints.ApiMethod),
		restMethods:    make([]*restMethod, 0),
		configs:         make(map[lookupKey]*endpoints.ApiDescriptor),
		configLock:     sync.Mutex{},
	}
}

type lookupKey struct {
	methodName  string
	version     string
}

type restMethod struct {
	compiledPathPattern *regexp.Regexp // A compiled regex to match against the incoming URL.
	path                  string // The original path pattern (checked to prevent duplicates).
	methods               map[string]*methodInfo
}

type methodInfo struct {
	methodName string
	apiMethod  *endpoints.ApiMethod
}

// Switch the URLs in one API configuration to use HTTP instead of HTTPS.
//
// This modifies the given config in place.
func convertHttpsToHttp(config *endpoints.ApiDescriptor) {
	if len(config.Adapter.Bns) > 0 {
		bns := config.Adapter.Bns
		if strings.HasPrefix(bns, "https://") {
			config.Adapter.Bns = strings.Replace(bns, "https://", "http://", 1)
		}
	}
	if strings.HasPrefix(config.Root, "https://") {
		config.Root = strings.Replace(config.Root, "https://", "http://", 1)
	}
}

// Parses the JSON body of the getApiConfigs response and registers methods
// for dispatch.
//
// Parses method name, etc for all methods and updates the indexing
// datastructures with the information.
func (m *apiConfigManager) parseApiConfigResponse(body string) error {
	var responseObj map[string]interface{}
	err := json.Unmarshal([]byte(body), &responseObj)
	if err != nil {
		return fmt.Errorf("Cannot parse BackendService.getApiConfigs response: %s", body)
	}

	m.configLock.Lock()
	defer m.configLock.Unlock()

	m.addDiscoveryConfig()
	items, ok := responseObj["items"]
	if !ok {
		return errors.New(`BackendService.getApiConfigs response missing "items" key.`)
	}
	itemArray, ok := items.([]interface{})
	if !ok {
		return fmt.Errorf(`Invalid type for "items" value in response: %#v`, items)
	}

	for _, apiConfigJson := range itemArray {
		apiConfigJsonStr, ok := apiConfigJson.(string)
		if !ok {
			return fmt.Errorf(`Invalid type for "items" value in response: %#v`, apiConfigJson)
		}
		var config *endpoints.ApiDescriptor
		err := json.Unmarshal([]byte(apiConfigJsonStr), &config)
		if err != nil {
			glog.Errorf("Can not parse API config: %s", apiConfigJsonStr)
		} else {
			lookupKey := lookupKey{config.Name, config.Version}
			convertHttpsToHttp(config)
			m.configs[lookupKey] = config
		}
	}

	for _, config := range m.configs {
		name := config.Name
		version := config.Version
		sortedMethods := sortMethods(config.Methods)

		for _, methodInfo := range sortedMethods {
			m.saveRpcMethod(methodInfo.methodName, version, methodInfo.apiMethod)
			m.saveRestMethod(methodInfo.methodName, name, version, methodInfo.apiMethod)
		}
	}
	return nil
}

// Gets path parameters from a regular expression match.
//
// Returns a map containing the variable names converted from base64.
func pathParams(names []string, match []string) (map[string]string, error) {
	result := make(map[string]string)
	if len(names) != len(match) {
		return result, fmt.Errorf("Length of names [%d] and matches [%d] not equal.",
			len(names), len(match))
	}
	for i, varName := range names {
		if i == 0 || varName == "" {
			continue
		}
		value := match[i]
		actualVarName, err := fromSafePathParamName(varName)
		if err != nil {
			return result, err
		}
		result[actualVarName] = value
	}
	return result, nil
}

// Looks up the JsonRPC method at call time.
//
// The method is looked up in rpcMethods, the map that
// it is saved in for saveRpcMethod().
//
// Returns a method descriptor as specified in the API configuration.
func (m *apiConfigManager) lookupRpcMethod(methodName, version string) *endpoints.ApiMethod {
	m.configLock.Lock()
	defer m.configLock.Unlock()
	method, _ := m.rpcMethods[lookupKey{methodName, version}]
	return method
}

// Looks up the REST method at call time.
//
// The method is looked up in restMethods, the list it is saved
// in for saveRestMethod.
//
// Args:
//   path: A string containing the path from the URL of the request.
//   http_method: A string containing HTTP method of the request.
//
// Returns the name of the method that was matched, the descriptor as
// specified in the API configuration and a map of path parameters matched
// in the REST request.
func (m *apiConfigManager) lookupRestMethod(path, httpMethod string) (string, *endpoints.ApiMethod, map[string]string) {
	m.configLock.Lock()
	defer m.configLock.Unlock()
	for _, rm := range m.restMethods {
		match := rm.compiledPathPattern.MatchString(path)
		if match {
			params, err := pathParams(
				rm.compiledPathPattern.SubexpNames(),
				rm.compiledPathPattern.FindStringSubmatch(path),
			)
			if err != nil {
				glog.Errorf("Error extracting path [%s] parameters: %s",
					path, err.Error())
				continue
			}
			methodKey := strings.ToLower(httpMethod)
			method, ok := rm.methods[methodKey]
			if ok {
				return method.methodName, method.apiMethod, params
			} else {
				glog.Infof("No %s method found for path: %s", httpMethod, path)
			}
		}
	}
	glog.Infof("No endpoint found for path: %s", path)
	return "", nil, nil
}

func (m *apiConfigManager) addDiscoveryConfig() {
	lookupKey := lookupKey{discoveryApiConfig.Name, discoveryApiConfig.Version}
	m.configs[lookupKey] = discoveryApiConfig
}

// Takes a string containing the parameter matched from the URL template and
// returns a safe string to be used as a regex group name.
//
// Only alphanumeric characters and underscore are allowed in variable name
// tokens, and numeric are not allowed as the first character.
//
// We convert the matched_parameter to base32 (since the alphabet is safe),
// strip the padding (= not safe) and prepend with _, since we know a token
// can begin with underscore.
func toSafePathParamName(matchedParameter string) string {
	safe := "_" + base32.StdEncoding.EncodeToString([]byte(matchedParameter))
	return strings.TrimRight(safe, "=")
}

// Takes a safe regex group name and converts it back to the original value.
//
// Only alphanumeric characters and underscore are allowed in variable name
// tokens, and numeric are not allowed as the first character.
//
// The safe_parameter is a base32 representation of the actual value.
func fromSafePathParamName(safeParameter string) (string, error) {
	if !strings.HasPrefix(safeParameter, "_") {
		return "", fmt.Errorf(`Safe parameter lacks "_" prefix: %s`, safeParameter)
	}
	safeParameterAsBase32 := safeParameter[1:]

	paddingLength := -len(safeParameterAsBase32)%8
	if paddingLength < 0 {
		paddingLength = paddingLength + 8
	}
	padding := strings.Repeat("=", paddingLength)
	data, err := base32.StdEncoding.DecodeString(safeParameterAsBase32 + padding)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Replaces a {variable} with a regex to match it by name.
//
// Changes the string corresponding to the variable name to the base32
// representation of the string, prepended by an underscore. This is
// necessary because we can have message variable names in URL patterns
// (e.g. via {x.y}) but the character "." can"t be in a regex group name.
func replaceVariable(varName string) string {
	if varName != "" {
		safeName := toSafePathParamName(varName)
		return fmt.Sprintf("(?P<%s>%s)", safeName, pathValuePattern)
	}
	return varName
}

// Generates a compiled regex pattern for a parameterized path pattern.
//
// e.g. "/MyApi/v1/notes/{id}"
// returns regexp.MustCompile(r"^/MyApi/v1/notes/(?P<id>[^:/?#\[\]{}]*)")
func compilePathPattern(ppp string) (*regexp.Regexp, error) {

	idxs, err := braceIndices(ppp)
	if err != nil {
		return nil, err
	}
	replacements := make([]string, len(idxs)/2)
	for i := 0; i < len(idxs); i += 2 {
		varName := ppp[idxs[i] + 1 : idxs[i + 1] - 1]
		ok := pathVariablePattern.MatchString(varName)
		var replaced string
		if !ok {
			return nil, fmt.Errorf("Invalid variable name: %s", varName)
		}
		replaced = replaceVariable(varName)
		replacements[i/2] = replaced
	}

	var pattern bytes.Buffer
	start := 0
	for i := 0; i < len(idxs); i += 2 {
		pattern.WriteString(ppp[start:idxs[i]])
		pattern.WriteString(replacements[i/2])
		start = idxs[i + 1]
	}
	pattern.WriteString(ppp[start:])

	re, err := regexp.Compile("^" + pattern.String() + "/?$")
	if err != nil {
		return nil, err
	}
	return re, nil
}

// Store JsonRpc api methods in a map for lookup at call time.
//
// (methodName, version) => method.
func (m *apiConfigManager) saveRpcMethod(methodName, version string, method *endpoints.ApiMethod) {
	glog.Infof("Registering RPC method: %s %s %s %s", methodName, version, method.HttpMethod, method.Path)
	m.rpcMethods[lookupKey{methodName, version}] = method
}

// Store Rest api methods in an array for lookup at call time.
//
// The array is restMethods, an array of restMethod structs.
//
// This structure is a bit complex, it supports use in two contexts:
//
// Creation time:
//
// - saveRestMethod is called repeatedly, each method will have a path,
//   which we want to be compiled for fast lookup at call time
//
// - We want to prevent duplicate incoming path patterns, so store the
//   un-compiled path, not counting on a compiled regex being a stable
//   comparison as it is not documented as being stable for this use.
//
// - Need to store the method that will be mapped at calltime.
//
// - Different methods may have the same path but different http method.
//
// Call time:
//
// - Quickly scan through the list attempting .match(path) on each
//   compiled regex to find the path that matches.
//
// - When a path is matched, look up the API method from the request
//   and get the method name and method config for the matching
//   API method and method name.
func (m *apiConfigManager) saveRestMethod(methodName, apiName, version string, method *endpoints.ApiMethod) {
	var compiledPattern *regexp.Regexp
	var err error
	pathPattern := apiName + "/" + version + "/" + method.Path
	httpMethod := strings.ToLower(method.HttpMethod)
	for _, rm := range m.restMethods {
		if rm.path == pathPattern {
			rm.methods[httpMethod] = &methodInfo{methodName, method}
			goto R
		}
	}
	compiledPattern, err = compilePathPattern(pathPattern)
	if err != nil {
		glog.Errorln(err.Error()) // todo: handle error
		return
	}
	glog.Infof("Registering REST method: %s %s %s %s", apiName, version, methodName, pathPattern)
	m.restMethods = append(m.restMethods,
		&restMethod{
			compiledPattern,
			pathPattern,
			map[string]*methodInfo{
				httpMethod: &methodInfo{methodName, method},
			},
		},
	)
R:
	return
}
