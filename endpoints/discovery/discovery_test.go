
package discovery

import (
	"testing"
	"encoding/json"
	"net/http/httptest"
	"net/http"
	"fmt"
)

var api_config_map map[string]interface{}

func init() {
	json.Unmarshal(api_config_json, &api_config_map)
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

func test_generate_discovery_doc_rest(t *testing.T) {
//	discovery_api := &DiscoveryApiProxy{}
	baseUrl := "https://tictactoe.appspot.com/_ah/api/tictactoe/v1/"

	body := JsonObject{"baseUrl": baseUrl}
	body_json, _ := json.Marshal(body)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, body_json)
	}))
//	ts := prepare_discovery_request(200, body_json)
	defer ts.Close()
	_DISCOVERY_PROXY_HOST = ts.URL

//	self.mox.ReplayAll()
	doc, err := generate_discovery_doc(api_config_map, "rest")
//	self.mox.VerifyAll()

	if err != nil {
		t.Fail()
	}
	if doc == nil {
		t.Fail()
	}
	var api_config JsonObject
	err = json.Unmarshal(doc, api_config)
	if err != nil {
		t.Fail()
	}
	if api_config["baseUrl"] != baseUrl {
		t.Fail()
	}
}

func test_generate_discovery_doc_rpc(t *testing.T) {
	rpcUrl := "https://tictactoe.appspot.com/_ah/api/rpc"
	body := JsonObject{"rpcUrl": rpcUrl}
	body_json, _ := json.Marshal(body)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, body_json)
	}))
//	ts := prepare_discovery_request(200, body_json)
	defer ts.Close()
	_DISCOVERY_PROXY_HOST = ts.URL

//	self.mox.ReplayAll()
	doc, err := generate_discovery_doc(api_config_map, "rpc")
//	self.mox.VerifyAll()

	if err != nil {
		t.Fail()
	}
	if doc == nil {
		t.Fail()
	}
	var api_config JsonObject
	err = json.Unmarshal(doc, api_config)
	if err != nil {
		t.Fail()
	}
	if api_config["rpcUrl"] != rpcUrl {
		t.Fail()
	}
}

func test_generate_discovery_doc_invalid_format(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Error", 400)
	}))
	defer ts.Close()

	_DISCOVERY_PROXY_HOST = ts.URL
//	_DISCOVERY_API_PATH_PREFIX = ""

	_, err := generate_discovery_doc(api_config_map, "blah")
	if err == nil {
		t.Fail()
	}
}

func test_generate_discovery_doc_bad_api_config(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "", 503)
	}))
	defer ts.Close()
	_DISCOVERY_PROXY_HOST = ts.URL

//	mox.ReplayAll()
	doc, err := generate_discovery_doc(`{ "name": "none" }`, "rpc")
//	self.mox.VerifyAll()

	if err == nil {
		t.Fail()
	}
	if doc != nil {
		t.Fail()
	}
}

func test_get_static_file_existing(t *testing.T) {
	body := "static file body"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, body)
	}))
//	prepare_discovery_request(200, body)
	defer ts.Close()
	_STATIC_PROXY_HOST = ts.URL

//	mox.ReplayAll()
	response, response_body, err := get_static_file("/_ah/api/static/proxy.html")
//	self.mox.VerifyAll()

	if err != nil {
		t.Fail()
	}
	if response.StatusCode != 200 {
		t.Fail()
	}
	if body != response_body {
		t.Fail()
	}
}
