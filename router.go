
package endpoint

import (
	"regexp"
	"net/http"
)

type Route struct {
	re *regexp.Regexp
	handler func(http.ResponseWriter, *http.Request)
}

type Router struct {
	routes []*Route
}

func NewRouter() *Router {
	return &Router{}
}

func (h *Router) HandleFunc(re string, handler func(http.ResponseWriter, *http.Request)) {
	r := &Route{regexp.MustCompile(re), handler}
	h.routes = append(h.routes, r)
}

func (h *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range h.routes {
		if route.re.MatchString(r.URL.RequestURI()) {
			route.handler(w, r)
			break
		}
	}
}
