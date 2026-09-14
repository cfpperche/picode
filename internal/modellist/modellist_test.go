package modellist

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// list is a tiny helper: ask a test server for models.
func list(t *testing.T, srv *httptest.Server, api, key string) (Result, error) {
	t.Helper()
	return List(context.Background(), srv.URL, api, key)
}

func TestListOpenAIShape(t *testing.T) {
	var gotAuth, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		if r.URL.Path != "/models" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"object":"list","data":[
			{"id":"deepseek-v4.1-flash","context_length":1048576,"top_provider":{"max_completion_tokens":384000}},
			{"id":"glm-4.6","context_window":200000,"max_output_tokens":131072},
			"bare-string-id"
		]}`))
	}))
	defer srv.Close()

	res, err := list(t, srv, APIOpenAICompletions, "sk-secret")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer sk-secret" {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if !strings.Contains(gotUA, "picode") {
		t.Fatalf("user agent = %q", gotUA)
	}
	want := []Model{
		{ID: "deepseek-v4.1-flash", ContextWindow: 1048576, MaxTokens: 384000},
		{ID: "glm-4.6", ContextWindow: 200000, MaxTokens: 131072},
		{ID: "bare-string-id"},
	}
	if len(res.Models) != len(want) {
		t.Fatalf("models = %+v", res.Models)
	}
	for i, w := range want {
		if res.Models[i] != w {
			t.Fatalf("model %d = %+v want %+v", i, res.Models[i], w)
		}
	}
	if !strings.HasSuffix(res.URL, "/models") {
		t.Fatalf("url = %s", res.URL)
	}
}

func TestListAnthropicShapeAndHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "sk-ant" {
			t.Errorf("x-api-key = %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Error("anthropic-version missing")
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("bearer sent to anthropic: %q", got)
		}
		w.Write([]byte(`{"data":[{"type":"model","id":"claude-sonnet-4-5","display_name":"Sonnet"}]}`))
	}))
	defer srv.Close()

	res, err := list(t, srv, APIAnthropic, "sk-ant")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Models) != 1 || res.Models[0].ID != "claude-sonnet-4-5" {
		t.Fatalf("models = %+v", res.Models)
	}
}

func TestListGoogleShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-goog-api-key"); got != "goog" {
			t.Errorf("x-goog-api-key = %q", got)
		}
		w.Write([]byte(`{"models":[{"name":"models/gemini-2.5-pro","inputTokenLimit":1048576,"outputTokenLimit":65536}]}`))
	}))
	defer srv.Close()

	res, err := list(t, srv, APIGoogle, "goog")
	if err != nil {
		t.Fatal(err)
	}
	want := Model{ID: "gemini-2.5-pro", ContextWindow: 1048576, MaxTokens: 65536}
	if len(res.Models) != 1 || res.Models[0] != want {
		t.Fatalf("models = %+v want %+v", res.Models, want)
	}
}

// A gateway whose base URL has no /v1: the listing lives one level down, and
// only a missing list makes us try it.
func TestListFallsThroughOn404(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path != "/v1/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"data":[{"id":"m"}]}`))
	}))
	defer srv.Close()

	res, err := list(t, srv, APIOpenAICompletions, "k")
	if err != nil {
		t.Fatal(err)
	}
	if res.Models[0].ID != "m" || !strings.HasSuffix(res.URL, "/v1/models") {
		t.Fatalf("res = %+v", res)
	}
	if len(paths) != 2 || paths[0] != "/models" || paths[1] != "/v1/models" {
		t.Fatalf("paths = %v", paths)
	}
}

// A refused key must not be retried on the second path, and its message must
// not carry the key back.
func TestListAuthRefused(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key sk-secret","type":"invalid_request_error"}}`))
	}))
	defer srv.Close()

	_, err := list(t, srv, APIOpenAICompletions, "sk-secret")
	var typed *Error
	if !errors.As(err, &typed) || typed.Kind != KindAuth {
		t.Fatalf("err = %v", err)
	}
	if calls != 1 {
		t.Fatalf("a refused key was retried (%d calls)", calls)
	}
	if !strings.Contains(typed.Detail, "invalid api key") {
		t.Fatalf("detail = %q", typed.Detail)
	}
	// The provider echoed our key; the message we build must not repeat it.
	if strings.Contains(typed.Detail, "sk-secret") {
		t.Fatalf("detail leaked the key: %q", typed.Detail)
	}
}

func TestListMissingEverywhere(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := list(t, srv, APIOpenAICompletions, "k")
	var typed *Error
	if !errors.As(err, &typed) || typed.Kind != KindMissing {
		t.Fatalf("err = %v", err)
	}
}

func TestListTransportFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	host := srv.URL
	srv.Close() // nothing listens now

	_, err := List(context.Background(), host, APIOpenAICompletions, "k")
	var typed *Error
	if !errors.As(err, &typed) || typed.Kind != KindTransport {
		t.Fatalf("err = %v", err)
	}
	if typed.Host == "" {
		t.Fatal("the failed host must be named")
	}
}

func TestListEmptyIsNotAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer srv.Close()

	res, err := list(t, srv, APIOpenAICompletions, "k")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Models) != 0 {
		t.Fatalf("models = %+v", res.Models)
	}
}

func TestListRefusesInput(t *testing.T) {
	for _, tc := range []struct{ name, base, api string }{
		{"no url", "", APIOpenAICompletions},
		{"not a url", "notaurl", APIOpenAICompletions},
		{"wrong scheme", "ftp://example.com/v1", APIOpenAICompletions},
		{"unknown api", "https://example.com/v1", "telepathy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := List(context.Background(), tc.base, tc.api, "")
			var typed *Error
			if !errors.As(err, &typed) || typed.Kind != KindInput {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestListDedupesAndIgnoresEmptyIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"id":"m"},{"id":"m"},{"id":""},{"display_name":"no id"},{"id":"other"}]}`))
	}))
	defer srv.Close()

	res, err := list(t, srv, APIOpenAICompletions, "k")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Models) != 2 || res.Models[0].ID != "m" || res.Models[1].ID != "other" {
		t.Fatalf("models = %+v", res.Models)
	}
}

func TestListGarbageBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body>hi</body></html>`))
	}))
	defer srv.Close()

	_, err := list(t, srv, APIOpenAICompletions, "k")
	var typed *Error
	if !errors.As(err, &typed) || typed.Kind != KindInput {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(typed.Detail, "not a model list") {
		t.Fatalf("detail = %q", typed.Detail)
	}
}

// A base URL that already ends in /v1 must not grow a second one.
func TestListDoesNotDoubleV1(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := List(context.Background(), srv.URL+"/v1", APIOpenAICompletions, "k")
	if err == nil {
		t.Fatal("expected a missing list")
	}
	if path != "/v1/models" {
		t.Fatalf("path = %s", path)
	}
}
