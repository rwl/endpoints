// Test utilities.

package discovery

import (
	"io/ioutil"
	"net/http"
	"bytes"
	"testing"
	"github.com/stretchr/testify/mock"
	"net/http/httptest"
	"log"
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
func build_request(url, body string, http_headers http.Header) *ApiRequest {
	//	unused_scheme, unused_netloc, path, query, unused_fragment := urlparse.urlsplit(path)
	req, err := http.NewRequest("GET", url,//fmt.Sprintf("http://localhost%s", url),
		ioutil.NopCloser(bytes.NewBufferString(body)))
	if err != nil {
		log.Print(err.Error())
		panic(err.Error())
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
	if expected_status != response.StatusCode {
		t.Fail()
	}

	// Verify that headers match. Order shouldn't matter.
	if len(response.Header) != len(expected_headers) {
		t.Fail()
	}
	for key, value := range response.Header {
		expected_value, ok := expected_headers[key]
		if !ok {
			t.Fail()
		}
		if value[0] != expected_value[0] {
			t.Fail()
		}
	}

	// Convert the body to a string.
	body, _ := ioutil.ReadAll(response.Body)
	if expected_body != string(body) {
		t.Fail()
	}
}

// Test that the headers and body match.
func assert_http_match_recorder(t *testing.T, recorder *httptest.ResponseRecorder, expected_status int,
expected_headers http.Header, expected_body string) {
	if expected_status != recorder.Code {
		t.Fail()
	}

	// Verify that headers match. Order shouldn't matter.
	if len(recorder.Header()) != len(expected_headers) {
		t.Fail()
	}
	for key, value := range recorder.Header() {
		expected_value, ok := expected_headers[key]
		if !ok {
			t.Fail()
		}
		if value[0] != expected_value[0] {
			t.Fail()
		}
	}

	// Convert the body to a string.
	body, _ := ioutil.ReadAll(recorder.Body)
	if expected_body != string(body) {
		t.Fail()
	}
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

func (ed *MockEndpointsDispatcher) call_spi(w http.ResponseWriter, orig_request *ApiRequest) (string, error) {
	args := ed.Mock.Called(w, orig_request)
	return args.String(0), args.Error(1)
}
