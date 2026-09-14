package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// loadModels posts to the endpoint route and returns status + decoded body.
func loadModels(t *testing.T, req func(string, string, string) (int, string), body string) (int, map[string]any) {
	t.Helper()
	status, raw := req(http.MethodPost, "/api/providers/custom/models", body)
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("body is not JSON: %s", raw)
	}
	return status, out
}

// The happy path, end to end: the form's key reaches the endpoint, the ids
// come back, and the key itself never does.
func TestCustomProviderModelsListsAndHidesKey(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	var sawAuth string
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"data":[{"id":"deepseek-v4.1-flash","context_length":1048576,"max_completion_tokens":384000},{"id":"glm-4.6"}]}`))
	}))
	defer gw.Close()

	status, body := loadModels(t, req, `{"baseUrl":"`+gw.URL+`","api":"openai-completions","key":"sk-live-secret"}`)
	if status != http.StatusOK {
		t.Fatalf("status %d: %v", status, body)
	}
	if sawAuth != "Bearer sk-live-secret" {
		t.Fatalf("the endpoint saw %q", sawAuth)
	}
	if got := body["count"]; got != float64(2) {
		t.Fatalf("count = %v", got)
	}
	if !strings.Contains(body["url"].(string), "/models") {
		t.Fatalf("url = %v", body["url"])
	}
	models, _ := body["models"].([]any)
	first, _ := models[0].(map[string]any)
	if first["id"] != "deepseek-v4.1-flash" || first["contextWindow"] != float64(1048576) || first["maxTokens"] != float64(384000) {
		t.Fatalf("model = %v", first)
	}
	if strings.Contains(strings.Join([]string{body["url"].(string)}, " "), "sk-live-secret") {
		t.Fatal("the key leaked into the response")
	}
	encoded, _ := json.Marshal(body)
	if strings.Contains(string(encoded), "sk-live-secret") {
		t.Fatalf("the key leaked into the response: %s", encoded)
	}
}

// Editing an endpoint with a blank key field uses the saved credential: that
// is the whole reason the server reads auth.json here.
func TestCustomProviderModelsUsesSavedKey(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	var sawAuth string
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"data":[{"id":"m"}]}`))
	}))
	defer gw.Close()

	// Save the definition and its key the way the form does.
	if status, body := req(http.MethodPut, "/api/providers/custom/gw", `{
		"baseUrl":"`+gw.URL+`","api":"openai-completions","key":"sk-saved",
		"models":[{"id":"m"}]}`); status != http.StatusOK {
		t.Fatalf("PUT status %d: %s", status, body)
	}
	status, body := loadModels(t, req, `{"baseUrl":"`+gw.URL+`","api":"openai-completions","id":"gw"}`)
	if status != http.StatusOK {
		t.Fatalf("status %d: %v", status, body)
	}
	if sawAuth != "Bearer sk-saved" {
		t.Fatalf("the saved key did not reach the endpoint: %q", sawAuth)
	}
}

func TestCustomProviderModelsFailures(t *testing.T) {
	cases := []struct {
		name     string
		handler  http.HandlerFunc
		body     string
		status   int
		kind     string
		contains []string
	}{
		{
			name: "key refused",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"message":"invalid api key sk-typed","type":"invalid_request_error"}}`))
			},
			body:     `{"baseUrl":"%URL%","api":"openai-completions","key":"sk-typed"}`,
			status:   http.StatusBadGateway,
			kind:     "auth",
			contains: []string{"refused the key", "401", "invalid api key ***"},
		},
		{
			name: "no key sent",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"message":"missing credentials"}}`))
			},
			body:     `{"baseUrl":"%URL%","api":"openai-completions"}`,
			status:   http.StatusBadGateway,
			kind:     "auth",
			contains: []string{"wants a key", "missing credentials"},
		},
		{
			name: "no list anywhere",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			body:     `{"baseUrl":"%URL%","api":"openai-completions","key":"k"}`,
			status:   http.StatusBadGateway,
			kind:     "missing",
			contains: []string{"No model list at that address", "/v1"},
		},
		{
			name:     "bad base url",
			handler:  func(w http.ResponseWriter, r *http.Request) {},
			body:     `{"baseUrl":"notaurl","api":"openai-completions","key":"k"}`,
			status:   http.StatusBadRequest,
			kind:     "input",
			contains: []string{"must start with http://"},
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
				w.Write([]byte(`{"error":{"message":"upstream is down"}}`))
			},
			body:     `{"baseUrl":"%URL%","api":"openai-completions","key":"k"}`,
			status:   http.StatusBadGateway,
			kind:     "upstream",
			contains: []string{"answered 502", "upstream is down"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, req := newCustomProviderServer(t)
			gw := httptest.NewServer(tc.handler)
			defer gw.Close()
			status, body := loadModels(t, req, strings.ReplaceAll(tc.body, "%URL%", gw.URL))
			if status != tc.status {
				t.Fatalf("status %d want %d: %v", status, tc.status, body)
			}
			if body["kind"] != tc.kind {
				t.Fatalf("kind = %v want %s (%v)", body["kind"], tc.kind, body)
			}
			msg, _ := body["error"].(string)
			for _, want := range tc.contains {
				if !strings.Contains(msg, want) {
					t.Fatalf("message %q is missing %q", msg, want)
				}
			}
			if strings.Contains(msg, "sk-typed") {
				t.Fatalf("the key leaked into the message: %q", msg)
			}
		})
	}
}

// An endpoint that lists nothing is not an error: the dialog says so and keeps
// the ids a person typed.
func TestCustomProviderModelsEmptyList(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer gw.Close()

	status, body := loadModels(t, req, `{"baseUrl":"`+gw.URL+`","api":"openai-completions","key":"k"}`)
	if status != http.StatusOK || body["count"] != float64(0) {
		t.Fatalf("status %d body %v", status, body)
	}
}

// An unreachable endpoint names the host, never a bare "fetch failed".
func TestCustomProviderModelsUnreachable(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	host := strings.TrimPrefix(gw.URL, "http://")
	gw.Close()

	status, body := loadModels(t, req, `{"baseUrl":"http://`+host+`","api":"openai-completions","key":"k"}`)
	if status != http.StatusBadGateway || body["kind"] != "transport" {
		t.Fatalf("status %d body %v", status, body)
	}
	if msg, _ := body["error"].(string); !strings.Contains(msg, host) {
		t.Fatalf("message does not name the host: %q", msg)
	}
}

// The literal route must not be swallowed by the {id} patterns.
func TestCustomProviderModelsRouteIsLiteral(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	if status, _ := req(http.MethodPut, "/api/providers/custom/models", `{"baseUrl":"https://x.com/v1","models":[{"id":"m"}]}`); status != http.StatusOK {
		t.Fatalf("PUT /api/providers/custom/models = %d (the literal POST route must not shadow the id route)", status)
	}
}
