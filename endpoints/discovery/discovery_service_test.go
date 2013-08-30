package discovery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func common_setup() (*ApiConfigManager, *ApiRequest, *DiscoveryService) {
	api_config_dict := JsonObject{"items": []string{api_config_json}}
	api_config_manager := NewApiConfigManager()
	api_config, _ := json.Marshal(api_config_dict)
	api_config_manager.parse_api_config_response(string(api_config))

	api_request := build_request("/_ah/api/foo",
		`{"api": "tictactoe", "version": "v1"}`, nil)

	discovery := NewDiscoveryService(api_config_manager)
	//discovery._discovery_proxy = mox.CreateMock(NewDiscoveryApiProxy())

	return api_config_manager, api_request, discovery
}

func prepare_discovery_request(response_body string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, response_body)
	}))
	// Call ts.Close() when finished, to shut down the test server.

	//response := test_utils.MockConnectionResponse(200, response_body)
	return ts
}

func Test_generate_discovery_doc_rest_service(t *testing.T) {
	_, api_request, discovery := common_setup()
	body, _ := json.Marshal(JsonObject{
		"baseUrl": "https://tictactoe.appspot.com/_ah/api/tictactoe/v1/",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, string(body))
	}))
	defer ts.Close()
	_DISCOVERY_PROXY_HOST = ts.URL

	w := httptest.NewRecorder()

	//discovery := prepare_discovery_request(body, api_config_manager)
	//discovery._discovery_proxy.generate_discovery_doc(mox.IsA(object), "rest").AndReturn(body)

	//mox.ReplayAll()
	discovery.handle_discovery_request(_GET_REST_API, api_request, w)
	//mox.VerifyAll()

	assert_http_match_recorder(t, w, 200,
		http.Header{
			"Content-Type":   []string{"application/json; charset=UTF-8"},
			"Content-Length": []string{fmt.Sprintf("%d", len(body))},
		}, string(body))
}

func Test_generate_discovery_doc_rpc_service(t *testing.T) {
	_, api_request, discovery := common_setup()
	body, _ := json.Marshal(JsonObject{
		"rpcUrl": "https://tictactoe.appspot.com/_ah/api/rpc",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, string(body))
	}))
	defer ts.Close()
	_DISCOVERY_PROXY_HOST = ts.URL

	w := httptest.NewRecorder()

	//discovery := prepare_discovery_request(body, api_config_manager)
	//discovery._discovery_proxy.generate_discovery_doc(mox.IsA(object), "rpc").AndReturn(body)

	//mox.ReplayAll()
	discovery.handle_discovery_request(_GET_RPC_API, api_request, w)
	//mox.VerifyAll()

	assert_http_match_recorder(t, w, 200,
		http.Header{
			"Content-Type":   []string{"application/json; charset=UTF-8"},
			"Content-Length": []string{fmt.Sprintf("%d", len(body))},
		}, string(body))
}

func Test_generate_discovery_doc_rest_unknown_api(t *testing.T) {
	_, _, discovery_api := common_setup()
	request := build_request("/_ah/api/foo",
		`{"api": "blah", "version": "v1"}`, nil)
	w := httptest.NewRecorder()
	//discovery_api = NewDiscoveryService(api_config_manager)
	discovery_api.handle_discovery_request(_GET_REST_API, request, w)
	if w.Code != 404 {
		t.Fail()
	}
}

func Test_generate_directory(t *testing.T) {
	_, api_request, discovery := common_setup()
	body, _ := json.Marshal(JsonObject{"kind": "discovery#directoryItem"})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, string(body))
	}))
	defer ts.Close()
	_DISCOVERY_PROXY_HOST = ts.URL

	w := httptest.NewRecorder()

	//discovery := prepare_discovery_request(body, api_config_manager)
	//discovery._discovery_proxy.generate_directory(mox.IsA(list)).AndReturn(body)

	//mox.ReplayAll()
	discovery.handle_discovery_request(_LIST_API, api_request, w)
	//mox.VerifyAll()

	assert_http_match_recorder(t, w, 200,
		http.Header{
			"Content-Type":   []string{"application/json; charset=UTF-8"},
			"Content-Length": []string{fmt.Sprintf("%d", len(body))},
		}, string(body))
}
