package server

import (
	"net/http"
	"strings"
)

// Registrar is what route registration needs: the subset of
// *http.ServeMux the register* helpers use. Tests and the OpenAPI
// generator can pass a recorder instead of a live mux.
type Registrar interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	Handle(pattern string, handler http.Handler)
}

// RouteSpec is one registered route: "METHOD /path" as handed to the
// mux (websocket endpoints register methodless patterns and are
// reported as GET — the only verb an upgrade uses).
type RouteSpec struct {
	Method  string
	Pattern string
}

// specRecorder implements Registrar by collecting patterns.
type specRecorder struct{ routes *[]RouteSpec }

func (s specRecorder) HandleFunc(pattern string, _ func(http.ResponseWriter, *http.Request)) {
	s.record(pattern)
}
func (s specRecorder) Handle(pattern string, _ http.Handler) { s.record(pattern) }

func (s specRecorder) record(pattern string) {
	method, path, found := strings.Cut(pattern, " ")
	if !found {
		method = "GET" // websocket upgrade endpoint ("/ws/term")
		path = pattern
	}
	// The catch-all UI route (and any future static mount) is not API.
	if !strings.HasPrefix(path, "/api") && !strings.HasPrefix(path, "/ws") {
		return
	}
	*s.routes = append(*s.routes, RouteSpec{Method: method, Pattern: path})
}

// Routes reports every API route the server registers, in registration
// order. It runs the same registerAll the real binary runs, so the
// OpenAPI spec generated from it cannot describe something the server
// does not serve (and misses nothing it does).
func Routes() []RouteSpec {
	var routes []RouteSpec
	registerAll(specRecorder{routes: &routes}, Deps{})
	return routes
}
