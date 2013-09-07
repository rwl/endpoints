package endpoints

import (
	"encoding/json"
	"fmt"
	"github.com/crhym3/go-endpoints/endpoints"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/stretchr/testify/assert"
)

var api_config_map endpoints.ApiDescriptor

func init() {
	json.Unmarshal([]byte(api_config_json), &api_config_map)
}

//func prepare_discovery_request(status_code int, body string) *httptest.Server {
//	/*self.mox.StubOutWithMock(httplib.HTTPSConnection, "request")
//	self.mox.StubOutWithMock(httplib.HTTPSConnection, "getresponse")
//	self.mox.StubOutWithMock(httplib.HTTPSConnection, "close")
//
//	httplib.HTTPSConnection.request(mox.IsA(basestring), mox.IsA(basestring),
//		mox.IgnoreArg(), mox.IsA(dict))
//	httplib.HTTPSConnection.getresponse().AndReturn(
//		test_utils.MockConnectionResponse(status_code, body))
//	httplib.HTTPSConnection.close()*/
//
//	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
//		if status_code != 200 {
//			http.Error(w, "Error", status_code)
//		} else {
//			fmt.Fprintf(w, body)
//		}
//	}))
//	return ts
//}

func Test_generate_discovery_doc_rest(t *testing.T) {
	//discovery_api := &DiscoveryApiProxy{}
	baseUrl := "https://tictactoe.appspot.com/_ah/api/tictactoe/v1/"

	body := map[string]interface{}{"baseUrl": baseUrl}
	body_json, _ := json.Marshal(body)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, string(body_json))
	}))
	//ts := prepare_discovery_request(200, body_json)
	defer ts.Close()
	_DISCOVERY_PROXY_HOST = ts.URL

	//self.mox.ReplayAll()
	doc, err := generate_discovery_doc(&api_config_map, "rest")
	//self.mox.VerifyAll()

	assert.NoError(t, err)
	assert.NotEmpty(t, doc)

	var api_config map[string]interface{}
	err = json.Unmarshal([]byte(doc), &api_config)
	assert.NoError(t, err)
	assert.Equal(t, api_config["baseUrl"], baseUrl)
}

func Test_generate_discovery_doc_rpc(t *testing.T) {
	rpcUrl := "https://tictactoe.appspot.com/_ah/api/rpc"
	body := map[string]interface{}{"rpcUrl": rpcUrl}
	body_json, _ := json.Marshal(body)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, string(body_json))
	}))
	//ts := prepare_discovery_request(200, body_json)
	defer ts.Close()
	_DISCOVERY_PROXY_HOST = ts.URL

	//self.mox.ReplayAll()
	doc, err := generate_discovery_doc(&api_config_map, "rpc")
	//self.mox.VerifyAll()

	assert.NoError(t, err)
	assert.NotEmpty(t, doc)

	var api_config map[string]interface{}
	err = json.Unmarshal([]byte(doc), &api_config)
	assert.NoError(t, err)
	assert.Equal(t, api_config["rpcUrl"], rpcUrl)
}

func Test_generate_discovery_doc_invalid_format(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Error", 400)
	}))
	defer ts.Close()

	_DISCOVERY_PROXY_HOST = ts.URL
	//_DISCOVERY_API_PATH_PREFIX = ""

	_, err := generate_discovery_doc(&api_config_map, "blah")
	assert.Error(t, err)
}

func Test_generate_discovery_doc_bad_api_config(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "", 503)
	}))
	defer ts.Close()
	_DISCOVERY_PROXY_HOST = ts.URL

	bad := &endpoints.ApiDescriptor{
		Name: "none",
	}
	//mox.ReplayAll()
	doc, err := generate_discovery_doc(bad, "rpc")
	//self.mox.VerifyAll()

	assert.Error(t, err)
	assert.Empty(t, doc, "")
}

func Test_get_static_file_existing(t *testing.T) {
	body := "static file body"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, body)
	}))
	//prepare_discovery_request(200, body)
	defer ts.Close()
	_STATIC_PROXY_HOST = ts.URL

	//mox.ReplayAll()
	response, response_body, err := get_static_file("/_ah/api/static/proxy.html")
	//self.mox.VerifyAll()

	assert.NoError(t, err)
	assert.Equal(t, response.StatusCode, 200)
	assert.Equal(t, body, response_body)
}
