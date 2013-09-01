package discovery

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
)

func Test_parse_no_body(t *testing.T) {
	request := build_request("/_ah/api/foo?bar=baz", "", nil)
	assert.Equal(t, "foo", request.URL.Path)
	assert.Equal(t, "bar=baz", request.URL.RawQuery)
	query := url.Values{"bar": []string{"baz"}}
	assert.Equal(t, query, request.URL.Query())
	body, err := ioutil.ReadAll(request.Body)
	assert.NoError(t, err)
	assert.Empty(t, body)
	assert.Equal(t, request.body_json, make(map[string]interface{}))
	header := make(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	assert.Equal(t, header, request.Header)
	assert.Empty(t, request.request_id)
}

func Test_parse_with_body(t *testing.T) {
	request := build_request("/_ah/api/foo?bar=baz", `{"test": "body"}`, nil)
	assert.Equal(t, "foo", request.URL.Path)
	assert.Equal(t, "bar=baz", request.URL.RawQuery)
	params := url.Values{"bar": []string{"baz"}}
	assert.Equal(t, params, request.URL.Query())
	body, err := ioutil.ReadAll(request.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"test": "body"}`, string(body))
	body_json := map[string]interface{}{"test": "body"}
	assert.Equal(t, body_json, request.body_json)
	header := make(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	assert.Equal(t, header, request.Header)
	assert.Empty(t, request.request_id)
}

func Test_parse_empty_values(t *testing.T) {
	request := build_request("/_ah/api/foo?bar", "", nil)
	assert.Equal(t, "foo", request.URL.Path)
	assert.Equal(t, "bar", request.URL.RawQuery)
	params := url.Values{"bar": []string{""}}
	assert.Equal(t, params, request.URL.Query())
	body, err := ioutil.ReadAll(request.Body)
	assert.NoError(t, err)
	assert.Empty(t, body)
	assert.Equal(t, map[string]interface{}{}, request.body_json)
	header := make(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	assert.Equal(t, header, request.Header)
	assert.Empty(t, request.request_id)
}

func Test_parse_multiple_values(t *testing.T) {
	request := build_request("/_ah/api/foo?bar=baz&foo=bar&bar=foo", "", nil)
	assert.Equal(t, "foo", request.URL.Path)
	assert.Equal(t, "bar=baz&foo=bar&bar=foo", request.URL.RawQuery)
	params := url.Values{
		"bar": []string{"baz", "foo"},
		"foo": []string{"bar"},
	}
	assert.Equal(t, params, request.URL.Query())
	body, err := ioutil.ReadAll(request.Body)
	assert.NoError(t, err)
	assert.Empty(t, body)
	assert.Equal(t, map[string]interface{}{}, request.body_json)
	header := make(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	assert.Equal(t, header, request.Header)
	assert.Empty(t, request.request_id)
}

func Test_is_rpc(t *testing.T) {
	request := build_request("/_ah/api/rpc", "", nil)
	assert.Equal(t, "rpc", request.URL.Path)
	assert.Empty(t, request.URL.RawQuery)
	assert.True(t, request.is_rpc())
}

func Test_is_not_rpc(t *testing.T) {
	request := build_request("/_ah/api/guestbook/v1/greetings/7", "", nil)
	assert.Equal(t, "guestbook/v1/greetings/7", request.URL.Path)
	assert.Empty(t, request.URL.RawQuery)
	assert.False(t, request.is_rpc())
}

func Test_is_not_rpc_prefix(t *testing.T) {
	request := build_request("/_ah/api/rpcthing", "", nil)
	assert.Equal(t, "rpcthing", request.URL.Path)
	assert.Empty(t, request.URL.RawQuery)
	assert.False(t, request.is_rpc())
}

func Test_batch(t *testing.T) {
	request := build_request("/_ah/api/rpc",
		`[{"method": "foo", "apiVersion": "v1"}]`, nil)
	assert.True(t, request.is_batch)
	/*if isinstance(request.body_json, list) {
		t.Fail()
	}*/
}

// Verify that additional items are dropped if the batch size is > 1.
func Test_batch_too_large(t *testing.T) {
	request := build_request("/_ah/api/rpc",
		`[{"method": "foo", "apiVersion": "v1"},
		  {"method": "bar", "apiversion": "v1"}]`, nil)
	assert.True(t, request.is_batch)
	var body_json map[string]interface{}
	err := json.Unmarshal([]byte(`{"method": "foo", "apiVersion": "v1"}`),
		&body_json)
	assert.NoError(t, err)
	assert.Equal(t, body_json, request.body_json)
}

/*func Test_source_ip(t *testing.T) {
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

func Test_copy(t *testing.T) {
	request := build_request("/_ah/api/foo?bar=baz", `{"test": "body"}`, nil)
	copied, err := request.copy()
	assert.NoError(t, err)
	assert.Equal(t, request.Header, copied.Header)
	body, err := ioutil.ReadAll(request.Body)
	assert.NoError(t, err)
	body_copy, err2 := ioutil.ReadAll(copied.Body)
	assert.NoError(t, err2)
	assert.Equal(t, string(body), string(body_copy))
	assert.Equal(t, request.body_json, copied.body_json)
	assert.Equal(t, request.URL.Path, copied.URL.Path)

	copied.Header.Set("Content-Type", "text/plain")
	copied.Body = ioutil.NopCloser(bytes.NewBufferString("Got a whole new body!"))
	copied.body_json = map[string]interface{}{"new": "body"}
	copied.URL.Path = "And/a/new/path/"

	assert.NotEqual(t, request.Header, copied.Header)
	body, err = ioutil.ReadAll(request.Body)
	assert.NoError(t, err)
	body_copy, err2 = ioutil.ReadAll(copied.Body)
	assert.NoError(t, err2)
	assert.NotEqual(t, string(body), string(body_copy))
	assert.NotEqual(t, request.body_json, copied.body_json)
	assert.NotEqual(t, request.URL.Path, copied.URL.Path)
}
