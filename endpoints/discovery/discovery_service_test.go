
package discovery

import (
	"encoding/json"
	"testing"
	"net/http"
	"fmt"
)

func common_setup() (*ApiConfigManager, *ApiRequest) {
	api_config_dict = {"items": [api_config_json]}
	api_config_manager := NewApiConfigManager()
	api_config, _ := json.Marshal(api_config_dict)
	api_config_manager.parse_api_config_response(api_config)
	api_request := build_request("/_ah/api/foo",
		`{"api": "tictactoe", "version": "v1"}`)
	return api_config_manager, api_request
}

func prepare_discovery_request(response_body string, api_config_manager *ApiConfigManager) *DiscoveryService {
	response := test_utils.MockConnectionResponse(200, response_body)
	discovery := NewDiscoveryService(api_config_manager)
	discovery._discovery_proxy = mox.CreateMock(NewDiscoveryApiProxy())
	return discovery
}

func test_generate_discovery_doc_rest(t *testing.T) {
	api_config_manager, api_request := common_setup()
	body, err := json.Marshal(JsonObject{
		"baseUrl": "https://tictactoe.appspot.com/_ah/api/tictactoe/v1/",
	})
	discovery := prepare_discovery_request(body, api_config_manager)
	discovery._discovery_proxy.generate_discovery_doc(mox.IsA(object), "rest").AndReturn(body)

	mox.ReplayAll()
	response := discovery.handle_discovery_request(_GET_REST_API, api_request, self.start_response)
	mox.VerifyAll()

	assert_http_match(t, response, 200,
		http.Header{
			"Content-Type": "application/json; charset=UTF-8",
			"Content-Length": fmt.Sprintf("%d", len(body)),
		}, body)
}

func test_generate_discovery_doc_rpc(t *testing.T) {
	api_config_manager, api_request := common_setup()
	body, _ := json.Marshal(JsonObject{
		"rpcUrl": "https://tictactoe.appspot.com/_ah/api/rpc",
	})
	discovery := prepare_discovery_request(body, api_config_manager)
	discovery._discovery_proxy.generate_discovery_doc(mox.IsA(object), "rpc").AndReturn(body)

	mox.ReplayAll()
	response := discovery.handle_discovery_request(_GET_RPC_API, api_request, self.start_response)
	mox.VerifyAll()

	assert_http_match(t, response, 200,
		http.Header{
			"Content-Type": "application/json; charset=UTF-8",
			"Content-Length": fmt.Sprintf("%d", len(body)),
		}, body)
}

func test_generate_discovery_doc_rest_unknown_api(t *testing.T) {
	api_config_manager, api_request := common_setup()
	request = build_request("/_ah/api/foo", `{"api": "blah", "version": "v1"}`)
	discovery_api = NewDiscoveryService(api_config_manager)
	discovery_api.handle_discovery_request(_GET_REST_API, request, self.start_response)
	if self.response_status != "404" {
		t.Fail()
	}
}

func test_generate_directory(t *testing.T) {
	api_config_manager, api_request := common_setup()
	body, _ := json.Marshal(JsonObject{"kind": "discovery#directoryItem"})
	discovery := prepare_discovery_request(body, api_config_manager)
	discovery._discovery_proxy.generate_directory(mox.IsA(list)).AndReturn(body)

	mox.ReplayAll()
	response := discovery.handle_discovery_request(_LIST_API, api_request, self.start_response)
	mox.VerifyAll()

	assert_http_match(t, response, 200,
		http.Header{
			"Content-Type": "application/json; charset=UTF-8",
			"Content-Length": fmt.Sprintf("%d", len(body)),
		}, body)
}
