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

// Regression tests for Endpoints server.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

// Test that a GET request to a REST API works.
func TestRestGet(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	resp, err := http.Get(ts.URL + "/_ah/api/test_service/v1/test")
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(body, &responseJson)
	assert.NoError(t, err)

	//text, ok := responseJson["text"]
	//assert.True(t, ok)
	//textStr, ok := text.(string)
	//assert.True(t, ok)
	//assert.Equal(t, textStr, "Test response")

	expected := map[string]interface{}{"text": "Test response"}
	assert.Equal(t, expected, responseJson)
}

// Test that a POST request to a REST API works.
func TestRestPost(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	body, err := json.Marshal(map[string]interface{}{
		"name":   "MyName",
		"number": 23,
	})
	assert.NoError(t, err)

	resp, err := http.Post(ts.URL + "/_ah/api/test_service/v1/t2path",
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

	//text, ok := responseJson["text"]
	//assert.True(t, ok)
	//textStr, ok := text.(string)
	//assert.True(t, ok)
	//assert.Equal(t, textStr, "MyName 23")

	expected := map[string]interface{}{"text": "MyName 23"}
	assert.Equal(t, expected, responseJson)
}

// Test that CORS headers are handled properly.
func TestCors(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	client := &http.Client{}

	req, err := http.NewRequest("GET",
			ts.URL + "/_ah/api/test_service/v1/test", nil)

	req.Header.Set("Origin", "test.com")
	req.Header.Set("Access-control-request-method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Date,Expires")

	resp, err := client.Do(req)

	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get(corsHeaderAllowOrigin), "test.com")
	assert.Equal(t, resp.Header.Get(corsHeaderAllowHeaders), "Date,Expires")

	allowed := strings.Split(resp.Header.Get(corsHeaderAllowMethods), ",")
	for _, header := range allowed {
		if header == "GET" {
			return
		}
	}
	t.Fail()
}

// Test that an RPC request works.
func TestRpc(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	body, err := json.Marshal([]map[string]interface{}{
		map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      "gapiRpc",
			"method":  "test_service.t2name",
			"params": map[string]interface{}{
				"name":   "MyName",
				"number": 23,
			},
			"apiVersion": "v1",
		},
	})
	assert.NoError(t, err)

	resp, err := http.Post(ts.URL + "/_ah/api/rpc",
		"application-rpc", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

	//jsonArray, ok := responseJson.([]map[string]interface{})
	//assert.True(t, ok)
	//assert.Equal(t, len(jsonArray), 1)
	//
	//result, ok := jsonArray[0]["result"]
	//assert.True(t, ok)
	//resultMap, ok := result.(map[string]interface{})
	//assert.True(t, ok)
	//text, ok := resultMap["text"]
	//assert.True(t, ok)
	//textStr, ok := text.(string)
	//assert.True(t, ok)
	//assert.Equals(t, textStr, "MyName 23")
	//
	//id, ok := jsonArray[0]["id"]
	//assert.True(t, ok)
	//idStr, ok := id.(string)
	//assert.True(t, ok)
	//assert.Equal(t, idStr, "gapiRpc")

	assert.Equal(t, []interface{}{
		map[string]interface{}{
			"result": map[string]interface{}{
				"text": "MyName 23",
			},
			"id": "gapiRpc",
		},
	}, responseJson)
}

// Test sending and receiving a datetime.
func TestEchoDatetimeMessage(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	body, err := json.Marshal(map[string]interface{}{
		"milliseconds":     5000,
		"time_zone_offset": 60,
	})
	assert.NoError(t, err)

	resp, err := http.Post(ts.URL + "/_ah/api/test_service/v1/echodatemessage",
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

	expected := map[string]interface{}{
		"milliseconds":     5000,
		"time_zone_offset": 60,
	}
	assert.Equal(t, expected, responseJson)
}

// Test sending and receiving a message that includes a time.
func TestEchoDatetimeField(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	bodyJson := map[string]interface{}{
		"date": "2013-03-13T15:29:37.883+08:00",
	}
	body, err := json.Marshal(bodyJson)
	assert.NoError(t, err)

	resp, err := http.Post(ts.URL + "/_ah/api/test_service/v1/echodatetimefield",
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)
	assert.Equal(t, bodyJson, responseJson)
}

// Test sending and receiving integer values.
func TestIncrementIntegers(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	bodyJson := map[string]interface{}{
		"var_int32": 100,
		"var_int64": "1000",
		"var_repeated_int64": []string{
			"10", "11", "900",
		},
		//"var_sint64": -555,
		"var_uint64": "4320",
	}
	body, err := json.Marshal(bodyJson)
	assert.NoError(t, err)

	resp, err := http.Post(ts.URL + "/_ah/api/test_service/v1/incrementintegers",
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

	expectedResponse := map[string]interface{}{
		"var_int32": 101,
		"var_int64": "1001",
		"var_repeated_int64": []interface{}{
			"11", "12", "901",
		},
		//"var_sint64": "-554",
		"var_uint64": "4321",
	}
	assert.Equal(t, expectedResponse, responseJson)
}

// Test sending and receiving a BytesField parameter.
func TestEchoBytes(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	value := []byte("This is a test of a message encoded as a BytesField.01234\000\001")
	bytesValue := base64.URLEncoding.EncodeToString(value)
	bodyJson := map[string]interface{}{"bytes_value": bytesValue}
	body, err := json.Marshal(bodyJson)
	assert.NoError(t, err)

	resp, err := http.Post(ts.URL + "/_ah/api/test_service/v1/echobytes",
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

	assert.Equal(t, responseJson, bodyJson)
	dec, err := base64.URLEncoding.DecodeString(bytesValue)
	assert.NoError(t, err)
	assert.Equal(t, value, dec)
}

// Test that an empty response that should have an object returns 200.
func TestEmptyTest(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	resp, err := http.Get(ts.URL + "/_ah/api/test_service/v1/emptytest")

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "2", resp.Header.Get("Content-Length"))

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, "{}", string(content))
}

// An empty response that should be empty should return 204.
func TestEmptyResponse(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	resp, err := http.Get(ts.URL + "/_ah/api/test_service/v1/empty_response")

	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, "0", resp.Header.Get("Content-Length"))

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, "", string(content))
}

// Test that the discovery configuration looks right.
func TestDiscoveryConfig(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	resp, err := http.Get(ts.URL + "/_ah/api/discovery/v1/apis/test_service/v1/rest")

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/json; charset=UTF-8",
		resp.Header.Get("Content-Type"))

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

	//assertRegexpMatches(responseJson["baseUrl"],
	//	`^http://localhost(:\d+)?/_ah/api/test_service/v1/$`)
	//assertRegexpMatches(responseJson["rootUrl"],
	//	`^http://localhost(:\d+)?/_ah/api/$`)
}

// Test that a GET request to a second class in the REST API works.
func TestMulticlassRestGet(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	resp, err := http.Get(ts.URL + "/_ah/api/test_service/v1/extrapath/test")
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)
	expected := map[string]string{"text": "Extra test response"}
	assert.Equal(t, expected, responseJson)
}

// Test that an RPC request to a second class in the API works.
func TestMulticlassRpc(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	body, err := json.Marshal([]map[string]interface{}{
		map[string]interface{}{
			"jsonrpc":    "2.0",
			"id":         "gapiRpc",
			"method":     "testservice.extraname.test",
			"params":     make(map[string]interface{}),
			"apiVersion": "v1",
		},
	})
	assert.NoError(t, err)

	resp, err := http.Post(ts.URL + "/_ah/api/rpc",
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &responseJson)
	expected := []map[string]interface{}{
		map[string]interface{}{
			"result": map[string]interface{}{
				"text": "Extra test response",
			},
			"id": "gapiRpc",
		},
	}
	assert.Equal(t, expected, responseJson)
}

// Test that a GET request to a second similar API works.
func TestSecondApiNoCollision(t *testing.T) {
	//ts := initTestApi(t)
	//defer ts.Close()

	resp, err := http.Get(ts.URL + "/_ah/api/second_service/v1/test")

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)
	expected := map[string]interface{}{"text": "Second response"}
	assert.Equal(t, expected, responseJson)
}
