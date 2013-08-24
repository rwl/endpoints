
package discovery

import (
	"testing"
	"net/http"
	"encoding/json"
	"net/url"
	"reflect"
	"io/ioutil"
	"bytes"
)

func test_parse_no_body(t *testing.T) {
	request := build_request("/_ah/api/foo?bar=baz", "", nil)
	if "foo" != request.URL.Path {
		t.Fail()
	}
	if "bar=baz" != request.URL.RawQuery {
		t.Fail()
	}
	query := url.Values{"bar": []string{"baz"}}
	if !reflect.DeepEqual(query, request.URL.Query()) {
		t.Fail()
	}
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		t.Fail()
	}
	if "" != string(body) {
		t.Fail()
	}
	if !reflect.DeepEqual(request.body_json, make(JsonObject)) {
		t.Fail()
	}
	header := new(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	if !reflect.DeepEqual(header, request.Header) {
		t.Fail()
	}
	if "" != request.request_id {
		t.Fail()
	}
}

func test_parse_with_body(t *testing.T) {
	request := build_request("/_ah/api/foo?bar=baz", `{"test": "body"}`, nil)
	if "foo" != request.URL.Path {
		t.Fail()
	}
	if "bar=baz" != request.URL.RawQuery {
		t.Fail()
	}
	params := JsonObject{"bar": []string{"baz"}}
	if !reflect.DeepEqual(params, request.URL.Query()) {
		t.Fail()
	}
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		t.Fail()
	}
	if `{"test": "body"}` != string(body) {
		t.Fail()
	}
	body_json := JsonObject{"test": "body"}
	if !reflect.DeepEqual(body_json, request.body_json) {
		t.Fail()
	}
	header := new(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	if !reflect.DeepEqual(header, request.Header) {
		t.Fail()
	}
	if "" != request.request_id {
		t.Fail()
	}
}

func test_parse_empty_values(t *testing.T) {
	request := build_request("/_ah/api/foo?bar", "", nil)
	if "foo" != request.URL.Path {
		t.Fail()
	}
	if "bar" != request.URL.RawQuery {
		t.Fail()
	}
	params := JsonObject{"bar": []string{""}}
	if !reflect.DeepEqual(params, request.URL.Query()) {
		t.Fail()
	}
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		t.Fail()
	}
	if "" != string(body) {
		t.Fail()
	}
	if len(request.body_json) != 0 {
		t.Fail()
	}
	header := new(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	if !reflect.DeepEqual(header, request.Header) {
		t.Fail()
	}
	if "" != request.request_id {
		t.Fail()
	}
}

func test_parse_multiple_values(t *testing.T) {
	request := build_request("/_ah/api/foo?bar=baz&foo=bar&bar=foo", "", nil)
	if "foo" != request.URL.Path {
		t.Fail()
	}
	if "bar=baz&foo=bar&bar=foo" != request.URL.RawQuery {
		t.Fail()
	}
	params := JsonObject{
		"bar": []string{"baz", "foo"},
		"foo": []string{"bar"},
	}
	if !reflect.DeepEqual(params, request.URL.Query()) {
		t.Fail()
	}
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		t.Fail()
	}
	if "" != string(body) {
		t.Fail()
	}
	if len(request.body_json) != 0 {
		t.Fail()
	}
	header := new(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	if !reflect.DeepEqual(header, request.Header) {
		t.Fail()
	}
	if "" != request.request_id {
		t.Fail()
	}
}

func test_is_rpc(t *testing.T) {
	request := build_request("/_ah/api/rpc", "", nil)
	if "rpc" != request.URL.Path {
		t.Fail()
	}
	if "" != request.URL.RawQuery {
		t.Fail()
	}
	if !request.is_rpc() {
		t.Fail()
	}
}

func test_is_not_rpc(t *testing.T) {
	request := build_request("/_ah/api/guestbook/v1/greetings/7", "", nil)
	if "guestbook/v1/greetings/7" != request.URL.Path {
		t.Fail()
	}
	if "" != request.URL.RawQuery {
		t.Fail()
	}
	if request.is_rpc() {
		t.Fail()
	}
}

func test_is_not_rpc_prefix(t *testing.T) {
	request := build_request("/_ah/api/rpcthing", "", nil)
	if "rpcthing" != request.URL.Path {
		t.Fail()
	}
	if "" != request.URL.RawQuery {
		t.Fail()
	}
	if request.is_rpc() {
		t.Fail()
	}
}

func test_batch(t *testing.T) {
	request := build_request("/_ah/api/rpc",
		`[{"method": "foo", "apiVersion": "v1"}]`, nil)
	if !request.is_batch {
		t.Fail()
	}
	/*if isinstance(request.body_json, list) {
		t.Fail()
	}*/
}

// Verify that additional items are dropped if the batch size is > 1.
func test_batch_too_large(t *testing.T) {
	request := build_request("/_ah/api/rpc",
		`[{"method": "foo", "apiVersion": "v1"},
		  {"method": "bar", "apiversion": "v1"}]`, nil)
	if !request.is_batch {
		t.Fail()
	}
	var body_json JsonObject
	json.Unmarshal([]byte(`{"method": "foo", "apiVersion": "v1"}`), &body_json)
	if !reflect.DeepEqual(body_json, request.body_json) {
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
	request := build_request("/_ah/api/foo?bar=baz", `{"test": "body"}`, nil)
	copied := request.copy()
	if reflect.DeepEqual(request.Header, copied.Header) {
		t.Fail()
	}
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		t.Fail()
	}
	body_copy, err2 := ioutil.ReadAll(copied.Body)
	if err2 != nil {
		t.Fail()
	}
	if string(body) != string(body_copy) {
		t.Fail()
	}
	if !reflect.DeepEqual(request.body_json, copied.body_json) {
		t.Fail()
	}
	if request.URL.Path != copied.URL.Path {
		t.Fail()
	}

	copied.Header.Set("Content-Type", "text/plain")
	copied.Body = ioutil.NopCloser(bytes.NewBufferString("Got a whole new body!"))
	copied.body_json = JsonObject{"new": "body"}
	copied.URL.Path = "And/a/new/path/"

	if reflect.DeepEqual(request.Header, copied.Header) {
		t.Fail()
	}
	body, err = ioutil.ReadAll(request.Body)
	if err != nil {
		t.Fail()
	}
	body_copy, err2 = ioutil.ReadAll(copied.Body)
	if err2 != nil {
		t.Fail()
	}
	if string(body) == string(body_copy) {
		t.Fail()
	}
	if reflect.DeepEqual(request.body_json, copied.body_json) {
		t.Fail()
	}
	if request.URL.Path == copied.URL.Path {
		t.Fail()
	}
}
