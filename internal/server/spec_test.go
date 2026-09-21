package server

import (
	"reflect"
	"sort"
	"testing"
)

// recordWith runs registerAll against a recorder and reports the patterns
// it collected, one "METHOD /path" per route, sorted.
func recordWith(deps Deps) []string {
	var routes []RouteSpec
	registerAll(specRecorder{routes: &routes}, deps)
	out := make([]string, 0, len(routes))
	for _, r := range routes {
		out = append(out, r.Method+" "+r.Pattern)
	}
	sort.Strings(out)
	return out
}

// filledDeps returns a Deps whose every field carries a non-zero value:
// pointers point at zero values, funcs are no-ops, maps are allocated,
// strings and bools are set. Registration must not read any of it — the
// register* helpers only close over deps — so the route set it produces
// has to match the one Routes() records from a zero Deps.
func filledDeps(t *testing.T) Deps {
	t.Helper()
	var deps Deps
	v := reflect.ValueOf(&deps).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		switch f.Kind() {
		case reflect.Pointer:
			f.Set(reflect.New(f.Type().Elem()))
		case reflect.Map:
			f.Set(reflect.MakeMap(f.Type()))
		case reflect.Slice:
			f.Set(reflect.MakeSlice(f.Type(), 0, 0))
		case reflect.Func:
			f.Set(reflect.MakeFunc(f.Type(), func([]reflect.Value) []reflect.Value {
				out := make([]reflect.Value, f.Type().NumOut())
				for j := range out {
					out[j] = reflect.Zero(f.Type().Out(j))
				}
				return out
			}))
		case reflect.String:
			f.SetString("x")
		case reflect.Bool:
			f.SetBool(true)
		case reflect.Interface:
			// Left nil: there is no generic non-nil value to invent, and a
			// register* helper that branched on one would still be caught
			// by the pointer and func fields it branches on in practice.
		}
	}
	return deps
}

// Routes() is what cmd/picode-openapi turns into the published spec, and
// it records registerAll against a zero Deps. A helper that registers
// conditionally therefore writes a route the binary serves out of the
// public API reference without failing anything: registerAuthRoutes did
// exactly that, and every /api/auth route plus /pair was missing from
// docs-site/public/api/openapi.json while the daemon served all nine.
//
// docs-check.mjs only proves the committed JSON matches this generator,
// never that the generator matches the mux — so the invariant lives here:
// registration may not depend on what Deps carries.
func TestRoutesCoverEveryRegisteredPattern(t *testing.T) {
	zero := recordWith(Deps{})
	filled := recordWith(filledDeps(t))

	if !reflect.DeepEqual(zero, filled) {
		missing, extra := diffPatterns(zero, filled)
		for _, p := range missing {
			t.Errorf("registered only with a populated Deps, so the OpenAPI spec omits it: %s", p)
		}
		for _, p := range extra {
			t.Errorf("registered only with a zero Deps, so the OpenAPI spec invents it: %s", p)
		}
		t.Log("make registration unconditional and answer the missing dependency in the handler " +
			"(the house rule Deps documents: \"nil-safe = 503 on the routes\")")
	}
}

// The auth surface is the one an API consumer needs first, and it is the
// one that was missing. Name it explicitly so a regression reads as what
// it is rather than as a diff of two lists.
func TestRoutesIncludeTheAuthSurface(t *testing.T) {
	got := map[string]bool{}
	for _, p := range recordWith(Deps{}) {
		got[p] = true
	}
	for _, want := range []string{
		"GET /api/auth/session",
		"GET /api/auth/sessions",
		"DELETE /api/auth/sessions/{id}",
		"POST /api/auth/pairings",
		"POST /api/auth/logout",
		"PUT /api/auth/mode",
		"POST /api/auth/token/rotate",
	} {
		if !got[want] {
			t.Errorf("Routes() omits %s — the published spec will too", want)
		}
	}
}

func diffPatterns(a, b []string) (onlyB, onlyA []string) {
	inA := map[string]bool{}
	for _, s := range a {
		inA[s] = true
	}
	inB := map[string]bool{}
	for _, s := range b {
		inB[s] = true
	}
	for _, s := range b {
		if !inA[s] {
			onlyB = append(onlyB, s)
		}
	}
	for _, s := range a {
		if !inB[s] {
			onlyA = append(onlyA, s)
		}
	}
	return onlyB, onlyA
}
