// Proxy that dispatches Discovery requests to the prod Discovery service.
// Copyright 2007 Google Inc.

package endpoint

import (
	"encoding/json"
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"io/ioutil"
	"net/http"
)

// The endpoint host we're using to proxy discovery and static requests.
// Using separate constants to make it easier to change the discovery service.
var (
	_DISCOVERY_PROXY_HOST      = "webapis-discovery.appspot.com"
	_STATIC_PROXY_HOST         = "webapis-discovery.appspot.com"
	_DISCOVERY_API_PATH_PREFIX = "/_ah/api/discovery/v1/"
)

type ApiFormat string

const (
	REST ApiFormat = "rest"
	RPC  ApiFormat = "rpc"
)

// Proxies discovery service requests to a known cloud endpoint.
/*type DiscoveryApiProxy struct {}

func NewDiscoveryApiProxy() *DiscoveryApiProxy {
	return &DiscoveryApiProxy{}
}*/

// Proxies GET request to discovery service API.
//
// Args:
// path: A string containing the URL path relative to discovery service.
// body: A string containing the HTTP POST request body.
//
// Returns:
// HTTP response body or None if it failed.
func /*(dp *DiscoveryApiProxy)*/ dispatch_discovery_request(path, body string) (string, error) {
	full_path := _DISCOVERY_API_PATH_PREFIX + path
	client := &http.Client{}

	req, err := http.NewRequest("POST", _DISCOVERY_PROXY_HOST+full_path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)

	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	response_body := string(resp_body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Discovery API proxy failed on %s with %d.\r\nRequest: %s\r\nResponse: %s",
			full_path, resp.Status, body, response_body)
	}
	return response_body, nil
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
func /*(dp *DiscoveryApiProxy)*/ generate_discovery_doc(api_config *endpoints.ApiDescriptor, api_format ApiFormat) (string, error) {
	path := "apis/generate/" + string(api_format)
	//var config interface{}
	//err := json.Unmarshal([]byte(api_config), &config)
	//if err != nil {
	//	return "", err
	//}
	request_dict := map[string]interface{}{"config": api_config}
	request_body, err := json.Marshal(request_dict)
	if err != nil {
		return "", err
	}
	return dispatch_discovery_request(path, string(request_body))
}

// Generates an API directory from a list of API files.
//
// Args:
//   api_configs: A list of strings which are the .api file contents.
//
// Returns:
// The API directory as JSON string.
func /*(dp *DiscoveryApiProxy)*/ generate_discovery_directory(api_configs []string) (string, error) {
	request_dict := map[string]interface{}{"configs": api_configs}
	request_body, err := json.Marshal(request_dict)
	if err != nil {
		return "", err
	}
	return dispatch_discovery_request("apis/generate/directory", string(request_body))
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
func /*(dp *DiscoveryApiProxy)*/ get_static_file(path string) (*http.Response, string, error) {
	resp, err := http.Get(_STATIC_PROXY_HOST + path)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return resp, string(body), nil
}
