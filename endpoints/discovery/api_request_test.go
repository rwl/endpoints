
package discovery

import (
	"testing"
	"net/http"
	"encoding/json"
)

func test_parse_no_body(t *testing.T) {
	request := build_request("/_ah/api/foo?bar=baz")
	if "foo" != request.Path {
		t.Fail()
	}
	if "bar=baz" != request.Query {
		t.Fail()
	}
	query := JsonObject{"bar": []string{"baz"}}
	if query != request.Query() {
		t.Fail()
	}
	if "" != request.Body {
		t.Fail()
	}
	if request.body_json != make(JsonObject) {
		t.Fail()
	}
	header := new(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	if header != request.Header {
		t.Fail()
	}
	if nil != request.request_id {
		t.Fail()
	}
}

func test_parse_with_body(t *testing.T) {
	request := build_request("/_ah/api/foo?bar=baz", `{"test": "body"}`)
	if "foo" != request.Path {
		t.Fail()
	}
	if "bar=baz" != request.RawQuery {
		t.Fail()
	}
	params := JsonObject{"bar": []string{"baz"}}
	if params != request.Query() {
		t.Fail()
	}
	if `{"test": "body"}` != request.Body {
		t.Fail()
	}
	body_json := JsonObject{"test": "body"}
	if body_json != request.body_json {
		t.Fail()
	}
	header := new(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	if header != request.Header {
		t.Fail()
	}
	if nil != request.request_id {
		t.Fail()
	}
}

func test_parse_empty_values(t *testing.T) {
	request := build_request("/_ah/api/foo?bar")
	if "foo" != request.Path {
		t.Fail()
	}
	if "bar" != request.RawQuery {
		t.Fail()
	}
	params := JsonObject{"bar": []string{""}}
	if params != request.Query() {
		t.Fail()
	}
	if "" != request.Body {
		t.Fail()
	}
	if new(JsonObject) != request.body_json {
		t.Fail()
	}
	header := new(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	if header != request.Header {
		t.Fail()
	}
	if nil != request.request_id {
		t.Fail()
	}
}

func test_parse_multiple_values(t *testing.T) {
	request := build_request("/_ah/api/foo?bar=baz&foo=bar&bar=foo")
	if "foo" != request.Path {
		t.Fail()
	}
	if "bar=baz&foo=bar&bar=foo" != request.RawQuery {
		t.Fail()
	}
	params := JsonObject{
		"bar": []string{"baz", "foo"},
		"foo": []string{"bar"},
	}
	if params != request.Query() {
		t.Fail()
	}
	if "" != request.Body {
		t.Fail()
	}
	if new(JsonObject) != request.body_json {
		t.Fail()
	}
	header := new(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	if header != request.Header {
		t.Fail()
	}
	if nil != request.request_id {
		t.Fail()
	}
}

func test_is_rpc(t *testing.T) {
	request := build_request("/_ah/api/rpc")
	if "rpc" != request.Path {
		t.Fail()
	}
	if nil != request.RawQuery {
		t.Fail()
	}
	if !request.is_rpc() {
		t.Fail()
	}
}

func test_is_not_rpc(t *testing.T) {
	request := build_request("/_ah/api/guestbook/v1/greetings/7")
	if "guestbook/v1/greetings/7" != request.Path {
		t.Fail()
	}
	if nil != request.RawQuery {
		t.Fail()
	}
	if request.is_rpc() {
		t.Fail()
	}
}

func test_is_not_rpc_prefix(t *testing.T) {
	request := build_request("/_ah/api/rpcthing")
	if "rpcthing" != request.Path {
		t.Fail()
	}
	if nil != request.RawQuery {
		t.Fail()
	}
	if request.is_rpc() {
		t.Fail()
	}
}

func test_batch(t *testing.T) {
	request := build_request("/_ah/api/rpc", `[{"method": "foo", "apiVersion": "v1"}]`)
	if !request.is_batch() {
		t.Fail()
	}
	/*if isinstance(request.body_json, list) {
		t.Fail()
	}*/
}

// Verify that additional items are dropped if the batch size is > 1.
func test_batch_too_large(t *testing.T) {
	request = test_utils.build_request("/_ah/api/rpc",
		`[{"method": "foo", "apiVersion": "v1"},
		  {"method": "bar", "apiversion": "v1"}]`)
	if !request.is_batch() {
		t.Fail()
	}
	var body_json JsonObject
	json.Unmarshal(`{"method": "foo", "apiVersion": "v1"}`, &body_json)
	if body_json != request.body_json {
		t.Fail()
	}
}

/*func test_source_ip(t *testing.T) {
	body := "{}"
	path := "/_ah/api/guestbook/v1/greetings"
	env = {"SERVER_PORT": 42, "REQUEST_METHOD": "GET",
	"SERVER_NAME": "localhost", "HTTP_CONTENT_TYPE": "application/json",
	"PATH_INFO": path, "wsgi.input": cStringIO.StringIO(body)}

	request = api_request.ApiRequest(env)
	self.assertEqual(request.source_ip, None)

	env["REMOTE_ADDR"] = "1.2.3.4"
	request = api_request.ApiRequest(env)
	self.assertEqual(request.source_ip, "1.2.3.4")
}*/

func test_copy(t *testing.T) {
	request := build_request("/_ah/api/foo?bar=baz", `{"test": "body"}`)
	copied := request.Copy()
	if request.Header != copied.Header {
		t.Fail()
	}
	if request.body != copied.body {
		t.Fail()
	}
	if request.body_json != copied.body_json {
		t.Fail()
	}
	if request.Path != copied.Path {
		t.Fail()
	}

	copied.Header.Set("Content-Type", "text/plain")
	copied.Body = "Got a whole new body!"
	copied.body_json = JsonObject{"new": "body"}
	copied.Path = "And/a/new/path/"

	if request.Header == copied.Header {
		t.Fail()
	}
	if request.Body == copied.Body {
		t.Fail()
	}
	if request.body_json == copied.body_json {
		t.Fail()
	}
	if request.Path == copied.Path {
		t.Fail()
	}
}
