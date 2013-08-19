// Configuration manager to store API configurations.
package endpoints

import (
	"sync"
	"encoding/json"
	"fmt"
	"regexp"
	"log"
	"strings"
	"errors"
	"sort"
	"encoding/base32"
)

const _PATH_VARIABLE_PATTERN = `[a-zA-Z_][a-zA-Z_.\d]*`
const _PATH_VALUE_PATTERN = `[^:/?#\[\]{}]*`

// Manages loading api configs and method lookup.
type ApiConfigManager struct {
	rpc_method_dict map[lookupKey]*ApiMethod
	rest_methods []restMethod
	configs map[lookupKey]*ApiDescriptor
	config_lock sync.Mutex
}

func NewApiConfigManager() *ApiConfigManager {
	return &ApiConfigManager{
		rpc_method_dict: make(map[lookupKey]*ApiMethod),
		rest_methods: make([]string, 0),
		configs: make(map[lookupKey]*ApiDescriptor),
		config_lock: sync.Mutex{},
	}
}

type lookupKey struct {
	method_name string
	version string
}

type restMethod struct {
	compiled_path_pattern *regexp.Regexp
	path string
	methods map[string]method
}

type method struct {
	name string
	api_method *ApiMethod
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
func convert_https_to_http(config *ApiDescriptor) {
	if config.Adapter != nil && len(config.Adapter.Bns) > 0 {
		bns := config.Adapter.Bns
		if strings.HasPrefix(bns, "https://") {
			config.Adapter.Bns = strings.Replace(bns, "https://", "http://", 1)
		}
	}
	if strings.HasPrefix(config.Root, "https://") {
		config.Root = strings.Replace(config.Root, "https://", "http://", 1)
	}
}
/*func _convert_https_to_http(config map[string]interface{}) {
	if "adapter" in config and "bns" in config["adapter"]:
		bns_adapter = config["adapter"]["bns"]
		if bns_adapter.startswith("https://"):
			config["adapter"]["bns"] = bns_adapter.replace("https", "http", 1)
	if "root" in config and config["root"].startswith("https://"):
		config["root"] = config["root"].replace("https", "http", 1)
}*/

// Parses a json api config and registers methods for dispatch.
//
// Side effects:
//   Parses method name, etc for all methods and updates the indexing
//   datastructures with the information.
//
// Args:
//   body: A string, the JSON body of the getApiConfigs response.
func (m *ApiConfigManager) parse_api_config_response(body string) error {
	var response_obj map[string]interface{}
	err := json.Unmarshal(body, &response_obj)
	if err != nil {
		return fmt.Errorf("Cannot parse BackendService.getApiConfigs response: %s",
			body)
	}

	m.config_lock.Lock()
	defer m.config_lock.Unlock()

	m.add_discovery_config()
	items, ok := response_obj["items"]
	if !ok {
		return errors.New(`BackendService.getApiConfigs response missing "items" key.`)
	}
	item_array, ok := items.([]string)
	if !ok {
		return errors.New(`Invalid type for "items" value in response`)
	}

	for _, api_config_json := range item_array {
		var config *ApiDescriptor
		err := json.Unmarshal(api_config_json, config)
		if err != nil {
			return fmt.Errorf("Can not parse API config: %s", api_config_json)
		}
		lookup_key := lookupKey{config.Name, config.Version}
		convert_https_to_http(config)
		m.configs[lookup_key] = config
	}

	for _, config := range m.configs {
		name := config.Name
		version := config.Version
		sorted_methods := get_sorted_methods(config.Methods)

		for method_name, method := range sorted_methods {
			m.save_rpc_method(method_name, version, method)
			m.save_rest_method(method_name, name, version, method)
		}
	}
	return nil
}

// Get a copy of "methods" sorted the way they would be on the live server.
//
// Args:
//   methods: JSON configuration of an API"s methods.
//
// Returns:
//   The same configuration with the methods sorted based on what order
//   they'll be checked by the server.
func get_sorted_methods(methods map[string]*ApiMethod) []*ApiMethod {
	if methods == nil {
		return nil
	}
	methods := make([]*ApiMethod, len(methods))
	i := 0
	for _, m := range methods {
		methods[i] = m
		i++
	}
	sort.Sort(ByPath(methods))
	return methods
}

type ByPath []*ApiMethod

func (by ByPath) Len() int {
	return len(by)
}

// Less returns whether the element with index i should sort
// before the element with index j.
func (by ByPath) Less(i, j int) bool {
	method_info1 := by[i]
	method_info2 := by[j]

	path1 := method_info1.Path
	path2 := method_info2.Path

	path_score1 := score_path(path1)
	path_score2 := score_path(path2)
	if path_score1 != path_score2 {
		// Higher path scores come first.
		return path_score1 > path_score2
	}

	// Compare by path text next, sorted alphabetically.
	if path1 != path2 {
		return path1 < path2
	}

	// All else being equal, sort by HTTP method.
	httpMethod1 := method_info1.HttpMethod
	httpMethod2 := method_info2.HttpMethod
	return httpMethod1 < httpMethod2
}

func (by ByPath) Swap(i, j int) {
	by[i], by[j] = by[j], by[i]
}

// Calculate the score for this path, used for comparisons.
//
// Higher scores have priority, and if scores are equal, the path text
// is sorted alphabetically.  Scores are based on the number and location
// of the constant parts of the path.  The server has some special handling
// for variables with regexes, which we don"t handle here.
//
// Args:
//   path: The request path that we"re calculating a score for.
//
// Returns:
//   The score for the given path.
func score_path(path string) int {
	score := 0
	parts := strings.Split(path, "/")
	for _, part := range parts {
		score <<= 1
		if part == nil || part[0] != "{" {
			// Found a constant.
			score += 1
		}
	}
	// Shift by 31 instead of 32 because some (!) versions of Python like
	// to convert the int to a long if we shift by 32, and the sorted()
	// function that uses this blows up if it receives anything but an int.
	score <<= 31 - len(parts)
	return score
}

// Gets path parameters from a regular expression match.
//
// Args:
//   match: A regular expression Match object for a path.
//
// Returns:
//   A dictionary containing the variable names converted from base64.
func get_path_params(match) map[string]string {
	var result map[string]string
	for var_name, value := range match.groupdict() {
		actual_var_name = from_safe_path_param_name(var_name)
		result[actual_var_name] = value
	}
	return result
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
func (m *ApiConfigManager) lookup_rpc_method(method_name, version string) *ApiMethod {
	m.config_lock.Lock()
	defer m.config_lock.Unlock()
	method, _ := m.rpc_method_dict[lookupKey{method_name, version}]
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
func (m *ApiConfigManager) lookup_rest_method(path, http_method string) (string, *ApiMethod, map[string]string) {
	m.config_lock.Lock()
	defer m.config_lock.Unlock()
	for _, rm := range m.rest_methods {
		match := rm.compiled_path_pattern.Match(path)
		if match {
			params := get_path_params(match)
			method_key := strings.ToLower(http_method)
			method, ok := rm.methods[method_key]
			if ok {
				//break
				return method.name, method.api_method, params
			}
		}
	}
	log.Printf("No endpoint found for path: %s", path)
	return "", nil, nil
}

func (m *ApiConfigManager) add_discovery_config() {
	lookup_key := lookupKey{DISCOVERY_API_CONFIG.Name, DISCOVERY_API_CONFIG.Version}
	m.configs[lookup_key] = DISCOVERY_API_CONFIG
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
func to_safe_path_param_name(matched_parameter string) string {
	safe := "_" + base32.StdEncoding.EncodeToString([]byte(matched_parameter))
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
func from_safe_path_param_name(safe_parameter string) (string, error) {
	if !strings.HasPrefix(safe_parameter, "_") {
		return "", fmt.Errorf(`Safe parameter lacks "_" prefix: %s`, safe_parameter)
	}
	safe_parameter_as_base32 := safe_parameter[1:]

	padding_length := - len(safe_parameter_as_base32) % 8
	padding := strings.Repeat("=", padding_length)
	data, err := base32.StdEncoding.DecodeString(safe_parameter_as_base32 + padding)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Generates a compiled regex pattern for a path pattern.
//
// e.g. "/MyApi/v1/notes/{id}"
// returns re.compile(r"/MyApi/v1/notes/(?P<id>[^:/?#\[\]{}]*)")
//
// Args:
//   pattern: A string, the parameterized path pattern to be checked.
//
// Returns:
// A compiled regex object to match this path pattern.
func compile_path_pattern(pattern string) *regexp.Regexp {

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
	replace_variable := func(match *regexp.Regexp) string {
		if match.lastindex > 1 {
			var_name = to_safe_path_param_name(match.group(2))
			return fmt.Sprintf("%s(?P<%s>%s)", match.group(1), var_name,
				_PATH_VALUE_PATTERN)
		}
		return match.group(0)
	}

	pattern = re.sub(fmt.Sprintf("(/|^){(%s)}(?=/|$)", _PATH_VARIABLE_PATTERN),
		replace_variable, pattern)
	re, err := regexp.Compile(pattern + "/?$")
	if err != nil {

	}
	return re
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
func (m *ApiConfigManager) save_rpc_method(method_name, version string, method *ApiMethod) {
	m.rpc_method_dict[lookupKey{method_name, version}] = method
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
func (m *ApiConfigManager) save_rest_method(method_name, api_name, version string, method *ApiMethod) {
	path_pattern := api_name + "/" + version + "/" + method.Path
	http_method := strings.ToLower(method.HttpMethod)
	for _, rm := range m.rest_methods {
		if rm.path == path_pattern {
			rm.methods[http_method] = method{method_name, method}
			break
		} else {
			rm.rest_methods = append(m.rest_methods,
				method{
					compile_path_pattern(path_pattern),
					path_pattern,
					map[string]*method{
						http_method: method{method_name, method},
					},
				},
			)
		}
	}
}
