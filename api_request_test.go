// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package endpoint

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
)

func TestParseNoBody(t *testing.T) {
	request := buildApiRequest("/_ah/api/foo?bar=baz", "", nil)
	assert.Equal(t, "foo", request.URL.Path)
	assert.Equal(t, "bar=baz", request.URL.RawQuery)
	query := url.Values{"bar": []string{"baz"}}
	assert.Equal(t, query, request.URL.Query())
	body, err := ioutil.ReadAll(request.Body)
	assert.NoError(t, err)
	assert.Empty(t, body)
	assert.Equal(t, request.BodyJson, make(map[string]interface{}))
	header := make(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	assert.Equal(t, header, request.Header)
	assert.Empty(t, request.RequestId)
}

func TestParseWithBody(t *testing.T) {
	request := buildApiRequest("/_ah/api/foo?bar=baz", `{"test": "body"}`, nil)
	assert.Equal(t, "foo", request.URL.Path)
	assert.Equal(t, "bar=baz", request.URL.RawQuery)
	params := url.Values{"bar": []string{"baz"}}
	assert.Equal(t, params, request.URL.Query())
	body, err := ioutil.ReadAll(request.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"test": "body"}`, string(body))
	bodyJson := map[string]interface{}{"test": "body"}
	assert.Equal(t, bodyJson, request.BodyJson)
	header := make(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	assert.Equal(t, header, request.Header)
	assert.Empty(t, request.RequestId)
}

func TestParseEmptyValues(t *testing.T) {
	request := buildApiRequest("/_ah/api/foo?bar", "", nil)
	assert.Equal(t, "foo", request.URL.Path)
	assert.Equal(t, "bar", request.URL.RawQuery)
	params := url.Values{"bar": []string{""}}
	assert.Equal(t, params, request.URL.Query())
	body, err := ioutil.ReadAll(request.Body)
	assert.NoError(t, err)
	assert.Empty(t, body)
	assert.Equal(t, map[string]interface{}{}, request.BodyJson)
	header := make(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	assert.Equal(t, header, request.Header)
	assert.Empty(t, request.RequestId)
}

func TestParseMultipleValues(t *testing.T) {
	request := buildApiRequest("/_ah/api/foo?bar=baz&foo=bar&bar=foo", "", nil)
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
	assert.Equal(t, map[string]interface{}{}, request.BodyJson)
	header := make(http.Header)
	header.Set("CONTENT-TYPE", "application/json")
	assert.Equal(t, header, request.Header)
	assert.Empty(t, request.RequestId)
}

func TestIsRPC(t *testing.T) {
	request := buildApiRequest("/_ah/api/rpc", "", nil)
	assert.Equal(t, "rpc", request.URL.Path)
	assert.Empty(t, request.URL.RawQuery)
	assert.True(t, request.IsRpc())
}

func TestIsNotRPC(t *testing.T) {
	request := buildApiRequest("/_ah/api/guestbook/v1/greetings/7", "", nil)
	assert.Equal(t, "guestbook/v1/greetings/7", request.URL.Path)
	assert.Empty(t, request.URL.RawQuery)
	assert.False(t, request.IsRpc())
}

func TestIsNotRPCPrefix(t *testing.T) {
	request := buildApiRequest("/_ah/api/rpcthing", "", nil)
	assert.Equal(t, "rpcthing", request.URL.Path)
	assert.Empty(t, request.URL.RawQuery)
	assert.False(t, request.IsRpc())
}

func TestBatch(t *testing.T) {
	request := buildApiRequest("/_ah/api/rpc",
		`[{"method": "foo", "apiVersion": "v1"}]`, nil)
	assert.True(t, request.IsBatch)
}

// Verify that additional items are dropped if the batch size is > 1.
func Test_batch_too_large(t *testing.T) {
	request := buildApiRequest("/_ah/api/rpc",
		`[{"method": "foo", "apiVersion": "v1"},
		  {"method": "bar", "apiversion": "v1"}]`, nil)
	assert.True(t, request.IsBatch)
	var bodyJson map[string]interface{}
	err := json.Unmarshal([]byte(`{"method": "foo", "apiVersion": "v1"}`),
		&bodyJson)
	assert.NoError(t, err)
	assert.Equal(t, bodyJson, request.BodyJson)
}

func TestCopy(t *testing.T) {
	request := buildApiRequest("/_ah/api/foo?bar=baz", `{"test": "body"}`, nil)
	copied, err := request.Copy()
	assert.NoError(t, err)
	assert.Equal(t, request.Header, copied.Header)
	body, err := ioutil.ReadAll(request.Body)
	assert.NoError(t, err)
	body_copy, err2 := ioutil.ReadAll(copied.Body)
	assert.NoError(t, err2)
	assert.Equal(t, string(body), string(body_copy))
	assert.Equal(t, request.BodyJson, copied.BodyJson)
	assert.Equal(t, request.URL.Path, copied.URL.Path)

	copied.Header.Set("Content-Type", "text/plain")
	copied.Body = ioutil.NopCloser(bytes.NewBufferString("Got a whole new body!"))
	copied.BodyJson = map[string]interface{}{"new": "body"}
	copied.URL.Path = "And/a/new/path/"

	assert.NotEqual(t, request.Header, copied.Header)
	body, err = ioutil.ReadAll(request.Body)
	assert.NoError(t, err)
	body_copy, err2 = ioutil.ReadAll(copied.Body)
	assert.NoError(t, err2)
	assert.NotEqual(t, string(body), string(body_copy))
	assert.NotEqual(t, request.BodyJson, copied.BodyJson)
	assert.NotEqual(t, request.URL.Path, copied.URL.Path)
}
