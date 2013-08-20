
package discovery

import (
	"testing"
	"encoding/json"
)

var api_config_map map[string]interface{}

func init() {
	json.Unmarshal(api_config_json, &api_config_map)
}

func prepare_discovery_request(status_code int, body string) {
	self.mox.StubOutWithMock(httplib.HTTPSConnection, "request")
	self.mox.StubOutWithMock(httplib.HTTPSConnection, "getresponse")
	self.mox.StubOutWithMock(httplib.HTTPSConnection, "close")

	httplib.HTTPSConnection.request(mox.IsA(basestring), mox.IsA(basestring),
		mox.IgnoreArg(), mox.IsA(dict))
	httplib.HTTPSConnection.getresponse().AndReturn(
		test_utils.MockConnectionResponse(status_code, body))
	httplib.HTTPSConnection.close()
}

func test_generate_discovery_doc_rest(t *testing.T) {
	discovery_api := &DiscoveryApiProxy{}
	baseUrl := "https://tictactoe.appspot.com/_ah/api/tictactoe/v1/"

	var body map[string]interface{}
	body["baseUrl"] = baseUrl
	prepare_discovery_request(200, json.dumps(body))

	self.mox.ReplayAll()
	doc = generate_discovery_doc(api_config_map, "rest")
	self.mox.VerifyAll()

	if doc == nil {
		t.Fail()
	}
	api_config := json.loads(doc)
	if api_config["baseUrl"] != baseUrl {
		t.Fail()
	}
}

func test_generate_discovery_doc_rpc(t *testing.T) {
	rpcUrl := "https://tictactoe.appspot.com/_ah/api/rpc"
	var body map[string]interface{}
	body["rpcUrl"] = rpcUrl
	prepare_discovery_request(200, json.dumps(body))

	self.mox.ReplayAll()
	doc = generate_discovery_doc(api_config_map, "rpc")
	self.mox.VerifyAll()

	if doc == nil {
		t.Fail()
	}
	api_config = json.loads(doc)
	if api_config["rpcUrl"] != rpcUrl {
		t.Fail()
	}
}

func test_generate_discovery_doc_invalid_format(t *testing.T) {
	discovery_api := &DiscoveryApiProxy{}

	_response := test_utils.MockConnectionResponse(400, "Error")
	doc, err := discovery_api.generate_discovery_doc(api_config_map, "blah")
	if err == nil {
		t.Fail()
	}
}

func test_generate_discovery_doc_bad_api_config(t *testing.T) {
	prepare_discovery_request(503, nil)

	mox.ReplayAll()
	doc, _ = generate_discovery_doc(`{ "name": "none" }`, "rpc")
	self.mox.VerifyAll()

	if doc != nil {
		t.Fail()
	}
}

func test_get_static_file_existing(t *testing.T) {
	body := "static file body"
	prepare_discovery_request(200, body)

	mox.ReplayAll()
	response, response_body = get_static_file("/_ah/api/static/proxy.html")
	self.mox.VerifyAll()

	if response.status != 200 {
		t.Fail()
	}
	if body != response_body {
		t.Fail()
	}
}
