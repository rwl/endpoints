// Configuration manager to store API configurations.
package discovery

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/crhym3/go-endpoints/endpoints"
	"log"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const _PATH_VALUE_PATTERN = `[^:/?#\[\]{}]*`

var _PATH_VARIABLE_PATTERN = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z_.\d]*`)


// Manages loading api configs and method lookup.
type ApiConfigManager struct {
	rpc_method_dict map[lookupKey]*endpoints.ApiMethod
	rest_methods    []*restMethod
	configs         map[lookupKey]*endpoints.ApiDescriptor
	config_lock     sync.Mutex
}

func NewApiConfigManager() *ApiConfigManager {
	return &ApiConfigManager{
		rpc_method_dict: make(map[lookupKey]*endpoints.ApiMethod),
		rest_methods:    make([]*restMethod, 0),
		configs:         make(map[lookupKey]*endpoints.ApiDescriptor),
		config_lock:     sync.Mutex{},
	}
}

type lookupKey struct {
	method_name string
	version     string
}

type restMethod struct {
	compiled_path_pattern *regexp.Regexp
	path                  string
	methods               map[string]*methodInfo
}

type methodInfo struct {
	method_name string
	api_method  *endpoints.ApiMethod
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
func convert_https_to_http(config *endpoints.ApiDescriptor) {
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
	err := json.Unmarshal([]byte(body), &response_obj)
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
	item_array, ok := items.([]interface{})
	if !ok {
		return fmt.Errorf(`Invalid type for "items" value in response: %v`, reflect.TypeOf(items))
	}

	for _, api_config_json := range item_array {
		api_config_json_str, ok := api_config_json.(string)
		if !ok {
			return fmt.Errorf(`Invalid type for "items" value in response: %v`, reflect.TypeOf(api_config_json))
		}
		var config *endpoints.ApiDescriptor
		err := json.Unmarshal([]byte(api_config_json_str), &config)
		if err != nil {
			log.Printf("Can not parse API config: %s", api_config_json_str)
		} else {
			lookup_key := lookupKey{config.Name, config.Version}
			convert_https_to_http(config)
			m.configs[lookup_key] = config
		}
	}

	for _, config := range m.configs {
		name := config.Name
		version := config.Version
		sorted_methods := get_sorted_methods(config.Methods)

		for _, method_info := range sorted_methods {
			m.save_rpc_method(method_info.method_name, version, method_info.api_method)
			m.save_rest_method(method_info.method_name, name, version, method_info.api_method)
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
func get_sorted_methods(methods map[string]*endpoints.ApiMethod) []*methodInfo {
	if methods == nil {
		return nil
	}
	sorted_methods := make([]*methodInfo, len(methods))
	i := 0
	for name, m := range methods {
		sorted_methods[i] = &methodInfo{name, m}
		i++
	}
	sort.Sort(ByPath(sorted_methods))
	return sorted_methods
}

type ByPath []*methodInfo

func (by ByPath) Len() int {
	return len(by)
}

// Less returns whether the element with index i should sort
// before the element with index j.
func (by ByPath) Less(i, j int) bool {
	method_info1 := by[i].api_method
	method_info2 := by[j].api_method

	path1 := method_info1.Path
	path2 := method_info2.Path

	path_score1 := score_path(path1)
	path_score2 := score_path(path2)
	//	fmt.Printf("1: %s - %d\n", path1, path_score1)
	//	fmt.Printf("2: %s - %d\n", path2, path_score2)
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
// for variables with regexes, which we don't handle here.
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
		if part == "" || !strings.HasPrefix(part, "{") {
			// Found a constant.
			score += 1
		}
	}
	// Shift by 31 instead of 32 because some (!) versions of Python like
	// to convert the int to a long if we shift by 32, and the sorted()
	// function that uses this blows up if it receives anything but an int.
	score <<= uint(31 - len(parts))
	return score
}

// Gets path parameters from a regular expression match.
//
// Args:
//   match: A regular expression Match object for a path.
//
// Returns:
//   A dictionary containing the variable names converted from base64.
func get_path_params(names []string, match []string) (map[string]string, error) {
	result := make(map[string]string)
	if len(names) != len(match) {
		return result, fmt.Errorf("Length of names [%d] and matches [%d] not equal.",
			len(names), len(match))
	}
	for i, var_name := range names {
		if i == 0 || var_name == "" {
			continue
		}
		value := match[i]
		actual_var_name, err := from_safe_path_param_name(var_name)
		if err != nil {
			return result, err
		}
		result[actual_var_name] = value
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
func (m *ApiConfigManager) lookup_rpc_method(method_name, version string) *endpoints.ApiMethod {
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
func (m *ApiConfigManager) lookup_rest_method(path, http_method string) (string, *endpoints.ApiMethod, map[string]string) {
	m.config_lock.Lock()
	defer m.config_lock.Unlock()
	for _, rm := range m.rest_methods {
		match := rm.compiled_path_pattern.MatchString(path)
		if match {
			params, err := get_path_params(
				rm.compiled_path_pattern.SubexpNames(),
				rm.compiled_path_pattern.FindStringSubmatch(path),
			)
			if err != nil {
				log.Printf("Error extracting path [%s] parameters: %s",
					path, err.Error())
				continue
			}
			method_key := strings.ToLower(http_method)
			method, ok := rm.methods[method_key]
			if ok {
				//break
				return method.method_name, method.api_method, params
			} else {
				log.Printf("No %s method found for path: %s", http_method, path)
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

	padding_length := -len(safe_parameter_as_base32) % 8
	if padding_length < 0 {
		padding_length = padding_length + 8
	}
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
func compile_path_pattern(ppp string) (*regexp.Regexp, error) {

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
	replace_variable := func(var_name string) string {
		if var_name != "" {
			safe_name := to_safe_path_param_name(var_name)
			return fmt.Sprintf("(?P<%s>%s)", safe_name, _PATH_VALUE_PATTERN)
		}
		return var_name
	}
	/*replace_variable := func(match []string) string {
		if len(match) > 2 {
			var_name := to_safe_path_param_name(match[2])
			return fmt.Sprintf("%s(?P<%s>%s)", match[1], var_name,
				_PATH_VALUE_PATTERN)
		}
		return match[0]
	}

	//p := fmt.Sprintf("(/|^){(%s)}(?=/|$)", _PATH_VARIABLE_PATTERN)
	p := fmt.Sprintf("(/|^){(%s)}(/|$)", _PATH_VARIABLE_PATTERN)
	re, err := regexp.Compile(p)
	if err != nil {
		return re, err
	}

	matches := re.FindAllStringSubmatch(pattern, -1)
	indexes := re.FindAllStringSubmatchIndex(pattern, -1)

	offset := 0
	for i, match := range matches {
		index := indexes[i]
		replaced := replace_variable(match)
		if index != nil && len(index) > 1 {
			pattern = pattern[:offset+index[0]] + replaced + pattern[offset+index[1]:]
			offset += len(replaced) - index[1] - index[0]
		}
	}*/

	//	pattern = re.ReplaceAllString(pattern, replace_variable)

	idxs, err := braceIndices(ppp)
	if err != nil {
		return nil, err
	}
	replacements := make([]string, len(idxs)/2)
	for i := 0; i < len(idxs); i += 2 {
		var_name := ppp[idxs[i]+1 : idxs[i+1]-1]
		ok := _PATH_VARIABLE_PATTERN.MatchString(var_name)
		var replaced string
		if !ok {
			return nil, fmt.Errorf("Invalid variable name: %s", var_name)
		}
		replaced = replace_variable(var_name)
		replacements[i/2] = replaced
	}

	var pattern bytes.Buffer
	start := 0
	for i := 0; i < len(idxs); i += 2 {
		pattern.WriteString(ppp[start:idxs[i]])
		pattern.WriteString(replacements[i/2])
		start = idxs[i+1] // + 1
	}
	pattern.WriteString(ppp[start:])

	re, err := regexp.Compile(pattern.String() + "/?$")
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
func (m *ApiConfigManager) save_rpc_method(method_name, version string, method *endpoints.ApiMethod) {
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
func (m *ApiConfigManager) save_rest_method(method_name, api_name, version string, method *endpoints.ApiMethod) {
	var compiled_pattern *regexp.Regexp
	var err error
	path_pattern := api_name + "/" + version + "/" + method.Path
	http_method := strings.ToLower(method.HttpMethod)
	for _, rm := range m.rest_methods {
		if rm.path == path_pattern {
			rm.methods[http_method] = &methodInfo{method_name, method}
			goto R
		}
	}
	compiled_pattern, err = compile_path_pattern(path_pattern)
	if err != nil {
		log.Println(err.Error())
		// fixme: handle error
		return
	}
	m.rest_methods = append(m.rest_methods,
		&restMethod{
			compiled_pattern,
			path_pattern,
			map[string]*methodInfo{
				http_method: &methodInfo{method_name, method},
			},
		},
	)
R:
	return
}
