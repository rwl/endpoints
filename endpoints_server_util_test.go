
package endpoint

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"net/url"
)

func TestSendRedirectResponse(t *testing.T) {
	w := httptest.NewRecorder()
	r := &http.Request{
		Method: "GET",
		URL: &url.URL{Path: "/"},
	}
	response := sendRedirectResponse("http://www.google.com", w, r, nil)

	header := http.Header{
		"Location": []string{"http://www.google.com"},
		"Content-Length": []string{"0"},
	}
	assertHttpMatchRecorder(t, w, 302, header, "")
}

func TestSendNoContentResponse(t *testing.T) {
	w := httptest.NewRecorder()
	response := sendNoContentResponse(w, nil)
	header := http.Header{
		"Content-Length": []string{"0"},
	}
	assertHttpMatchRecorder(t, w, 204, header, "")
}
