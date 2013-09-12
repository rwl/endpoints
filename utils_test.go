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
func buildApiRequest(url, body string, httpHeaders http.Header) *ApiRequest {
	req := buildRequest(url, body, httpHeaders)
	apiRequest, err := newApiRequest(req)
	if err != nil {
		log.Fatal(err.Error())
	}

	return apiRequest
}

func buildRequest(url, body string, httpHeaders http.Header) *http.Request {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:42%s", url),
		ioutil.NopCloser(bytes.NewBufferString(body)))
	if err != nil {
		log.Fatal(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	if httpHeaders != nil {
		for key, value := range httpHeaders {
			req.Header.Set(key, value[0])
		}
	}
	return req
}

// Test that the headers and body match.
func assertHttpMatch(t *testing.T, response *http.Response, expectedStatus int,
	expectedHeaders http.Header, expectedBody string) {
	assert.Equal(t, expectedStatus, response.StatusCode)

	// Verify that headers match. Order shouldn't matter.
	assert.Equal(t, response.Header, expectedHeaders)

	// Convert the body to a string.
	body, _ := ioutil.ReadAll(response.Body)
	assert.Equal(t, expectedBody, string(body))
}

// Test that the headers and body match.
func assertHttpMatchRecorder(t *testing.T, recorder *httptest.ResponseRecorder,
	expectedStatus int, expectedHeaders http.Header, expectedBody string) {
	assert.Equal(t, expectedStatus, recorder.Code)

	// Verify that headers match. Order shouldn't matter.
	assert.Equal(t, recorder.Header(), expectedHeaders)

	// Convert the body to a string.
	assert.Equal(t, expectedBody, recorder.Body.String())
}

type MockEndpointsServer struct {
	mock.Mock
	*EndpointsServer
}

func newMockEndpointsServer() (*MockEndpointsServer) {
	return &MockEndpointsServer{
		EndpointsServer: NewEndpointsServer(),
	}
}

// fixme: mock handle_spi_response without duplicating dispatch
func (ed *MockEndpointsServer) dispatch(w http.ResponseWriter, ar *ApiRequest) string {
	apiConfigResponse, _ := ed.getApiConfigs()
	ed.handleApiConfigResponse(apiConfigResponse)
	body, _ := ed.callSpi(w, ar)
	return body
}

// fixme: mock handle_spi_response without duplicating call_spi
func (ed *MockEndpointsServer) callSpi(w http.ResponseWriter, origRequest *ApiRequest) (string, error) {
	var methodConfig *endpoints.ApiMethod
	var params map[string]string
	if origRequest.IsRpc() {
		methodConfig = ed.lookupRpcMethod(origRequest)
		params = nil
	} else {
		methodConfig, params = ed.lookupRestMethod(origRequest)
	}

	spiRequest, _ := ed.transformRequest(origRequest, params, methodConfig)

	discovery := NewDiscoveryService(ed.configManager)
	discoveryResponse, ok := discovery.handleDiscoveryRequest(
		spiRequest.URL.Path, spiRequest, w)
	if ok {
		return discoveryResponse, nil
	}

	url := fmt.Sprintf(SpiRootFormat, spiRequest.URL.Path)
	req, _ := http.NewRequest("POST", url, spiRequest.Body)
	req.Header.Add("Content-Type", "application/json")
	req.RemoteAddr = spiRequest.RemoteAddr
	client := &http.Client{}
	resp, _ := client.Do(req)
	return ed.handleSpiResponse(origRequest, spiRequest, resp, w)
}

func (ed *MockEndpointsServer) handleSpiResponse(origRequest, spiRequest *ApiRequest,
		response *http.Response, w http.ResponseWriter) (string, error) {
	args := ed.Mock.Called(origRequest, spiRequest, response, w)
	return args.String(0), args.Error(1)
}

type MockEndpointsServerSpi struct {
	mock.Mock
	*EndpointsServer
}

func newMockEndpointsServerSpi() (*MockEndpointsServerSpi) {
	return &MockEndpointsServerSpi{
		EndpointsServer: NewEndpointsServer(),
	}
}

// fixme: mock call_spi without duplicating dispatch
func (ed *MockEndpointsServerSPI) dispatch(w http.ResponseWriter, ar *ApiRequest) string {
	// Get API configuration first.  We need this so we know how to
	// call the back end.
	apiConfigResponse, err := ed.getApiConfigs()
	if err != nil {
		return ed.failRequest(w, ar.Request, "BackendService.getApiConfigs Error: "+err.Error())
	}
	err = ed.handleGetApiConfigResponse(apiConfigResponse)
	if err != nil {
		return ed.failRequest(w, ar.Request, "BackendService.getApiConfigs Error: "+err.Error())
	}

	// Call the service.
	body, err := ed.callSpi(w, ar)
	if err != nil {
		reqErr, ok := err.(RequestError)
		if ok {
			return ed.handleRequestError(w, ar, reqErr)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return body
		}
	}
	return body
}

func (ed *MockEndpointsServerSPI) callSpi(w http.ResponseWriter, origRequest *ApiRequest) (string, error) {
	args := ed.Mock.Called(w, origRequest)
	return args.String(0), args.Error(1)
}
