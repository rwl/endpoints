
package endpoint

import (
	"net/http"
	"strings"
)

type Route struct {
	pathPrefix string
	handler func(http.ResponseWriter, *http.Request)
}

type Router struct {
	routes []*Route
}

func NewRouter() *Router {
	return &Router{}
}

func (h *Router) HandleFunc(pathPrefix string, handler func(http.ResponseWriter, *http.Request)) {
	r := &Route{pathPrefix, handler}
	h.routes = append(h.routes, r)
}

func (h *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range h.routes {
		if strings.HasPrefix(r.URL.Path, route.pathPrefix) {
			route.handler(w, r)
			break
		}
	}
}
