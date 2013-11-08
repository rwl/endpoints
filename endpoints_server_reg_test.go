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
	"encoding/json"
	"testing"
	"net/http"
	"strings"
	"time"
	"github.com/rwl/go-endpoints/endpoints"
	"fmt"
	"log"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"path"
	"io/ioutil"
	"bytes"
	"encoding/base64"
)

// Simple Endpoints request, for testing.
type TestRequest struct {
	Name string `json:"name"`
	Number int `json:"number"`
}

// Simple Endpoints response with a text field.
type TestResponse struct {
	Text string `json:"text"`
}

// Simple Endpoints request/response with a time.
type TestDateTime struct {
	Date time.Time `json:"date"`
}

// Simple Endpoints request/response with a few integer types.
type TestIntegers struct {
	VarInt32 int32 `json:"var_int32"`
	VarInt64 int64 `json:"var_int64"`
	VarRepeatedInt64 []int64 `json:"var_repeated_int64"`
	VarUnsignedInt64 uint64 `json:"var_uint64"`
}

// Simple Endpoints request/response with a bytes field.
type TestBytes struct {
	BytesValue []byte `json:"bytes_value"`
}

func initTestApi(t *testing.T) *httptest.Server {
	testService := &TestService{}
	api, err := endpoints.RegisterService(testService,
		"test_service", "v1", "Test API", true)
	assert.NoError(t, err)

	info := api.MethodByName("Test").Info()
	info.HttpMethod, info.Desc = "GET", "Responds with a text value."

	info = api.MethodByName("EmptyTest").Info()
	info.HttpMethod = "GET"

	info = api.MethodByName("Environ").Info()
	info.Name, info.HttpMethod, info.Path = "t2name", "POST", "t2path"

	info = api.MethodByName("EchoDateMessage").Info()
	info.Name, info.HttpMethod = "echodtmsg", "POST"

	info = api.MethodByName("EmptyResponse").Info()
	info.HttpMethod, info.Path = "GET", "empty_response"


	// Some extra test methods in the test API.
	extraMethods := &ExtraMethods{}
	api, err = endpoints.RegisterService(extraMethods,
		"extraname", "v1", "Extra methods", false)
	//path = 'extrapath'
	assert.NoError(t, err)

	info = api.MethodByName("Test").Info()
	info.Name, info.HttpMethod, info.Path = "test", "GET", "test"


	// Test a second API, same version, same path. Shouldn't collide.
	secondService := &SecondService{}
	api, err = endpoints.RegisterService(secondService,
		"second_service", "v1", "Second service", false)
	assert.NoError(t, err)

	info = api.MethodByName("SecondTest").Info()
	info.Name, info.HttpMethod, info.Path = "test_name", "GET", "test"


	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)

	//endpoints.HandleHttp()
	endpoints.DefaultServer.HandleHttp(mux)

	server := NewEndpointsServer("", ts.URL)
	server.HandleHttp(mux)

	return ts
}

// Test RPC service for Cloud Endpoints.
type TestService struct {
}

//@endpoints.method(message_types.VoidMessage, TestResponse, http_method='GET', scopes=[])
func (s *TestService) Test(_ *http.Request, _ *VoidMessage, resp *TestResponse) error {
	resp.Text = "Test response"
	return nil
}

//@endpoints.method(message_types.VoidMessage, TestResponse, http_method='GET', scopes=[])
func (s *TestService) EmptyTest(_ *http.Request, _ *VoidMessage, _ *TestResponse) error {
	return nil
}

//@endpoints.method(TestRequest, TestResponse, http_method='POST', name='t2name', path='t2path', scopes=[])
func (s *TestService) Environ(_ *http.Request, req *TestRequest, resp *TestResponse) error {
	resp.Text = fmt.Sprintf("%s %d", req.Name, req.Number)
	return nil
}

//@endpoints.method(message_types.DateTimeMessage, message_types.DateTimeMessage, http_method='POST', name='echodtmsg', scopes=[])
func (s *TestService) EchoDateMessage(_ *http.Request, req *time.Time, resp *time.Time) error {
	resp.Date = req.Date
	return nil
}

//@endpoints.method(TestDateTime, TestDateTime, http_method='POST', name='echodtfield', path='echo_dt_field', scopes=[])
func (s *TestService) EchoDatetimeField(_ *http.Request, req *TestDateTime, _ *TestDateTime) error {
	// Make sure we can access the fields of the datetime object.
	log.Printf("Year %d, Month %d", req.Date.Year(), req.Date.Month())
	return nil
}

//@endpoints.method(TestIntegers, TestIntegers, http_method='POST', scopes=[])
func (s *TestService) IncrementIntegers(_ *http.Request, req *TestIntegers, resp *TestIntegers) error {
	resp.VarInt32 = req.VarInt32 + 1
	resp.VarInt64 = req.VarInt64 + 1
	resp.VarRepeatedInt64 = make([]int64, len(req.VarRepeatedInt64))
	for i, v := range req.VarRepeatedInt64 {
		resp.VarRepeatedInt64[i] = v + 1
	}
	resp.VarUInt64 = req.VarUInt64 + 1
	return nil
}

//@endpoints.method(TestBytes, TestBytes, scopes=[])
func (s *TestService) EchoBytes(_ *http.Request, req *TestBytes, _ *TestBytes) error {
	log.Printf("Found bytes: %s", string(req.BytesValue))
	return nil
}

//@endpoints.method(message_types.VoidMessage, message_types.VoidMessage, path='empty_response', http_method='GET', scopes=[])
func (s *TestService) EmptyResponse(_ *http.Request, _ *VoidMessage, _ *VoidMessage) error {
	return nil
}

//@my_api.api_class(resource_name='extraname', path='extrapath')
// Additional test methods in the test API.
type ExtraMethods struct {}

//@endpoints.method(message_types.VoidMessage, TestResponse, http_method='GET', name='test', path='test', scopes=[])
func (em *ExtraMethods) Test(_ *http.Request, _ *VoidMessage, resp *TestResponse) error {
	resp.Text = "Extra test response"
	return nil
}

//@endpoints.api(name='second_service', version='v1')
// Second test class for Cloud Endpoints.
type SecondService struct {}

//@endpoints.method(message_types.VoidMessage, TestResponse, http_method='GET', name='test_name', path='test', scopes=[])
func (ss *SecondService) SecondTest(_ *http.Request, _ *VoidMessage, resp *TestResponse) error {
	resp.Text = "Second response"
	return nil
}


// Test that a GET request to a REST API works.
func TestRestGet(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

	resp, err := http.Get(path.Join(ts.URL.String(),
		"/_ah/api/test_service/v1/test"))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(body, &responseJson)
	assert.NoError(t, err)

	/*text, ok := responseJson["text"]
	assert.True(t, ok)
	textStr, ok := text.(string)
	assert.True(t, ok)
	assert.Equal(t, textStr, "Test response")*/

	expected := map[string]interface{} {"text": "Test response"}
	assert.Equal(t, expected, responseJson)
}

// Test that a POST request to a REST API works.
func TestRestPost(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

	body, err := json.Marshal(map[string]interface{}{
		"name": "MyName",
		"number": 23,
	})
	assert.NoError(t, err)

	resp, err := http.Post(path.Join(ts.URL, "/_ah/api/test_service/v1/t2path"),
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

	/*text, ok := responseJson["text"]
	assert.True(t, ok)
	textStr, ok := text.(string)
	assert.True(t, ok)
	assert.Equal(t, textStr, "MyName 23")*/

	expected := map[string]interface{} {"text": "MyName 23"}
	assert.Equal(t, expected, responseJson)
}

// Test that CORS headers are handled properly.
func TestCors(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

	client := &http.Client{}

	req, err := http.NewRequest("GET",
		path.Join(ts.URL, "/_ah/api/test_service/v1/test"), nil)

	req.Header.Set("Origin", "test.com")
	req.Header.Set("Access-control-request-method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Date,Expires")

	resp, err := client.Do(req)

	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get(corsHeaderAllowOrigin), "test.com")

	allowed := strings.Split(resp.Header.Get(corsAllowedMethods), ",")
	for _, header := range allowed {
		if header == "GET" {
			goto P
		}
	}
	t.Fail()
P:
	assert.Equal(resp.Header.Get(corsHeaderAllowHeaders), "Date,Expires")
}

// Test that an RPC request works.
func TestRpc(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

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
	assert.NoError(t, err)

	resp, err := http.Post(path.Join(ts.URL, "/_ah/api/rpc"),
		"application-rpc", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

	/*jsonArray, ok := responseJson.([]map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, len(jsonArray), 1)

	result, ok := jsonArray[0]["result"]
	assert.True(t, ok)
	resultMap, ok := result.(map[string]interface{})
	assert.True(t, ok)
	text, ok := resultMap["text"]
	assert.True(t, ok)
	textStr, ok := text.(string)
	assert.True(t, ok)
	assert.Equals(t, textStr, "MyName 23")

	id, ok := jsonArray[0]["id"]
	assert.True(t, ok)
	idStr, ok := id.(string)
	assert.True(t, ok)
	assert.Equal(t, idStr, "gapiRpc")*/

	assert.Equal(t, []map[string]interface{} {
		map[string]interface{} {
			"result": map[string]interface{} {
				"text": "MyName 23",
			},
			"id": "gapiRpc",
		},
	}, responseJson)
}

// Test sending and receiving a datetime.
func TestEchoDatetimeMessage(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

	body, err := json.Marshal(map[string]interface{}{
		"milliseconds": "5000",
		"time_zone_offset": "60",
	})
	assert.NoError(t, err)

	resp, err := http.Post(path.Join(ts.URL, "/_ah/api/test_service/v1/echo_datetime_message"),
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err = json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

	expected := map[string]interface{} {
		"milliseconds": "5000",
		"time_zone_offset": "60",
	}
	assert.Equal(t, expected, responseJson)
}

// Test sending and receiving a message that includes a time.
func TestEchoDatetimeField(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

	bodyJson := map[string]interface{} {
		"date": "2013-03-13T15:29:37.883000+08:00",
	}
	body, err := json.Marshal(bodyJson)
	assert.NoError(t, err)

	resp, err := http.Post(path.Join(ts.URL, "/_ah/api/test_service/v1/echo_datetime_field"),
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err := json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)
	assert.Equal(t, bodyJson, responseJson)
}

// Test sending and receiving integer values.
func TestIncrementIntegers(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

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
	assert.NoError(t, err)

	resp, err := http.Post(path.Join(ts.URL, "/_ah/api/test_service/v1/increment_integers"),
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err := json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

	expectedResponse := map[string]interface{}{
		"var_int32": 101,
		"var_int64": "1001",
		"var_repeated_int64": []interface{}{
			"11", "12", "901",
		},
		"var_sint64": "-554",
		"var_uint64": "4321",
	}
	assert.Equal(t, expectedResponse, responseJson)
}

// Test sending and receiving a BytesField parameter.
func TestEchoBytes(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

    value := []byte("This is a test of a message encoded as a BytesField.01234\000\001")
    bytesValue := base64.URLEncoding.EncodeToString(value)
    bodyJson := map[string]interface{} {"bytes_value": bytesValue}
    body, err := json.Marshal(bodyJson)
	assert.NoError(t, err)

	resp, err := http.Post(path.Join(ts.URL, "/_ah/api/test_service/v1/echo_bytes"),
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err := json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

    assert.Equal(t, responseJson, bodyJson)
	dec, err := base64.URLEncoding.DecodeString(bytesValue)
	assert.NoError(t, err)
    assert.Equal(t, value, dec)
}

// Test that an empty response that should have an object returns 200.
func TestEmptyTest(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

	resp, err := http.Get(path.Join(ts.URL.String(),
		"/_ah/api/test_service/v1/empty_test"))

    assert.Equal(t, 200, resp.StatusCode)
    assert.Equal(t, "2", resp.Header.Get("Content-Length"))

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

    assert.Equal(t, "{}", string(content))
}

// An empty response that should be empty should return 204.
func TestEmptyResponse(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

	resp, err := http.Get(path.Join(ts.URL.String(),
		"/_ah/api/test_service/v1/empty_response"))

    assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, "0", resp.Header.Get("Content-Length"))

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

    assert.Equal(t, "", string(content))
}

// Test that the discovery configuration looks right.
func TestDiscoveryConfig(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

	resp, err := http.Get(path.Join(ts.URL.String(),
		"/_ah/api/discovery/v1/apis/test_service/v1/rest"))

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/json; charset=UTF-8",
		resp.Header.Get("Content-Type"))

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err := json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)

	assertRegexpMatches(responseJson["baseUrl"],
		`^http://localhost(:\d+)?/_ah/api/test_service/v1/$`)
	assertRegexpMatches(responseJson["rootUrl"],
		`^http://localhost(:\d+)?/_ah/api/$`)
}

// Test that a GET request to a second class in the REST API works.
func TestMulticlassRestGet(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

	resp, err := http.Get(path.Join(ts.URL.String(),
		"/_ah/api/test_service/v1/extrapath/test"))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err := json.Unmarshal(content, &responseJson)
	assert.NoError(t, err)
	expected := map[string]string{"text": "Extra test response"}
	assert.Equal(t, expected, responseJson)
}

// Test that an RPC request to a second class in the API works.
func TestMulticlassRpc(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

	body, err := json.Marshal([]map[string]interface{} {
		map[string]interface{} {
			"jsonrpc": "2.0",
			"id": "gapiRpc",
			"method": "testservice.extraname.test",
			"params": make(map[string]interface{}),
			"apiVersion": "v1",
		},
	})
	assert.NoError(t, err)

	resp, err := http.Post(path.Join(ts.URL, "/_ah/api/rpc"),
		"application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

	content, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseJson map[string]interface{}
	err := json.Unmarshal(content, &responseJson)
	expected := []map[string]interface{} {
		map[string]interface{} {
			"result": map[string]interface{} {
				"text": "Extra test response",
			},
			"id": "gapiRpc",
		},
	}
	assert.Equal(t, expected, responseJson)
}

// Test that a GET request to a second similar API works.
func TestSecondApiNoCollision(t *testing.T) {
	ts := initTestApi(t)
	defer ts.Close()

	resp, err := http.Get(path.Join(ts.URL.String(),
		"/_ah/api/second_service/v1/test"))

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
