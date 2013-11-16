package endpoints_server


import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSendRedirectResponse(t *testing.T) {
	w := httptest.NewRecorder()
	r := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/"},
	}
	urlStr := "http://www.google.com"
	/*response :=*/ sendRedirectResponse(urlStr, w, r, nil)

	header := http.Header{
		"Location":       []string{"http://www.google.com"},
		"Content-Length": []string{"0"},
	}
	note := "<a href=\"" + htmlReplacer.Replace(urlStr) + "\">" +
		http.StatusText(302) + "</a>.\n\n"

	assertHttpMatchRecorder(t, w, 302, header, note)
}

func TestSendNoContentResponse(t *testing.T) {
	w := httptest.NewRecorder()
	/*response :=*/ sendNoContentResponse(w, nil)
	header := http.Header{
		"Content-Length": []string{"0"},
	}
	assertHttpMatchRecorder(t, w, 204, header, "")
}

var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&#34;",
	"'", "&#39;",
)
