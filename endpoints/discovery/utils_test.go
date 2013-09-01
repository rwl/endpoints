// Test utilities.

package discovery

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func set_up() (*EndpointsDispatcher, *MockDispatcher) {
	config_manager := NewApiConfigManager()
	mock_dispatcher := new(MockDispatcher)
	server := NewEndpointsDispatcherConfig(mock_dispatcher, config_manager)
	return server, mock_dispatcher
}

// Build an ApiRequest for the given path and body.
//
// Args:
//   url: A string containing the URL for the proposed request.
//   body: A string containing the body of the proposed request.
//   http_headers: A list of (header, value) headers to add to the request.
//
// Returns:
//   An ApiRequest object built based on the incoming parameters.
func build_request(url, body string, http_headers http.Header) *ApiRequest {
	//unused_scheme, unused_netloc, path, query, unused_fragment := urlparse.urlsplit(path)
	req, err := http.NewRequest("GET", url, //fmt.Sprintf("http://localhost%s", url),
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

	api_request, err := newApiRequest(req)
	if err != nil {
		log.Fatal(err.Error())
	}

	return api_request
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
	body, _ := ioutil.ReadAll(recorder.Body)
//	fmt.Printf("EXPECTED:\n%s", expected_body)
//	fmt.Printf("ACTUAL:\n%s", body)
	assert.Equal(t, expected_body, string(body))
}

type MockDispatcher struct {
	mock.Mock
}

func (md *MockDispatcher) Do(request *http.Request) (*http.Response, error) {
	args := md.Mock.Called(request)
	return args.Get(0).(*http.Response), args.Error(1)
}

type MockEndpointsDispatcher struct {
	mock.Mock
	*EndpointsDispatcher
}

func newMockEndpointsDispatcher() (*MockEndpointsDispatcher, *MockDispatcher) {
	server, dispatcher := set_up()
	return &MockEndpointsDispatcher{
		EndpointsDispatcher: server,
	}, dispatcher
}

func (ed *MockEndpointsDispatcher) handle_spi_response(orig_request, spi_request *ApiRequest, response *http.Response, w http.ResponseWriter) (string, error) {
	args := ed.Mock.Called(orig_request, spi_request, response, w)
	return args.String(0), args.Error(1)
}

type MockEndpointsDispatcherSPI struct {
	mock.Mock
	*EndpointsDispatcher
}

func newMockEndpointsDispatcherSPI() (*MockEndpointsDispatcherSPI, *MockDispatcher) {
	server, dispatcher := set_up()
	return &MockEndpointsDispatcherSPI{
		EndpointsDispatcher: server,
	}, dispatcher
}

func (ed *MockEndpointsDispatcherSPI) dispatch(w http.ResponseWriter, ar *ApiRequest) string {
	// Check if this matches any of our special handlers.
	dispatched_response, err := ed.dispatch_non_api_requests(w, ar.Request)
	if err == nil {
		return dispatched_response
	}

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

func (ed *MockEndpointsDispatcherSPI) call_spi(w http.ResponseWriter, orig_request *ApiRequest) (string, error) {
	args := ed.Mock.Called(w, orig_request)
	return args.String(0), args.Error(1)
}
