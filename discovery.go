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
	DiscoveryProxyHost = "https://webapis-discovery.appspot.com"
	StaticProxyHost = "https://webapis-discovery.appspot.com"
	DiscoveryApiPathPrefix = "/_ah/api/discovery/v1/"
)

type ApiFormat string
const (
	REST ApiFormat = "rest"
	RPC  ApiFormat = "rpc"
)

// Proxies GET request to discovery service API.
//
// Args:
// path: A string containing the URL path relative to discovery service.
// body: A string containing the HTTP POST request body.
//
// Returns:
// HTTP response body or None if it failed.
func dispatchDiscoveryRequest(path, body string) (string, error) {
	fullPath := DiscoveryProxyHost+DiscoveryApiPathPrefix+path
	client := &http.Client{}

	req, err := http.NewRequest("POST", fullPath, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	responseBody := string(respBody)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Discovery API proxy failed on %s with %d.\r\nRequest: %s\r\nResponse: %s",
			fullPath, resp.StatusCode, body, responseBody)
	}
	return responseBody, nil
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
func generateDiscoveryDoc(apiConfig *endpoints.ApiDescriptor, apiFormat ApiFormat) (string, error) {
	path := "apis/generate/" + string(apiFormat)
	requestMap := map[string]interface{}{"config": apiConfig}
	requestBody, err := json.Marshal(requestMap)
	if err != nil {
		return "", err
	}
	return dispatchDiscoveryRequest(path, string(requestBody))
}

// Generates an API directory from a list of API files.
//
// Args:
//   api_configs: A list of strings which are the .api file contents.
//
// Returns:
// The API directory as JSON string.
func generateDiscoveryDirectory(apiConfigs []string) (string, error) {
	requestMap := map[string]interface{}{"configs": apiConfigs}
	requestBody, err := json.Marshal(requestMap)
	if err != nil {
		return "", err
	}
	return dispatchDiscoveryRequest("apis/generate/directory", string(requestBody))
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
func getStaticFile(path string) (*http.Response, string, error) {
	resp, err := http.Get(StaticProxyHost+path)
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
