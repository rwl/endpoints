// Proxy that dispatches Discovery requests to the prod Discovery service.
// Copyright 2007 Google Inc.

package endpoints

import (
	"log"
	"encoding/json"
)

// The endpoint host we're using to proxy discovery and static requests.
// Using separate constants to make it easier to change the discovery service.
const _DISCOVERY_PROXY_HOST = "webapis-discovery.appspot.com"
const _STATIC_PROXY_HOST = "webapis-discovery.appspot.com"
const _DISCOVERY_API_PATH_PREFIX = "/_ah/api/discovery/v1/"

// Proxies discovery service requests to a known cloud endpoint.
type DiscoveryApiProxy struct {}

// Proxies GET request to discovery service API.
//
// Args:
// path: A string containing the URL path relative to discovery service.
// body: A string containing the HTTP POST request body.
//
// Returns:
// HTTP response body or None if it failed.
func (dp *DiscoveryApiProxy) dispatch_request(path, body) string {
	full_path = _DISCOVERY_API_PATH_PREFIX + path
	headers = {"Content-type": "application/json"}
	connection = httplib.HTTPSConnection(_DISCOVERY_PROXY_HOST)
	defer connection.close()
	connection.request("POST", full_path, body, headers)
	response = connection.getresponse()
	response_body = response.read()
	if response.status != 200 {
		log.Error("Discovery API proxy failed on %s with %d.\r\n" +
			"Request: %s\r\nResponse: %s",
			full_path, response.status, body, response_body)
		return None
	}
	return response_body
}

// Generates a discovery document from an API file.
//
// Args:
// api_config: A string containing the .api file contents.
// api_format: A string, either 'rest' or 'rpc' depending on the which kind
// of discvoery doc is requested.
//
// Returns:
// The discovery doc as JSON string.
//
// Raises:
// ValueError: When api_format is invalid.
func (dp *DiscoveryApiProxy) generate_discovery_doc(api_config, api_format) (string, error) {
	if api_format not in ["rest", "rpc"] {
		return "", NewValueError("Invalid API format")
	}
	path = "apis/generate/" + api_format
	request_dict = {"config": json.dumps(api_config)}
	request_body = json.dumps(request_dict)
	return dp.dispatch_request(path, request_body)
}

// Generates an API directory from a list of API files.
//
// Args:
// api_configs: A list of strings which are the .api file contents.
//
// Returns:
// The API directory as JSON string.
func (dp *DiscoveryApiProxy) generate_directory(api_configs) {
	request_dict = {"configs": api_configs}
	request_body = json.dumps(request_dict)
	return dp.dispatch_request("apis/generate/directory", request_body)
}

// Returns static content via a GET request.
//
// Args:
// path: A string containing the URL path after the domain.
//
// Returns:
// A tuple of (response, response_body):
// response: A HTTPResponse object with the response from the static
// proxy host.
// response_body: A string containing the response body.
func (dp *DiscoveryApiProxy) get_static_file(path string) (string, string) {
	connection = httplib.HTTPSConnection(_STATIC_PROXY_HOST)
	defer connection.close()
	connection.request("GET", path, None, {})
	response = connection.getresponse()
	response_body = response.read()
	return response, response_body
}
