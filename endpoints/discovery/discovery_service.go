
package endpoints

import (
	"fmt"
	"net/http"
	"log"
	"encoding/json"
)

const _GET_REST_API = "apisdev.getRest"
const _GET_RPC_API = "apisdev.getRpc"
const _LIST_API = "apisdev.list"

var DISCOVERY_API_CONFIG = ApiDescriptor{
	Name: "discovery",
	Version: "v1",
	Methods: map[string]*ApiMethod{
		"discovery.apis.getRest": &ApiMethod{
			"path": "apis/{api}/{version}/rest",
			"httpMethod": "GET",
			"rosyMethod": _GET_REST_API,
		},
		"discovery.apis.getRpc": &ApiMethod{
			"path": "apis/{api}/{version}/rpc",
			"httpMethod": "GET",
			"rosyMethod": _GET_RPC_API,
		},
		"discovery.apis.list": &ApiMethod{
			"path": "apis",
			"httpMethod": "GET",
			"rosyMethod": _LIST_API,
		},
	},
}

// Implements the local devserver discovery service.
//
// This has a static minimal version of the discoverable part of the
// discovery .api file.
//
// It only handles returning the discovery doc and directory, and ignores
// directory parameters to filter the results.
//
// The discovery docs/directory are created by calling a cloud endpoint
// discovery service to generate the discovery docs/directory from an .api
// file/set of .api files.
type DiscoveryService struct {
	config_manager *ApiConfigManager
}

func NewDiscoveryService(config_manager *ApiConfigManager) *DiscoveryService {
	return &DiscoveryService{config_manager}
}

// Sends an HTTP 200 json success response.
//
// This calls start_response and returns the response body.
//
// Args:
//   response: A string containing the response body to return.
//   start_response: A function with semantics defined in PEP-333.
//
// Returns:
//   A string, the response body.
func send_success_response(response string, w http.ResponseWriter) string {
	w.Header.Set("Content-Type", "application/json; charset=UTF-8")
	fmt.Fprintf(w, response)
//	return send_response('200', headers, response, start_response)
	return response
}

// Sends back HTTP response with API directory.
//
// This calls start_response and returns the response body.  It will return
// the discovery doc for the requested api/version.
//
// Args:
//   api_format: A string containing either 'rest' or 'rpc'.
//   request: An ApiRequest, the transformed request sent to the Discovery SPI.
//   start_response: A function with semantics defined in PEP-333.
//
// Returns:
//   A string, the response body.
func (ds *DiscoveryService) get_rpc_or_rest(api_format string, request *ApiRequest, w http.ResponseWriter) string {
	api := request.body_json["api"]
	version := request.body_json["version"]
	lookup_key := lookupKey{api, version}
	api_config, ok := ds.config_manager.configs[lookup_key]
	if !ok {
		log.Printf("No discovery doc for version %s of api %s", version, api)
		return send_not_found_response(w, nil)
	}
	doc, err := generate_discovery_doc(api_config, api_format)
	if err != nil {
		error_msg := fmt.Sprintf("Failed to convert .api to discovery doc for version %s of api %s", version, api)
		log.Print(error_msg)
		return send_error_response(error_msg, w, nil)
	}
	return send_success_response(doc, w)
}

// Sends HTTP response containing the API directory.
//
// This calls start_response and returns the response body.
//
// Args:
//   start_response: A function with semantics defined in PEP-333.
//
// Returns:
//   A string containing the response body.
func (ds *DiscoveryService) list(w http.ResponseWriter) string {
	api_configs := make([]string, 0)
	for _, api_config := range ds.config_manager.configs {
		if api_config != DISCOVERY_API_CONFIG {
			ac, err := json.Marshal(api_config)
			if err != nil {
				log.Printf("Failed to marshal API config: %v", api_config)
				return send_not_found_response(w, nil)
			}
			api_configs = append(api_configs, string(ac))
		}
	}
	directory, err := generate_discovery_directory(api_configs)
	if err != nil {
		log.Printf("Failed to get API directory: %s", err.Error())
		// By returning a 404, code explorer still works if you select the
		// API in the URL
		return send_not_found_response(w, nil)
	}
	return send_success_response(directory, w)
}

// Returns the result of a discovery service request.
//
// This calls start_response and returns the response body.
//
// Args:
//   path: A string containing the SPI API path (the portion of the path
//     after /_ah/spi/).
//   request: An ApiRequest, the transformed request sent to the Discovery SPI.
//   start_response:
//
// Returns:
//   The response body. Or returns False if the request wasn't handled by
//   DiscoveryService.
//func (ds *DiscoveryService) handle_discovery_request(path string, request *ApiRequest, w http.ResponseWriter) bool {
//	handled := true
//	switch path {
//	case _GET_REST_API:
//		/*return */ds.get_rpc_or_rest("rest", request, w)
//	case _GET_RPC_API:
//		/*return */ds.get_rpc_or_rest("rpc", request, w)
//	case _LIST_API:
//		/*return */ds.list(w)
//	default:
//		handled = false
//	}
//	return handled
//}
func (ds *DiscoveryService) handle_discovery_request(path string, request *ApiRequest, w http.ResponseWriter) (string, bool) {
	switch path {
	case _GET_REST_API:
		return ds.get_rpc_or_rest("rest", request, w), true
	case _GET_RPC_API:
		return ds.get_rpc_or_rest("rpc", request, w), true
	case _LIST_API:
		return ds.list(w), true
	}
	return "", false
}
