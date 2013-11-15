// Test service for regression testing of Cloud Endpoints support.
package endpoint

import (
	"time"
	"github.com/rwl/go-endpoints/endpoints"
	"net/http/httptest"
	"net/http"
	"net/url"
	"fmt"
	"github.com/golang/glog"
	"strconv"
)

var ts *httptest.Server

type VoidMessage struct{}

type TimeMessage struct {
	Milliseconds int `json:"milliseconds"`
	TimeZoneOffset int `json:"time_zone_offset"`
}

type Int64 string

func (n Int64) Int64() (int64, error) {
	return strconv.ParseInt(string(n), 10, 64)
}

type UInt64 string

func (n UInt64) Int64() (uint64, error) {
	i, err := strconv.ParseInt(string(n), 10, 64)
	return uint64(i), err
}

// Simple Endpoints request, for testing.
type TestRequest struct {
	Name   string `json:"name"`
	Number int    `json:"number"`
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
	VarInt32         int32   `json:"var_int32"`
	VarInt64         Int64   `json:"var_int64"`
	VarRepeatedInt64 []Int64 `json:"var_repeated_int64"`
	VarUnsignedInt64 UInt64  `json:"var_uint64"`
}

// Simple Endpoints request/response with a bytes field.
type TestBytes struct {
	BytesValue []byte `json:"bytes_value"`
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
func (s *TestService) EchoDateMessage(_ *http.Request, req *TimeMessage, resp *TimeMessage) error {
	resp.Milliseconds = req.Milliseconds
	resp.TimeZoneOffset = req.TimeZoneOffset
	return nil
}

//@endpoints.method(TestDateTime, TestDateTime, http_method='POST', name='echodtfield', path='echo_dt_field', scopes=[])
func (s *TestService) EchoDatetimeField(_ *http.Request, req *TestDateTime, resp *TestDateTime) error {
	// Make sure we can access the fields of the datetime object.
	glog.Infof("Year %d, Month %d", req.Date.Year(), req.Date.Month())
	resp.Date = req.Date
	return nil
}

//@endpoints.method(TestIntegers, TestIntegers, http_method='POST', scopes=[])
func (s *TestService) IncrementIntegers(_ *http.Request, req *TestIntegers, resp *TestIntegers) error {
	resp.VarInt32 = req.VarInt32 + 1
	val, _ := req.VarInt64.Int64()
	resp.VarInt64 = Int64(fmt.Sprintf("%d", val + 1))
	resp.VarRepeatedInt64 = make([]Int64, len(req.VarRepeatedInt64))
	for i, v := range req.VarRepeatedInt64 {
		val, _ = v.Int64()
		resp.VarRepeatedInt64[i] = Int64(fmt.Sprintf("%d", val + 1))
	}
	uval, _ := req.VarUnsignedInt64.Int64()
	resp.VarUnsignedInt64 = UInt64(fmt.Sprintf("%d", uval + 1))
	return nil
}

//@endpoints.method(TestBytes, TestBytes, scopes=[])
func (s *TestService) EchoBytes(_ *http.Request, req *TestBytes, resp *TestBytes) error {
	glog.Infof("Found bytes: %s", string(req.BytesValue))
	resp.BytesValue = req.BytesValue
	return nil
}

//@endpoints.method(message_types.VoidMessage, message_types.VoidMessage, path='empty_response', http_method='GET', scopes=[])
func (s *TestService) EmptyResponse(_ *http.Request, _ *VoidMessage, _ *VoidMessage) error {
	return nil
}

//@my_api.api_class(resource_name='extraname', path='extrapath')
// Additional test methods in the test API.
/*type ExtraMethods struct{}

//@endpoints.method(message_types.VoidMessage, TestResponse, http_method='GET', name='test', path='test', scopes=[])
func (em *ExtraMethods) Test(_ *http.Request, _ *VoidMessage, resp *TestResponse) error {
	resp.Text = "Extra test response"
	return nil
}*/

//@endpoints.api(name='second_service', version='v1')
// Second test class for Cloud Endpoints.
type SecondService struct{}

//@endpoints.method(message_types.VoidMessage, TestResponse, http_method='GET', name='test_name', path='test', scopes=[])
func (ss *SecondService) SecondTest(_ *http.Request, _ *VoidMessage, resp *TestResponse) error {
	resp.Text = "Second response"
	return nil
}

//func initTestApi(t *testing.T) *httptest.Server {
func init() {
	testService := &TestService{}
	api, err := endpoints.RegisterService(testService,
		"test_service", "v1", "Test API", true)
	//assert.NoError(t, err)
	if err != nil {
		panic(err.Error())
	}

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
	/*extraMethods := &ExtraMethods{}
	api, err = endpoints.RegisterService(extraMethods,
		"extraname", "v1", "Extra methods", false)
	//path = 'extrapath'
	assert.NoError(t, err)

	info = api.MethodByName("Test").Info()
	info.Name, info.HttpMethod, info.Path = "test", "GET", "test"*/

	// Test a second API, same version, same path. Shouldn't collide.
	secondService := &SecondService{}
	api, err = endpoints.RegisterService(secondService,
		"second_service", "v1", "Second service", false)
	//assert.NoError(t, err)
	if err != nil {
		panic(err.Error())
	}

	info = api.MethodByName("SecondTest").Info()
	info.Name, info.HttpMethod, info.Path = "test_name", "GET", "test"

	mux := http.NewServeMux()
	ts = httptest.NewServer(mux)

	//endpoints.HandleHttp()
	endpoints.DefaultServer.HandleHttp(mux)

	u, err := url.Parse(ts.URL)
	//assert.NoError(t, err)
	if err != nil {
		panic(err.Error())
	}
	server := NewEndpointsServer("", u)
	server.HandleHttp(mux)

	//return ts
}
