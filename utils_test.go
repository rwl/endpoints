// Test utilities.

package endpoint

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/rwl/go-endpoints/endpoints"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"fmt"
)

// Build an ApiRequest for the given path and body.
//
// Args:
//   url: A string containing the URL for the proposed request.
//   body: A string containing the body of the proposed request.
//   http_headers: A list of (header, value) headers to add to the request.
//
// Returns:
//   An ApiRequest object built based on the incoming parameters.
func build_api_request(url, body string, http_headers http.Header) *ApiRequest {
	req := build_request(url, body, http_headers)
	api_request, err := newApiRequest(req)
	if err != nil {
		log.Fatal(err.Error())
	}

	return api_request
}

func build_request(url, body string, http_headers http.Header) *http.Request {
	//unused_scheme, unused_netloc, path, query, unused_fragment := urlparse.urlsplit(path)
	req, err := http.NewRequest("GET", /*url, */fmt.Sprintf("http://localhost:42%s", url),
		ioutil.NopCloser(bytes.NewBufferString(body)))
	if err != nil {
		log.Fatal(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	if http_headers != nil {
		for key, value := range http_headers {
			req.Header.Set(key, value[0])
		}
	}
	return req
}

// Test that the headers and body match.
func assert_http_match(t *testing.T, response *http.Response, expected_status int,
	expected_headers http.Header, expected_body string) {
	assert.Equal(t, expected_status, response.StatusCode)

	// Verify that headers match. Order shouldn't matter.
	/*assert.Equal(t, len(response.Header), len(expected_headers))
	for key, value := range response.Header {
		expected_value, ok := expected_headers[key]
		assert.True(t, ok)
		assert.Equal(t, value[0], expected_value[0])
	}*/
	assert.Equal(t, response.Header, expected_headers)

	// Convert the body to a string.
	body, _ := ioutil.ReadAll(response.Body)
	assert.Equal(t, expected_body, string(body))
}

// Test that the headers and body match.
func assert_http_match_recorder(t *testing.T, recorder *httptest.ResponseRecorder, expected_status int,
	expected_headers http.Header, expected_body string) {
	assert.Equal(t, expected_status, recorder.Code)

	// Verify that headers match. Order shouldn't matter.
	/*assert.Equal(t, len(recorder.Header()), len(expected_headers))
	for key, value := range recorder.Header() {
		expected_value, ok := expected_headers[key]
		assert.True(t, ok)
		assert.Equal(t, value[0], expected_value[0])
	}*/
	assert.Equal(t, recorder.Header(), expected_headers)

	// Convert the body to a string.
	//body := recorder.Body.Bytes()
//	fmt.Printf("EXPECTED:\n%s", expected_body)
//	fmt.Printf("ACTUAL:\n%s", recorder.Body.String())
	assert.Equal(t, expected_body, recorder.Body.String())
}

type MockEndpointsDispatcher struct {
	mock.Mock
	*EndpointsDispatcher
}

func newMockEndpointsDispatcher() (*MockEndpointsDispatcher) {
	return &MockEndpointsDispatcher{
		EndpointsDispatcher: NewEndpointsDispatcher(),
	}
}

// fixme: mock handle_spi_response without duplicating dispatch
func (ed *MockEndpointsDispatcher) dispatch(w http.ResponseWriter, ar *ApiRequest) string {
	api_config_response, _ := ed.get_api_configs()
	ed.handle_get_api_configs_response(api_config_response)

	body, _ := ed.call_spi(w, ar)
	return body
}

// fixme: mock handle_spi_response without duplicating call_spi
func (ed *MockEndpointsDispatcher) call_spi(w http.ResponseWriter, orig_request *ApiRequest) (string, error) {
	var method_config *endpoints.ApiMethod
	var params map[string]string
	if orig_request.is_rpc() {
		method_config = ed.lookup_rpc_method(orig_request)
		params = nil
	} else {
		method_config, params = ed.lookup_rest_method(orig_request)
	}

	spi_request, _ := ed.transform_request(orig_request, params, method_config)

	discovery := NewDiscoveryService(ed.config_manager)
	discovery_response, ok := discovery.handle_discovery_request(spi_request.URL.Path,
		spi_request, w)
	if ok {
		return discovery_response, nil
	}

	url := fmt.Sprintf(_SPI_ROOT_FORMAT, spi_request.URL.Path)
	req, _ := http.NewRequest("POST", url, spi_request.Body)
	req.Header.Add("Content-Type", "application/json")
	req.RemoteAddr = spi_request.RemoteAddr
	client := &http.Client{}
	resp, _ := client.Do(req)
	return ed.handle_spi_response(orig_request, spi_request, resp, w)
}

func (ed *MockEndpointsDispatcher) handle_spi_response(orig_request, spi_request *ApiRequest, response *http.Response, w http.ResponseWriter) (string, error) {
	args := ed.Mock.Called(orig_request, spi_request, response, w)
	return args.String(0), args.Error(1)
}

type MockEndpointsDispatcherSPI struct {
	mock.Mock
	*EndpointsDispatcher
}

func newMockEndpointsDispatcherSPI() (*MockEndpointsDispatcherSPI) {
	return &MockEndpointsDispatcherSPI{
		EndpointsDispatcher: NewEndpointsDispatcher(),
	}
}

// fixme: mock call_spi without duplicating dispatch
func (ed *MockEndpointsDispatcherSPI) dispatch(w http.ResponseWriter, ar *ApiRequest) string {
	// Check if this matches any of our special handlers.
	/*dispatched_response, err := ed.dispatch_non_api_requests(w, ar)
	if err == nil {
		return dispatched_response
	}*/

	// Get API configuration first.  We need this so we know how to
	// call the back end.
	api_config_response, err := ed.get_api_configs()
	if err != nil {
		return ed.fail_request(w, ar.Request, "BackendService.getApiConfigs Error: "+err.Error())
	}
	err = ed.handle_get_api_configs_response(api_config_response)
	if err != nil {
		return ed.fail_request(w, ar.Request, "BackendService.getApiConfigs Error: "+err.Error())
	}

	// Call the service.
	body, err := ed.call_spi(w, ar)
	if err != nil {
		req_err, ok := err.(RequestError)
		if ok {
			return ed.handle_request_error(w, ar, req_err)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return body
		}
	}
	return body
}
/*{
	api_config_response, _ := ed.get_api_configs()
	ed.handle_get_api_configs_response(api_config_response)

	body, _ := ed.call_spi(w, ar)
	return body
}*/

func (ed *MockEndpointsDispatcherSPI) call_spi(w http.ResponseWriter, orig_request *ApiRequest) (string, error) {
	args := ed.Mock.Called(w, orig_request)
	return args.String(0), args.Error(1)
}
