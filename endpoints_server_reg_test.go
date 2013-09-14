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

/*import (
	"encoding/json"
	"testing"
	"net/http"
	"strings"
)

// Test that a GET request to a REST API works.
func TestRestGet(t *testing.T) {
	status, content, headers := fetchUrl("default", "GET",
		"/_ah/api/test_service/v1/test")
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers["Content-Type"] {
		t.Fail()
	}

	var responseJson interface{}
	err := json.Unmarshal([]byte(content), &responseJson)
	if err != nil {
		t.Fail()
	}
	if map[string]string{"text": "Test response"} != responseJson {
		t.Fail()
	}
}

// Test that a POST request to a REST API works.
func TestRestPost(t *testing.T) {
	body, err := json.Marshal(map[string]interface{}{
		"name": "MyName",
		"number": 23,
	})
	sendJeaders := make(http.Header)
	sendHeader.Set("content-type", "application/json")
	status, content, headers := fetchUrl("default", "POST",
		"/_ah/api/test_service/v1/t2path",
		string(body), sendHeaders)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var responseJson interface{}
	err = json.Unmarshal(content, &responseJson)
	if map[string]string{"text": "MyName 23"} != responseJson {
		t.Fail()
	}
}

// Test that CORS headers are handled properly.
func TestCors(t *testing.T) {
	sendHeaders := make(http.Header)
	sendHeaders.Set("Origin", "test.com")
	sendHeaders.Set("Access-control-request-method", "GET")
	sendHeaders.Set("Access-Control-Request-Headers", "Date,Expires")
	status, _, headers := fetchUrl("default", "GET",
		"/_ah/api/test_service/v1/test", send_headers)
	if 200 != status {
		t.Fail()
	}
	if headers.Get(CORS_HEADER_ALLOW_ORIGIN) != "test.com" {
		t.Fail()
	}
	for _, header := range strings.Split(headers.Get(CorsAllowedMethods), ",") {
		if header == "GET" {
			goto P
		}
	}
	t.Fail()
P:
	if headers.Get(CORS_HEADER_ALLOW_HEADERS) != "Date,Expires" {
		t.Fail()
	}
}

// Test that an RPC request works.
func TestRpc(t *testing.T) {
	body, err := json.Marshal([]string{
		map[string]interface{}{
			"jsonrpc": "2.0",
			"id": "gapiRpc",
			"method": "testservice.t2name",
			"params": map[string]interface{}{
				"name": "MyName",
				"number": 23,
			},
			"apiVersion": "v1",
		},
	})
	sendHeaders := make(http.Header)
	sendHeaders.Set("content-type", "application-rpc")
	status, content, headers := fetchUrl("default", "POST",
		"/_ah/api/rpc",
		body, sendHeaders)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var responseJson interface{}
	err := json.Unmarshal(content, responseJson)
	if []string{
		map[string]interface{}{
			"result": map[string]string{
				"text": "MyName 23",
			},
			"id": "gapiRpc",
		},
	} != responseJson {
		t.Fail()
	}
}

// Test sending and receiving a datetime.
func TestEchoDatetimeMessage(t *testing.T) {
	body, err := json.Marshal(map[string]interface{}{
		"milliseconds": "5000",
		"time_zone_offset": "60",
	})
	sendHeaders := make(http.Header)
	sendHeaders.Set("content-type", "application/json")
	status, content, headers := fetchUrl(
		"default", "POST", "/_ah/api/test_service/v1/echo_datetime_message",
		body, sendHeaders)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var responseJson interface{}
	err := json.Unmarshal(content, &responseJson)
	if map[string]string{
		"milliseconds": "5000",
		"time_zone_offset": "60",
	} != responseJson {
		t.Fail()
	}
}

// Test sending and receiving a message that includes a datetime.
func TestEchoDatetimeField(t *testing.T) {
	bodyJson := map[string]string{
		"datetime_value": "2013-03-13T15:29:37.883000+08:00",
	}
	body, err := json.Marshal(bodyJson)
	sendHeaders := make(http.Header)
	sendHeaders.Set("content-type", "application/json")
	status, content, headers := fetchUrl(
		"default", "POST", "/_ah/api/test_service/v1/echo_datetime_field",
		body, sendHeaders)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var responseJson interface{}
	err := json.Unmarshal(content, &response_json)
	if bodyJson != responseJson {
		t.Fail()
	}
}

// Test sending and receiving integer values.
func TestIncrementIntegers(t *testing.T) {
	bodyJson := map[string]interface{}{
		"var_int32": 100,
		"var_int64": "1000",
		"var_repeated_int64": []string{
			"10", "11", "900",
		},
		"var_sint64": -555,
		"var_uint64": 4320,
	}
	body, err := json.Marshal(bodyJson)
	sendHeaders := make(http.Header)
	sendHeaders.Set("content-type", "application/json")
	status, content, headers := fetchUrl(
		"default", "POST", "/_ah/api/test_service/v1/increment_integers",
		body, sendHeaders)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var responseJson interface{}
	err := json.Unmarshal(content, &responseJson)
	expectedResponse := map[string]interface{}{
		"var_int32": 101,
		"var_int64": "1001",
		"var_repeated_int64": []string{
			"11", "12", "901",
		},
		"var_sint64": "-554",
		"var_uint64": "4321",
	}
	if expectedResponse != responseJson {
		t.Fail()
	}
}

// Test that the discovery configuration looks right.
func TestDiscoveryConfig(t *testing.T) {
	status, content, headers := fetchUrl(
		"default", "GET", "/_ah/api/discovery/v1/apis/test_service/v1/rest")
	if 200 != status {
		t.Fail()
	}
	if "application/json; charset=UTF-8" != headers.Get("Content-Type") {
		t.Fail()
	}

	var responseJson interface{}
	err := json.Unmarshal(content, &responseJson)
	if err != nil {
		t.Fail()
	}
	assertRegexpMatches(responseJson["baseUrl"],
		`^http://localhost(:\d+)?/_ah/api/test_service/v1/$`)
	assertRegexpMatches(responseJson["rootUrl"],
		`^http://localhost(:\d+)?/_ah/api/$`)
}

// Test that a GET request to a second class in the REST API works.
func TestMulticlassRestGet(t *testing.T) {
	status, content, headers := fetchUrl("default", "GET",
		"/_ah/api/test_service/v1/extrapath/test")
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var responseJson interface{}
	err := json.Unmarshal(content, &responseJson)
	if err != nil {
		t.Fail()
	}
	if map[string]string{"text": "Extra test response"} != responseJson {
		t.Fail()
	}
}

// Test that an RPC request to a second class in the API works.
func TestMulticlassRpc(t *testing.T) {
	body, err := json.Marshal([]string{
		map[string]interface{}{
			"jsonrpc": "2.0",
			"id": "gapiRpc",
			"method": "testservice.extraname.test",
			"params": make(map[string]interface{}),
			"apiVersion": "v1",
		},
	})
	sendHeaders := make(http.Header)
	sendHeaders.Set("content-type", "application-rpc")
	status, content, headers := fetchUrl("default", "POST",
		"/_ah/api/rpc", body, sendHeaders)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var responseJson interface{}
	err := json.Unmarshal(content, &responseJson)
	if []string{
		map[string]interface{}{
			"result": map[string]string{
				"text": "Extra test response",
			},
			"id": "gapiRpc",
		},
	} != responseJson {
		t.Fail()
	}
}

// Test that a GET request to a second similar API works.
func TestSecondApiNoCollision(t *testing.T) {
	status, content, headers := fetchUrl("default", "GET",
		"/_ah/api/second_service/v1/test")
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &response_json)
	expected := map[string]interface{}{"text": "Second response"}
	if expected != responseJson {
		t.Fail()
	}
}*/
