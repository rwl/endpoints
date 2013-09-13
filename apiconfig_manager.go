// Configuration manager to store API configurations.
package endpoint

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"log"
	"regexp"
	"strings"
	"sync"
)

const PATH_VALUE_PATTERN = `[^:/?#\[\]{}]*`

var PathVariablePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z_.\d]*`)

// Manages loading api configs and method lookup.
type ApiConfigManager struct {
	rpcMethods map[lookupKey]*endpoints.ApiMethod
	restMethods    []*restMethod
	configs         map[lookupKey]*endpoints.ApiDescriptor
	configLock     sync.Mutex
}

func NewApiConfigManager() *ApiConfigManager {
	return &ApiConfigManager{
		rpcMethods: make(map[lookupKey]*endpoints.ApiMethod),
		restMethods:    make([]*restMethod, 0),
		configs:         make(map[lookupKey]*endpoints.ApiDescriptor),
		configLock:     sync.Mutex{},
	}
}

type lookupKey struct {
	methodName string
	version     string
}

type restMethod struct {
	compiledPathPattern *regexp.Regexp
	path                  string
	methods               map[string]*methodInfo
}

type methodInfo struct {
	methodName string
	apiMethod  *endpoints.ApiMethod
}

// Switch the URLs in one API configuration to use HTTP instead of HTTPS.
//
// When doing local development in the dev server, any requests to the API
// need to use HTTP rather than HTTPS.  This converts the API configuration
// to use HTTP.  With this change, client libraries that use the API
// configuration will now be able to communicate with the local server.
//
// This modifies the given dictionary in place.
//
// Args:
//   config: A dict with the JSON configuration for an API.
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

// Parses a json api config and registers methods for dispatch.
//
// Side effects:
//   Parses method name, etc for all methods and updates the indexing
//   datastructures with the information.
//
// Args:
//   body: A string, the JSON body of the getApiConfigs response.
func (m *ApiConfigManager) parseApiConfigResponse(body string) error {
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
			log.Printf("Can not parse API config: %s", apiConfigJsonStr)
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
// Args:
//   match: A regular expression Match object for a path.
//
// Returns:
//   A dictionary containing the variable names converted from base64.
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

// Lookup the JsonRPC method at call time.
//
// The method is looked up in self._rpc_method_dict, the dictionary that
// it is saved in for SaveRpcMethod().
//
// Args:
//   method_name: A string containing the name of the method.
//   version: A string containing the version of the API.
//
// Returns:
//   Method descriptor as specified in the API configuration.
func (m *ApiConfigManager) lookupRpcMethod(methodName, version string) *endpoints.ApiMethod {
	m.configLock.Lock()
	defer m.configLock.Unlock()
	method, _ := m.rpcMethods[lookupKey{methodName, version}]
	return method
}

// Look up the rest method at call time.
//
// The method is looked up in self._rest_methods, the list it is saved
// in for SaveRestMethod.
//
// Args:
//   path: A string containing the path from the URL of the request.
//   http_method: A string containing HTTP method of the request.
//
// Returns:
//   Tuple of (<method name>, <method>, <params>)
//   Where:
//     <method name> is the string name of the method that was matched.
//     <method> is the descriptor as specified in the API configuration. -and-
//     <params> is a dict of path parameters matched in the rest request.
func (m *ApiConfigManager) lookupRestMethod(path, httpMethod string) (string, *endpoints.ApiMethod, map[string]string) {
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
				log.Printf("Error extracting path [%s] parameters: %s",
					path, err.Error())
				continue
			}
			methodKey := strings.ToLower(httpMethod)
			method, ok := rm.methods[methodKey]
			if ok {
				return method.methodName, method.apiMethod, params
			} else {
				log.Printf("No %s method found for path: %s", httpMethod, path)
			}
		}
	}
	log.Printf("No endpoint found for path: %s", path)
	return "", nil, nil
}

func (m *ApiConfigManager) addDiscoveryConfig() {
	lookupKey := lookupKey{DiscoveryApiConfig.Name, DiscoveryApiConfig.Version}
	m.configs[lookupKey] = DiscoveryApiConfig
}

// Creates a safe string to be used as a regex group name.
//
// Only alphanumeric characters and underscore are allowed in variable name
// tokens, and numeric are not allowed as the first character.
//
// We cast the matched_parameter to base32 (since the alphabet is safe),
// strip the padding (= not safe) and prepend with _, since we know a token
// can begin with underscore.
//
// Args:
//   matched_parameter: A string containing the parameter matched from the URL
//   template.
//
// Returns:
//   A string that"s safe to be used as a regex group name.
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
//
// Args:
//   safe_parameter: A string that was generated by _to_safe_path_param_name.
//
// Returns:
//   A string, the parameter matched from the URL template.
func fromSafePathParamName(safeParameter string) (string, error) {
	if !strings.HasPrefix(safeParameter, "_") {
		return "", fmt.Errorf(`Safe parameter lacks "_" prefix: %s`, safeParameter)
	}
	safeParameterAsBase32 := safeParameter[1:]

	paddingLength := -len(safeParameterAsBase32) % 8
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

// Generates a compiled regex pattern for a path pattern.
//
// e.g. "/MyApi/v1/notes/{id}"
// returns re.compile(r"^/MyApi/v1/notes/(?P<id>[^:/?#\[\]{}]*)")
//
// Args:
//   pattern: A string, the parameterized path pattern to be checked.
//
// Returns:
// A compiled regex object to match this path pattern.
func compilePathPattern(ppp string) (*regexp.Regexp, error) {

	// Replaces a {variable} with a regex to match it by name.
	//
	// Changes the string corresponding to the variable name to the base32
	// representation of the string, prepended by an underscore. This is
	// necessary because we can have message variable names in URL patterns
	// (e.g. via {x.y}) but the character "." can"t be in a regex group name.
	//
	// Args:
	//   match: A regex match object, the matching regex group as sent by
	//   re.sub().
	//
	// Returns:
	//   A string regex to match the variable by name, if the full pattern was
	//   matched.
	replaceVariable := func(varName string) string {
		if varName != "" {
			safeName := toSafePathParamName(varName)
			return fmt.Sprintf("(?P<%s>%s)", safeName, PATH_VALUE_PATTERN)
		}
		return varName
	}

	idxs, err := braceIndices(ppp)
	if err != nil {
		return nil, err
	}
	replacements := make([]string, len(idxs)/2)
	for i := 0; i < len(idxs); i += 2 {
		varName := ppp[idxs[i]+1 : idxs[i+1]-1]
		ok := PathVariablePattern.MatchString(varName)
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
		start = idxs[i+1]
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
// (rpcMethodName, apiVersion) => method.
//
// Args:
//   method_name: A string containing the name of the API method.
//   version: A string containing the version of the API.
//   method: A dict containing the method descriptor (as in the api config
//     file).
func (m *ApiConfigManager) saveRpcMethod(methodName, version string, method *endpoints.ApiMethod) {
	m.rpcMethods[lookupKey{methodName, version}] = method
}

// Store Rest api methods in a list for lookup at call time.
//
// The list is self._rest_methods, a list of tuples:
//   [(<compiled_path>, <path_pattern>, <method_dict>), ...]
// where:
//   <compiled_path> is a compiled regex to match against the incoming URL
//   <path_pattern> is a string representing the original path pattern,
//   checked on insertion to prevent duplicates.     -and-
//   <method_dict> is a dict of httpMethod => (method_name, method)
//
// This structure is a bit complex, it supports use in two contexts:
//   Creation time:
//     - SaveRestMethod is called repeatedly, each method will have a path,
//       which we want to be compiled for fast lookup at call time
//     - We want to prevent duplicate incoming path patterns, so store the
//       un-compiled path, not counting on a compiled regex being a stable
//       comparison as it is not documented as being stable for this use.
//     - Need to store the method that will be mapped at calltime.
//     - Different methods may have the same path but different http method.
//   Call time:
//     - Quickly scan through the list attempting .match(path) on each
//       compiled regex to find the path that matches.
//     - When a path is matched, look up the API method from the request
//       and get the method name and method config for the matching
//       API method and method name.
//
// Args:
//   method_name: A string containing the name of the API method.
//   api_name: A string containing the name of the API.
//   version: A string containing the version of the API.
//   method: A dict containing the method descriptor (as in the api config
//     file).
func (m *ApiConfigManager) saveRestMethod(methodName, apiName, version string, method *endpoints.ApiMethod) {
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
		log.Println(err.Error())
		// fixme: handle error
		return
	}
	//log.Printf("Registering rest method: %s %s %s %s", apiName, version, methodName, pathPattern)
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
