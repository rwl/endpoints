// Copyright 2007 Google Inc.

package endpoints

import (
	"net/http"
	"fmt"
	"log"
	"encoding/json"
)

// Pattern for paths handled by this module.
const API_SERVING_PATTERN = "_ah/api/.*"

const _SPI_ROOT_FORMAT = "/_ah/spi/%s"
const _SERVER_SOURCE_IP = "0.2.0.3"

// Internal constants
const _CORS_HEADER_ORIGIN = "Origin"
const _CORS_HEADER_REQUEST_METHOD = "Access-Control-Request-Method"
const _CORS_HEADER_REQUEST_HEADERS = "Access-Control-Request-Headers"
const _CORS_HEADER_ALLOW_ORIGIN = "Access-Control-Allow-Origin"
const _CORS_HEADER_ALLOW_METHODS = "Access-Control-Allow-Methods"
const _CORS_HEADER_ALLOW_HEADERS = "Access-Control-Allow-Headers"
var _CORS_ALLOWED_METHODS = []string{"DELETE", "GET", "PATCH", "POST", "PUT"}

const _API_EXPLORER_URL = "https://developers.google.com/apis-explorer/?base="

// Dispatcher that handles requests to the built-in apiserver handlers.
type EndpointsDispatcher struct {
	dispatcher *Dispatcher // A Dispatcher instance that can be used to make HTTP requests.
	config_manager *ConfigManager // An ApiConfigManager instance that allows a caller to set up an existing configuration for testing.
	dispatchers []dispatcher
}

type dispatcher struct {
	path_regex string
	dispatch_func interface{}
}

func NewEndpointsDispatcher(dispatcher *Dispatcher) *EndpointsDispatcher {
	return NewEndpointsDispatcherConfig(dispatcher, NewApiConfigManager())
}

func NewEndpointsDispatcherConfig(dispatcher *Dispatcher, config_manager *ConfigManager) *EndpointsDispatcher {
	d := &EndpointsDispatcher{
		dispatcher,
		config_manager,
		make([]Dispatcher, 0),
	}
	d.add_dispatcher("/_ah/api/explorer/?$", d.handle_api_explorer_request)
	d.add_dispatcher("/_ah/api/static/.*$", d.handle_api_static_request)
}

// Add a request path and dispatch handler.

// Args:
// path_regex: A string regex, the path to match against incoming requests.
// dispatch_function: The function to call for these requests.  The function
// should take (request, start_response) as arguments and
// return the contents of the response body.
func (ed *EndpointsDispatcher) add_dispatcher(path_regex string, dispatch_function interface{}) {
	ed.dispatchers = append(ed.dispatchers, &dispatcher{re.compile(path_regex), dispatch_function})
}

func (ed *EndpointsDispatcher) Handle(w http.ResponseWriter, r *http.Request) {
	fmt.fprintf(w, ed.dispatch(r))
}

func (ed *EndpointsDispatcher) dispatch(r *http.Request) string {
	// Check if this matches any of our special handlers.
	dispatched_response := ed.dispatch_non_api_requests(request, start_response)
	if dispatched_response != nil {
		return dispatched_response
	}

	// Get API configuration first.  We need this so we know how to
	// call the back end.
	api_config_response = ed.get_api_configs()
	if !ed.handle_get_api_configs_response(api_config_response) {
		return ed.fail_request(request, "BackendService.getApiConfigs Error", start_response)
	}

	// Call the service.
	body, err := self.call_spi(request, start_response)
	if err != nil {
		return self.handle_request_error(request, err, start_response)
	}
	return body
}

// Dispatch this request if this is a request to a reserved URL.
//
// If the request matches one of our reserved URLs, this calls
// start_response and returns the response body.
//
// Args:
// request: An ApiRequest, the request from the user.
// start_response:
//
// Returns:
// None if the request doesn't match one of the reserved URLs this
// handles.  Otherwise, returns the response body.
func (ed *EndpointsDispatcher) dispatch_non_api_requests(request *http.Request, start_response string) string {
	for _, d := range ed.dispatchers {
		if d.path_regex.match(request.relative_url) {
			return ed.dispatch_function(request, start_response)
		}
	}
	return ""
}

// Handler for requests to _ah/api/explorer.
//
// This calls start_response and returns the response body.
//
// Args:
// request: An ApiRequest, the request from the user.
// start_response:
//
// Returns:
// A string containing the response body (which is empty, in this case).
func (ed *EndpointsDispatcher) handle_api_explorer_request(request *http.Request, start_response string) {
	base_url := fmt.Sprintf("http://%s:%s/_ah/api", request.server, request.port)
	redirect_url := _API_EXPLORER_URL + base_url
	return send_wsgi_redirect_response(redirect_url, start_response)
}

// Handler for requests to _ah/api/static/.*.
//
// This calls start_response and returns the response body.
//
// Args:
// request: An ApiRequest, the request from the user.
// start_response:
//
// Returns:
// A string containing the response body.
func (ed *EndpointsDispatcher) handle_api_static_request(request *http.Request, start_response string) {
	discovery_api := NewDiscoveryApiProxy()
	response, body := /*discovery_api.*/get_static_file(request.relative_url)
	status_string := fmt.Sprintf("%d %s", response.status, response.reason)
	if response.status == 200 {
		// Some of the headers that come back from the server can't be passed
		// along in our response.  Specifically, the response from the server has
		// transfer-encoding: chunked, which doesn't apply to the response that
		// we're forwarding.  There may be other problematic headers, so we strip
		// off everything but Content-Type.
		return send_wsgi_response(status_string,
			[]string{"Content-Type", response.Header("Content-Type")},
			body, start_response)
	} else {
		log.Error("Discovery API proxy failed on %s with %d. Details: %s", request.relative_url, response.status, body)
		return send_wsgi_response(status_string, response.getheaders(), body, start_response)
	}
}

// Makes a call to the BackendService.getApiConfigs endpoint.
//
// Returns:
// A ResponseTuple containing the response information from the HTTP
// request.
func (ed *EndpointsDispatcher) get_api_configs() *ResponseTuple {
	headers = []string{"Content-Type", "application/json"}
	request_body = "{}"
	response = ed.dispatcher.add_request("POST", "/_ah/spi/BackendService.getApiConfigs", headers, request_body, _SERVER_SOURCE_IP)
	return response
}

// Verifies that a response has the expected status and content type.
//
// Args:
// response: The ResponseTuple to be checked.
// status_code: An int, the HTTP status code to be compared with response
// status.
// content_type: A string with the acceptable Content-Type header value.
// None allows any content type.
//
// Returns:
// True if both status_code and content_type match, else False.
func verify_response(response *ResponseTuple, status_code int, content_type string) bool {
	status = int(response.status.split(" ", 1)[0])
	if status != status_code {
		return false
	}
	if content_type == nil {
		return true
	}
	for _, header := range response.headers {
		if header.lower() == "content-type" {
			return value == content_type
		} else {
			return false
		}
	}
}

// Parses the result of GetApiConfigs and stores its information.
//
// Args:
// api_config_response: The ResponseTuple from the GetApiConfigs call.
//
// Returns:
// True on success, False on failure
func (ed *EndpointsDispatcher) handle_get_api_configs_response(api_config_response) bool {
	if ed.verify_response(api_config_response, 200, "application/json") {
		ed.config_manager.parse_api_config_response(api_config_response.content)
		return true
	} else {
		return false
	}
}

// Generate SPI call (from earlier-saved request).
//
// This calls start_response and returns the response body.
//
// Args:
// orig_request: An ApiRequest, the original request from the user.
// start_response:
//
// Returns:
// A string containing the response body.
func (ed *EndpointsDispatcher) call_spi(orig_request, start_response) string {
	var method_config bool
	if orig_request.is_rpc() {
		method_config = ed.lookup_rpc_method(orig_request)
		params = nil
	} else {
		method_config, params = ed.lookup_rest_method(orig_request)
	}
	if !method_config {
		cors_handler = EndpointsDispatcher.__CheckCorsHeaders(orig_request)
		return send_wsgi_not_found_response(start_response, /*cors_handler=*/cors_handler)
	}

	// Prepare the request for the back end.
	spi_request = ed.transform_request(orig_request, params, method_config)

	// Check if this SPI call is for the Discovery service.  If so, route
	// it to our Discovery handler.
	discovery = NewDiscoveryService(self.config_manager)
	discovery_response = discovery.handle_discovery_request(spi_request.path, spi_request, start_response)
	if len(discovery_response) > 0 {
		return discovery_response
	}

	// Send the request to the user's SPI handlers.
	url = fmt.Sprintf(_SPI_ROOT_FORMAT, spi_request.path)
	spi_request.headers["Content-Type"] = "application/json"
	response = ed.dispatcher.add_request("POST", url,
		spi_request.headers.items(),
		spi_request.body,
		spi_request.source_ip)
	return ed.handle_spi_response(orig_request, spi_request, response, start_response)
}

// Track information about CORS headers and our response to them.
type checkCorsHeaders struct {
	allow_cors_request bool
	origin string
	cors_request_method string
	cors_request_headers string
}

func newCheckCorsHeaders(request *http.Request) *checkCorsHeaders {
	c := &checkCorsHeaders{false}
	c.check_cors_request(request)
	return c
}

// Check for a CORS request, and see if it gets a CORS response.
func (c *checkCorsHeaders) check_cors_request(request *http.Request) {
	// Check for incoming CORS headers.
	c.origin = request.headers[_CORS_HEADER_ORIGIN]
	c.cors_request_method = request.headers[_CORS_HEADER_REQUEST_METHOD]
	c.cors_request_headers = request.headers[_CORS_HEADER_REQUEST_HEADERS]

	// Check if the request should get a CORS response.
	if c.origin && ((c.cors_request_method == nil) || (c.cors_request_method.upper() in _CORS_ALLOWED_METHODS)) {
		c.allow_cors_request = true
	}
}

// Add CORS headers to the response, if needed.
func (c *checkCorsHeaders) update_headers(headers_in) {
	if !c.allow_cors_request {
		return
	}

	// Add CORS headers.
	headers = wsgiref.headers.Headers(headers_in)
	headers[_CORS_HEADER_ALLOW_ORIGIN] = c.origin
	headers[_CORS_HEADER_ALLOW_METHODS] = ','.join(tuple(_CORS_ALLOWED_METHODS))
	if c.cors_request_headers != nil {
		headers[_CORS_HEADER_ALLOW_HEADERS] = c.cors_request_headers
	}
}

// Handle SPI response, transforming output as needed.
//
// This calls start_response and returns the response body.
//
// Args:
// orig_request: An ApiRequest, the original request from the user.
// spi_request: An ApiRequest, the transformed request that was sent to the
// SPI handler.
// response: A ResponseTuple, the response from the SPI handler.
// start_response:
//
// Returns:
// A string containing the response body.
func (ed *EndpointsDispatcher) handle_spi_response(orig_request, spi_request, response, start_response) string {
	// Verify that the response is json.  If it isn't treat, the body as an
	// error message and wrap it in a json error response.
	for header := range response.headers {
		if header.lower() == "content-type" && !value.lower().startswith("application/json") {
			return ed.fail_request(orig_request, fmt.Sprintf("Non-JSON reply: %s", response.content), start_response)
		}
	}

	ed.check_error_response(response)

	// Need to check is_rpc() against the original request, because the
	// incoming request here has had its path modified.
	if orig_request.is_rpc() {
		body = ed.transform_jsonrpc_response(spi_request, response.content)
	} else {
		body = ed.transform_rest_response(response.content)
	}

	cors_handler = newCheckCorsHeaders(orig_request)
	return send_wsgi_response(response.status, response.headers, body, start_response, /*cors_handler=*/cors_handler)
}

// Write an immediate failure response to outfile, no redirect.
//
// This calls start_response and returns the error body.
//
// Args:
// orig_request: An ApiRequest, the original request from the user.
// message: A string containing the error message to be displayed to user.
// start_response:
//
// Returns:
// A string containing the body of the error response.
func (ed *EndpointsDispatcher) fail_request(orig_request, message, start_response) string {
	cors_handler = newCheckCorsHeaders(orig_request)
	return send_wsgi_error_response(message, start_response, /*cors_handler=*/cors_handler)
}

// Looks up and returns rest method for the currently-pending request.
//
// Args:
// orig_request: An ApiRequest, the original request from the user.
//
// Returns:
// A tuple of (method descriptor, parameters), or (None, None) if no method
// was found for the current request.
func (ed *EndpointsDispatcher) lookup_rest_method(orig_request) (string, []string) {
	method_name, method, params := ed.config_manager.lookup_rest_method(orig_request.path, orig_request.http_method)
	orig_request.method_name = method_name
	return method, params
}

// Looks up and returns RPC method for the currently-pending request.
//
// Args:
// orig_request: An ApiRequest, the original request from the user.
//
// Returns:
// The RPC method descriptor that was found for the current request, or None
// if none was found.
func (ed *EndpointsDispatcher) lookup_rpc_method(orig_request) {
	if !orig_request.body_json {
		return nil
	}
	method_name = orig_request.body_json.get("method", "")
	version = orig_request.body_json.get("apiVersion", "")
	orig_request.method_name = method_name
	return ed.config_manager.lookup_rpc_method(method_name, version)
}

// Transforms orig_request to apiserving request.
//
// This method uses orig_request to determine the currently-pending request
// and returns a new transformed request ready to send to the SPI.  This
// method accepts a rest-style or RPC-style request.
//
// Args:
// orig_request: An ApiRequest, the original request from the user.
// params: A dictionary containing path parameters for rest requests, or
// None for an RPC request.
// method_config: A dict, the API config of the method to be called.
//
// Returns:
// An ApiRequest that's a copy of the current request, modified so it can
// be sent to the SPI.  The path is updated and parts of the body or other
// properties may also be changed.
func (ed *EndpointsDispatcher) transform_request(orig_request, params, method_config) {
	if orig_request.is_rpc() {
		request = ed.transform_jsonrpc_request(orig_request)
	} else {
		method_params = method_config.get("request", nil).get("parameters", nil)
	}
	request = ed.transform_rest_request(orig_request, params, method_params)
	request.path = method_config.get("rosyMethod", "")
	return request
}

// Checks if the parameter value is valid if an enum.
//
// If the parameter is not an enum, does nothing. If it is, verifies that
// its value is valid.
//
// Args:
// parameter_name: A string containing the name of the parameter, which is
// either just a variable name or the name with the index appended. For
// example 'var' or 'var[2]'.
// value: A string or list of strings containing the value(s) to be used as
// enum(s) for the parameter.
// field_parameter: The dictionary containing information specific to the
// field in question. This is retrieved from request.parameters in the
// method config.
//
// Raises:
// EnumRejectionError: If the given value is not among the accepted
// enum values in the field parameter.
func (ed *EndpointsDispatcher) check_enum(parameter_name, value, field_parameter) {
	if "enum" not in field_parameter {
		return nil
	}

	enum_values := make([]string, 0)
	for enum := range field_parameter["enum"].values() {
		if "backendValue" in enum {
			enum_values = append(enum_values, enum['backendValue'])
		}
	}

	if value not in enum_values {
		return NewEnumRejectionError(parameter_name, value, enum_values)
	}
}

// Checks if the parameter value is valid against all parameter rules.
//
// If the value is a list this will recursively call _check_parameter
// on the values in the list. Otherwise, it checks all parameter rules for the
// the current value.
//
// In the list case, '[index-of-value]' is appended to the parameter name for
// error reporting purposes.
//
// Currently only checks if value adheres to enum rule, but more checks may be
// added.
//
// Args:
// parameter_name: A string containing the name of the parameter, which is
// either just a variable name or the name with the index appended, in the
// recursive case. For example 'var' or 'var[2]'.
// value: A string or list of strings containing the value(s) to be used for
// the parameter.
// field_parameter: The dictionary containing information specific to the
// field in question. This is retrieved from request.parameters in the
// method config.
func (ed *EndpointsDispatcher) check_parameter(parameter_name, value, field_parameter) {
	if isinstance(value, list) {
		for index, element := range(value) {
			parameter_name_index = fmt.Sprintf("%s[%d]", parameter_name, index)
			ed.check_parameter(parameter_name_index, element, field_parameter)
		}
		return
	}

	ed.check_enum(parameter_name, value, field_parameter)
}

// Converts a . delimitied field name to a message field in parameters.
//
// This adds the field to the params dict, broken out so that message
// parameters appear as sub-dicts within the outer param.
//
// For example:
// {'a.b.c': ['foo']}
// becomes:
// {'a': {'b': {'c': ['foo']}}}
//
// Args:
// field_name: A string containing the '.' delimitied name to be converted
// into a dictionary.
// value: The value to be set.
// params: The dictionary holding all the parameters, where the value is
// eventually set.
func (ed *EndpointsDispatcher) add_message_field(field_name, value, params) {
	if "." not in field_name {
		params[field_name] = value
		return
	}

	root, remaining = field_name.split(".", 1)
	sub_params = params.setdefault(root, {})
	ed.add_message_field(remaining, value, sub_params)
}

// Updates the dictionary for an API payload with the request body.
//
// The values from the body should override those already in the payload, but
// for nested fields (message objects) the values can be combined
// recursively.
//
// Args:
// destination: A dictionary containing an API payload parsed from the
// path and query parameters in a request.
// source: A dictionary parsed from the body of the request.
func (ed *EndpointsDispatcher) update_from_body(self, destination, source) {
	for key, value := range source {
		destination_value = destination.get(key)
		if isinstance(value, dict) && isinstance(destination_value, dict) {
			ed.update_from_body(destination_value, value)
		} else {
			destination[key] = value
		}
	}
}

// Translates a Rest request into an apiserving request.
//
// This makes a copy of orig_request and transforms it to apiserving
// format (moving request parameters to the body).
//
// The request can receive values from the path, query and body and combine
// them before sending them along to the SPI server. In cases of collision,
// objects from the body take precedence over those from the query, which in
// turn take precedence over those from the path.
//
// In the case that a repeated value occurs in both the query and the path,
// those values can be combined, but if that value also occurred in the body,
// it would override any other values.
//
// In the case of nested values from message fields, non-colliding values
// from subfields can be combined. For example, if '?a.c=10' occurs in the
// query string and "{'a': {'b': 11}}" occurs in the body, then they will be
// combined as
//
// {
//   'a': {
//     'b': 11,
//     'c': 10,
//   }
// }
//
// before being sent to the SPI server.
//
// Args:
// orig_request: An ApiRequest, the original request from the user.
// params: A dict with URL path parameters extracted by the config_manager
// lookup.
// method_parameters: A dictionary containing the API configuration for the
// parameters for the request.
//
// Returns:
// A copy of the current request that's been modified so it can be sent
// to the SPI.  The body is updated to include parameters from the
// URL.
func (ed *EndpointsDispatcher) transform_rest_request(orig_request, params, method_parameters) *http.Request {
	request = orig_request.copy()
	body_json = make(map[string]interface{})

	// Handle parameters from the URL path.
	for key, value := range params {
		// Values need to be in a list to interact with query parameter values
		// and to account for case of repeated parameters
		body_json[key] = []string{value}
	}

	// Add in parameters from the query string.
	if request.parameters {
		// For repeated elements, query and path work together
		for key, value := range request.parameters {
			if key in body_json {
				body_json[key] = value + body_json[key]
			} else {
				body_json[key] = value
			}
		}
	}

	// Validate all parameters we've merged so far and convert any '.' delimited
	// parameters to nested parameters.  We don't use iteritems since we may
	// modify body_json within the loop.  For instance, 'a.b' is not a valid key
	// and would be replaced with 'a'.
	for key, value := range body_json {
		current_parameter = method_parameters.get(key, nil)
		repeated = current_parameter.get("repeated", false)

		if !repeated {
			body_json[key] = body_json[key][0]
		}

		// Order is important here.  Parameter names are dot-delimited in
		// parameters instead of nested in dictionaries as a message field is, so
		// we need to call _check_parameter on them before calling
		// _add_message_field.

		ed.check_parameter(key, body_json[key], current_parameter)
		// Remove the old key and try to convert to nested message value
		message_value = body_json.pop(key)
		ed.add_message_field(key, message_value, body_json)
	}

	// Add in values from the body of the request.
	if request.body_json {
		ed.update_from_body(body_json, request.body_json)
	}

	request.body_json = body_json
	request.body = json.dumps(request.body_json)
	return request
}

// Translates a JsonRpc request/response into apiserving request/response.
//
// Args:
// orig_request: An ApiRequest, the original request from the user.
//
// Returns:
// A new request with the request_id updated and params moved to the body.
func (ed *EndpointsDispatcher) transform_jsonrpc_request(orig_request) *http.Request {
	request = orig_request.copy()
	request.request_id = request.body_json.get("id")
	request.body_json = request.body_json.get("params", nil)
	request.body = json.dumps(request.body_json)
	return request
}

// Raise an exception if the response from the SPI was an error.
//
// Args:
// response: A ResponseTuple containing the backend response.
//
// Raises:
// BackendError if the response is an error.
func (ed *EndpointsDispatcher) check_error_response(response) error {
	status_code = int(response.status.split(" ", 1)[0])
	if status_code >= 300 {
		return NewBackendError(response)
	}
	return nil
}

// Translates an apiserving REST response so it's ready to return.
//
// Currently, the only thing that needs to be fixed here is indentation,
// so it's consistent with what the live app will return.
//
// Args:
// response_body: A string containing the backend response.
//
// Returns:
// A reformatted version of the response JSON.
func (ed *EndpointsDispatcher) transform_rest_response(response_body) string {
	body_json = json.loads(response_body)
	return json.dumps(body_json, /*indent=*/1, /*sort_keys=*/True)
}

// Translates an apiserving response to a JsonRpc response.
//
// Args:
// spi_request: An ApiRequest, the transformed request that was sent to the
// SPI handler.
// response_body: A string containing the backend response to transform
// back to JsonRPC.
//
// Returns:
// A string with the updated, JsonRPC-formatted request body.
func (ed *EndpointsDispatcher) transform_jsonrpc_response(spi_request, response_body) {
	body_json = {"result": json.loads(response_body)}
	return ed.finish_rpc_response(spi_request.request_id, spi_request.is_batch(), body_json)
}

// Finish adding information to a JSON RPC response.
//
// Args:
// request_id: None if the request didn't have a request ID.  Otherwise, this
// is a string containing the request ID for the request.
// is_batch: A boolean indicating whether the request is a batch request.
// body_json: A dict containing the JSON body of the response.
//
// Returns:
// A string with the updated, JsonRPC-formatted request body.
func (ed *EndpointsDispatcher) finish_rpc_response(request_id, is_batch, body_json) string {
	if request_id != nil {
		body_json["id"] = request_id
	}
	if is_batch {
		body_json = [body_json]
	}
	return json.dumps(body_json, /*indent=*/1, /*sort_keys=*/True)
}

// Handle a request error, converting it to a WSGI response.
//
// Args:
// orig_request: An ApiRequest, the original request from the user.
// error: A RequestError containing information about the error.
// start_response: A function with semantics defined in PEP-333.
//
// Returns:
// A string containing the response body.
func (ed *EndpointsDispatcher) handle_request_error(orig_request, error, start_response) string {
	headers = [("Content-Type", "application/json")]
	if orig_request.is_rpc() {
		// JSON RPC errors are returned with status 200 OK and the
		// error details in the body.
		status_code = 200
		body = ed.finish_rpc_response(orig_request.body_json.get("id"),
			orig_request.is_batch(), error.rpc_error())
	} else {
		status_code = error.status_code()
		body = error.rest_error()
	}

	response_status = fmt.Sprintf("%d %s", status_code, httplib.responses.get(status_code, "Unknown Error"))
	cors_handler = newCheckCorsHeaders(orig_request)
	return send_wsgi_response(response_status, headers, body, start_response, /*cors_handler=*/cors_handler)
}
