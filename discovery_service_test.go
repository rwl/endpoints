package endpoint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/stretchr/testify/assert"
)

func commonSetup() (*ApiConfigManager, *ApiRequest, *DiscoveryService) {
	apiConfigMap := map[string]interface{}{"items": []string{apiConfigJson}}
	apiConfigManager := NewApiConfigManager()
	apiConfig, _ := json.Marshal(apiConfigMap)
	apiConfigManager.parseApiConfigResponse(string(apiConfig))

	apiRequest := buildApiRequest("/_ah/api/foo",
		`{"api": "tictactoe", "version": "v1"}`, nil)

	discovery := NewDiscoveryService(apiConfigManager)

	return apiConfigManager, apiRequest, discovery
}

func TestGenerateDiscoveryDocRestService(t *testing.T) {
	_, apiRequest, discovery := commonSetup()
	body, _ := json.Marshal(map[string]interface{}{
		"baseUrl": "https://tictactoe.appspot.com/_ah/api/tictactoe/v1/",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, string(body))
	}))
	defer ts.Close()
	DiscoveryProxyHost = ts.URL

	w := httptest.NewRecorder()

	discovery.handleDiscoveryRequest(GET_REST_API, apiRequest, w)

	assertHttpMatchRecorder(t, w, 200,
		http.Header{
			"Content-Type":   []string{"application/json; charset=UTF-8"},
			"Content-Length": []string{fmt.Sprintf("%d", len(body))},
		}, string(body))
}

func TestGenerateDiscoveryDocRpcService(t *testing.T) {
	_, apiRequest, discovery := commonSetup()
	body, _ := json.Marshal(map[string]interface{}{
		"rpcUrl": "https://tictactoe.appspot.com/_ah/api/rpc",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, string(body))
	}))
	defer ts.Close()
	DiscoveryProxyHost = ts.URL

	w := httptest.NewRecorder()

	discovery.handleDiscoveryRequest(GET_RPC_API, apiRequest, w)

	assertHttpMatchRecorder(t, w, 200,
		http.Header{
			"Content-Type":   []string{"application/json; charset=UTF-8"},
			"Content-Length": []string{fmt.Sprintf("%d", len(body))},
		}, string(body))
}

func TestGenerateDiscoveryDocRestUnknownApi(t *testing.T) {
	_, _, discoveryApi := commonSetup()
	request := buildApiRequest("/_ah/api/foo",
		`{"api": "blah", "version": "v1"}`, nil)
	w := httptest.NewRecorder()
	discoveryApi.handleDiscoveryRequest(GET_REST_API, request, w)
	assert.Equal(t, w.Code, 404)
}

func TestGenerateDirectory(t *testing.T) {
	_, apiRequest, discovery := commonSetup()
	body, _ := json.Marshal(map[string]interface{}{"kind": "discovery#directoryItem"})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, string(body))
	}))
	defer ts.Close()
	DiscoveryProxyHost = ts.URL

	w := httptest.NewRecorder()

	discovery.handleDiscoveryRequest(LIST_API, apiRequest, w)

	assertHttpMatchRecorder(t, w, 200,
		http.Header{
			"Content-Type":   []string{"application/json; charset=UTF-8"},
			"Content-Length": []string{fmt.Sprintf("%d", len(body))},
		}, string(body))
}
