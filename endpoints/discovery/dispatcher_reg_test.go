// Regression tests for Endpoints server.

package endpoints

import (
	"encoding/json"
	"testing"
	"net/http"
	"strings"
)

// Test that a GET request to a REST API works.
func test_rest_get(t *testing.T) {
	status, content, headers := fetch_url("default", "GET",
		"/_ah/api/test_service/v1/test")
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers["Content-Type"] {
		t.Fail()
	}

	var response_json interface{}
	err := json.Unmarshal([]byte(content), &response_json)
	if err != nil {
		t.Fail()
	}
	if map[string]string{"text": "Test response"} != response_json {
		t.Fail()
	}
}

// Test that a POST request to a REST API works.
func test_rest_post(t *testing.T) {
	body, err := json.Marshal(map[string]interface{}{
		"name": "MyName",
		"number": 23,
	})
	send_headers := new(http.Header)
	send_header.Add("content-type", "application/json")
	status, content, headers := fetch_url("default", "POST",
		"/_ah/api/test_service/v1/t2path",
		string(body), send_headers)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var response_json interface{}
	err = json.Unmarshal(content, &response_json)
	if map[string]string{"text": "MyName 23"} != response_json {
		t.Fail()
	}
}

// Test that CORS headers are handled properly.
func test_cors(t *testing.T) {
	send_headers := new(http.Header)
	send_headers.Set("Origin", "test.com")
	send_headers.Set("Access-control-request-method", "GET")
	send_headers.Set("Access-Control-Request-Headers", "Date,Expires")
	status, _, headers := fetch_url("default", "GET",
		"/_ah/api/test_service/v1/test",
		/*headers=*/send_headers)
	if 200 != status {
		t.Fail()
	}
	if headers.Get(_CORS_HEADER_ALLOW_ORIGIN) != "test.com" {
		t.Fail()
	}
	assertIn(t, "GET", strings.Split(headers.Get(_CORS_HEADER_ALLOW_METHODS), ","))
	if headers.Get(_CORS_HEADER_ALLOW_HEADERS) != "Date,Expires" {
		t.Fail()
	}
}

// Test that an RPC request works.
func test_rpc(t *testing.T) {
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
	send_headers := new(http.Header)
	send_headers.Set("content-type", "application-rpc")
	status, content, headers := fetch_url("default", "POST",
		"/_ah/api/rpc",
		body, send_headers)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var response_json interface{}
	err := json.Unmarshal(content, response_json)
	if []string{
		map[string]interface{}{
			"result": map[string]string{
				"text": "MyName 23",
			},
			"id": "gapiRpc",
		},
	} != response_json {
		t.Fail()
	}
}

// Test sending and receiving a datetime.
func test_echo_datetime_message(t *testing.T) {
	body, err := json.Marshal(map[string]interface{}{
		"milliseconds": "5000",
		"time_zone_offset": "60",
	})
	send_headers := new(http.Header)
	send_headers.Set("content-type", "application/json")
	status, content, headers := fetch_url(
		"default", "POST", "/_ah/api/test_service/v1/echo_datetime_message",
		body, send_headers)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var response_json interface{}
	err := json.Unmarshal(content, response_json)
	if map[string]string{
		"milliseconds": "5000",
		"time_zone_offset": "60",
	} != response_json {
		t.Fail()
	}
}

// Test sending and receiving a message that includes a datetime.
func test_echo_datetime_field(t *testing.T) {
	body_json := map[string]string{
		"datetime_value": "2013-03-13T15:29:37.883000+08:00",
	}
	body, err := json.Marshal(body_json)
	send_headers := new(http.Header)
	send_headers.Set("content-type", "application/json")
	status, content, headers := fetch_url(
		"default", "POST", "/_ah/api/test_service/v1/echo_datetime_field",
		body, send_headers)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var response_json interface{}
	err := json.Unmarshal(content, response_json)
	if body_json != response_json {
		t.Fail()
	}
}

// Test sending and receiving integer values.
func test_increment_integers(t *testing.T) {
	body_json := map[string]interface{}{
		"var_int32": 100,
		"var_int64": "1000",
		"var_repeated_int64": []string{
			"10", "11", "900",
		},
		"var_sint64": -555,
		"var_uint64": 4320,
	}
	body, err := json.Marshal(body_json)
	send_headers := new(http.Header)
	send_headers.Set("content-type", "application/json")
	status, content, headers := fetch_url(
		"default", "POST", "/_ah/api/test_service/v1/increment_integers",
		body, send_headers)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var response_json interface{}
	err := json.Unmarshal(content, response_json)
	expected_response := map[string]interface{}{
		"var_int32": 101,
		"var_int64": "1001",
		"var_repeated_int64": []string{
			"11", "12", "901",
		},
		"var_sint64": "-554",
		"var_uint64": "4321",
	}
	if expected_response != response_json {
		t.Fail()
	}
}

// Test that the discovery configuration looks right.
func test_discovery_config(t *testing.T) {
	status, content, headers := fetch_url(
		"default", "GET", "/_ah/api/discovery/v1/apis/test_service/v1/rest")
	if 200 != status {
		t.Fail()
	}
	if "application/json; charset=UTF-8" != headers.Get("Content-Type") {
		t.Fail()
	}

	var response_json interface{}
	err := json.Unmarshal(content, response_json)
	assertRegexpMatches(
		response_json["baseUrl"],
		`^http://localhost(:\d+)?/_ah/api/test_service/v1/$`,
	)
	assertRegexpMatches(
		response_json["rootUrl"],
		`^http://localhost(:\d+)?/_ah/api/$`,
	)
}

// Test that a GET request to a second class in the REST API works.
func test_multiclass_rest_get(t *testing.T) {
	status, content, headers := fetch_url(
		"default", "GET", "/_ah/api/test_service/v1/extrapath/test")
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var response_json interface{}
	err := json.Unmarshal(content, response_json)
	if map[string]string{"text": "Extra test response"} != response_json {
		t.Fail()
	}
}

// Test that an RPC request to a second class in the API works.
func test_multiclass_rpc(t *testing.T) {
	body, err := json.Marshal([]string{
		map[string]interface{}{
			"jsonrpc": "2.0",
			"id": "gapiRpc",
			"method": "testservice.extraname.test",
			"params": make(map[string]interface{}),
			"apiVersion": "v1",
		},
	})
	send_headers := new(http.Header)
	send_headers.Set("content-type", "application-rpc")
	status, content, headers := fetch_url("default", "POST",
		"/_ah/api/rpc",
		body, send_headers)
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var response_json interface{}
	err := json.Unmarshal(content, response_json)
	if []string{
		map[string]interface{}{
			"result": map[string]string{
				"text": "Extra test response",
			},
			"id": "gapiRpc",
		},
	} != response_json {
		t.Fail()
	}
}

// Test that a GET request to a second similar API works.
func test_second_api_no_collision(t *testing.T) {
	status, content, headers := fetch_url("default", "GET",
		"/_ah/api/second_service/v1/test")
	if 200 != status {
		t.Fail()
	}
	if "application/json" != headers.Get("Content-Type") {
		t.Fail()
	}

	var response_json interface{}
	err = json.Unmarshal(content, response_json)
	if map[string]string{"text": "Second response"} != response_json {
		t.Fail()
	}
}
