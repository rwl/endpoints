// Configuration manager to store API configurations.
package endpoints

import (
	"sync"
	"encoding/json"
	"fmt"
	"encoding/base64"
	"regexp"
)

const _PATH_VARIABLE_PATTERN = `[a-zA-Z_][a-zA-Z_.\d]*`
const _PATH_VALUE_PATTERN = `[^:/?#\[\]{}]*`

// Manages loading api configs and method lookup.
type ApiConfigManager struct {
	rpc_method_dict map[string]interface{}
	rest_methods []string
	configs map[string]interface{}
	config_lock sync.Mutex
}

func NewApiConfigManager() *ApiConfigManager {
	return &ApiConfigManager{
		rpc_method_dict: make(map[string]interface{}),
		rest_methods: make([]string, 0),
		configs: make(map[string]interface{}),
		config_lock: sync.Mutex{},
	}
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
	err := json.Marshal(body, &response_obj)
	if err != nil {
		return fmt.Errorf("Cannot parse BackendService.getApiConfigs response: %s", body)
	}
	m.config_lock.Lock()
	defer m.config_lock.Unlock()
	m.add_discovery_config()
	items, ok := response_obj["items"]
	if !ok {
		items = make([]string, 0)
	}
	for _, api_config_json := range items {
		var config map[string]interface{}
		err := json.Marshal(api_config_json, config)
		if err != nil {
			return fmt.Errorf("Can not parse API config: %s", api_config_json)
		}
		lookup_key = config.get("name", ""), config.get("version", "")
		_convert_https_to_http(config)
		m.configs[lookup_key] = config
	}

	for _, config := range m.configs {
		name = config.get("name", "")
		version = config.get("version", "")
		sorted_methods = self._get_sorted_methods(config.get("methods", {}))
	}

	for method_name, method := range sorted_methods {
		m.save_rpc_method(method_name, version, method)
		m.save_rest_method(method_name, name, version, method)
	}
}

type methodInfo struct {
	name string
	info map[string]string
}

type ByMethod []ApiDescriptor

func (by ByMethod) Len() int {
	return len(by)
}
func (by ByMethod) Less(i, j int) bool {
	method_info1 := by[i]
	method_info2 := by[j]

	path1 := method_info1.info["path"]
	path2 := method_info2.info["path"]

	// Higher path scores come first.
	path_score1 := score_path(path1)
	path_score2 := score_path(path2)
	if path_score1 != path_score2 {
//		return path_score2 - path_score1
		return path_score1 < path_score2
	}

	// Compare by path text next, sorted alphabetically.
//	path_result = cmp(path1, path2)
	if path1 != path2 {
		return path1 < path2
	}

	// All else being equal, sort by HTTP method.
	httpMethod1 := method_info1.info["httpMethod"]
	httpMethod2 := method_info2.info["httpMethod"]
	return httpMethod1 < httpMethod2
}

func (by ByMethod) Swap(i, j int) {
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
	score = 0
	parts = path.split("/")
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
func (m *ApiConfigManager) lookup_rpc_method(method_name, version string) string {
	m.config_lock.Lock()
	defer m.config_lock.Unlock()
	method, ok := m.rpc_method_dict[(method_name, version)]
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
func (m *ApiConfigManager) lookup_rest_method(path, http_method string) {
	m.config_lock.Lock()
	defer m.config_lock.Unlock()
	for compiled_path_pattern, unused_path, methods := range m.rest_methods {
		match = compiled_path_pattern.match(path)
		if match {
			params = get_path_params(match)
			method_key = http_method.lower()
			method_name, method = methods.get(method_key, (nil, nil))
			if method {
				//break
				return method_name, method, params
			}
		}
	}
	log.Warn("No endpoint found for path: %s", path)
	return nil, nil, nil
}

func (m *ApiConfigManager) add_discovery_config() {
	lookup_key = (DiscoveryService.API_CONFIG["name"],
		DiscoveryService.API_CONFIG["version"])
	m.configs[lookup_key] = DiscoveryService.API_CONFIG
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
func to_safe_path_param_name(matched_parameterstring ) string {
	return "_" + base64.b32encode(matched_parameter).rstrip("=")
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
func from_safe_path_param_name(safe_parameterstring ) string {
	if !safe_parameter.startswith("_") {
		panic()
	}
	safe_parameter_as_base32 = safe_parameter[1:]

	padding_length = - len(safe_parameter_as_base32) % 8
	padding = "=" * padding_length
	return base64.b32decode(safe_parameter_as_base32 + padding)
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
func compile_path_pattern(pattern) *regexp.Regexp {

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
			return fmt.Sprintf("%s(?P<%s>%s)", match.group(1), var_name, _PATH_VALUE_PATTERN)
		}
		return match.group(0)
	}

	pattern = re.sub(fmt.Sprintf("(/|^){(%s)}(?=/|$)", _PATH_VARIABLE_PATTERN),
		replace_variable, pattern)
	re, err := regexp.Compile(pattern + "/?$")
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
func (m *ApiConfigManager) save_rpc_method(method_name, version, method string) {
	m.rpc_method_dict[(method_name, version)] = method
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
func (m *ApiConfigManager) save_rest_method(method_name, api_name, version, method) {
	path_pattern = "/".join((api_name, version, method.get("path", "")))
	http_method = method.get("httpMethod", "").lower()
	for _, path, methods := range m.rest_methods {
		if path == path_pattern {
			methods[http_method] = method_name, method
			break
		} else {
			m.rest_methods.append(
				(compile_path_pattern(path_pattern),
				path_pattern,
				{http_method: (method_name, method)}))
		}
	}
}
