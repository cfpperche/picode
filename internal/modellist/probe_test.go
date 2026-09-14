package modellist

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// readBody returns the request body as a map, so a test can assert on exactly
// what was sent (the point of a probe is that it is tiny).
func readBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("body is not JSON: %s", raw)
	}
	return out
}

// The probe must be the smallest real request: one word in, one token out, and
// it must report what the endpoint said it spent.
func TestProbeSendsTheSmallestRequest(t *testing.T) {
	var got map[string]any
	var path, auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, auth = r.URL.Path, r.Header.Get("Authorization")
		got = readBody(t, r)
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`))
	}))
	defer srv.Close()

	res, err := Probe(context.Background(), srv.URL+"/v1", APIOpenAICompletions, "sk-secret", "deepseek-v4.1-flash")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/v1/chat/completions" {
		t.Fatalf("path = %s", path)
	}
	if auth != "Bearer sk-secret" {
		t.Fatalf("auth = %q", auth)
	}
	if got["max_tokens"] != float64(1) || got["stream"] != false || got["model"] != "deepseek-v4.1-flash" {
		t.Fatalf("body = %v", got)
	}
	msgs, _ := got["messages"].([]any)
	first, _ := msgs[0].(map[string]any)
	if len(msgs) != 1 || first["content"] != probePrompt {
		t.Fatalf("messages = %v", got["messages"])
	}
	if res.Model != "deepseek-v4.1-flash" || res.InputTokens != 3 || res.OutputTokens != 1 {
		t.Fatalf("result = %+v", res)
	}
	if res.MS < 0 {
		t.Fatalf("ms = %d", res.MS)
	}
}

// A gateway that requires the newer ceiling field gets one retry, not a lecture.
func TestProbeFallsBackToMaxCompletionTokens(t *testing.T) {
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := readBody(t, r)
		bodies = append(bodies, body)
		if _, old := body["max_tokens"]; old {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"message":"Unsupported parameter: 'max_tokens' is not supported with this model. Use 'max_completion_tokens' instead."}}`))
			return
		}
		w.Write([]byte(`{"choices":[{"message":{"content":"hi"}}]}`))
	}))
	defer srv.Close()

	if _, err := Probe(context.Background(), srv.URL+"/v1", APIOpenAICompletions, "k", "m"); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 {
		t.Fatalf("attempts = %d", len(bodies))
	}
	if bodies[1]["max_completion_tokens"] != float64(1) {
		t.Fatalf("retry body = %v", bodies[1])
	}
}

// The other API types use their own route, headers and body shape.
func TestProbePerAPIType(t *testing.T) {
	cases := []struct {
		api        string
		path       string
		header     string
		wantHeader string
		bodyKeys   []string
		body       string
	}{
		{"anthropic-messages", "/v1/messages", "x-api-key", "sk-ant", []string{"model", "max_tokens", "messages"}, `{"usage":{"input_tokens":2,"output_tokens":1}}`},
		{"openai-responses", "/v1/responses", "Authorization", "Bearer sk", []string{"model", "input", "max_output_tokens"}, `{"output":[]}`},
		{"google-generative-ai", "/v1beta/models/gemini-2.5-pro:generateContent", "x-goog-api-key", "goog", []string{"contents", "generationConfig"}, `{"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":1}}`},
	}
	for _, tc := range cases {
		t.Run(tc.api, func(t *testing.T) {
			var got map[string]any
			var path, header string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.Path
				header = r.Header.Get(tc.header)
				got = readBody(t, r)
				w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			base := srv.URL + "/v1"
			model := "m"
			if tc.api == APIGoogle {
				base = srv.URL + "/v1beta"
				model = "gemini-2.5-pro"
			}
			res, err := Probe(context.Background(), base, tc.api, strings.TrimPrefix(tc.wantHeader, "Bearer "), model)
			if err != nil {
				t.Fatal(err)
			}
			if path != tc.path {
				t.Fatalf("path = %s want %s", path, tc.path)
			}
			if header != tc.wantHeader {
				t.Fatalf("header %s = %q want %q", tc.header, header, tc.wantHeader)
			}
			for _, key := range tc.bodyKeys {
				if _, ok := got[key]; !ok {
					t.Fatalf("body is missing %s: %v", key, got)
				}
			}
			if res.Model != model {
				t.Fatalf("model = %q", res.Model)
			}
		})
	}
}

func TestProbeFailures(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		kind   Kind
		in     []string
	}{
		{"refused key", http.StatusUnauthorized, `{"error":{"message":"invalid api key sk-typed"}}`, KindAuth, []string{"invalid api key"}},
		{"quota spent", http.StatusTooManyRequests, `{"error":{"message":"insufficient credits"}}`, KindQuota, []string{"insufficient credits"}},
		{"payment needed", http.StatusPaymentRequired, `{}`, KindQuota, []string{}},
		{"unknown model", http.StatusNotFound, `{"error":{"message":"model not found"}}`, KindModel, []string{"the-model", "model not found"}},
		{"bad request", http.StatusBadRequest, `{"error":{"message":"temperature must be a number"}}`, KindInput, []string{"temperature"}},
		{"server error", http.StatusInternalServerError, `{"error":{"message":"boom"}}`, KindUpstream, []string{"boom"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			_, err := Probe(context.Background(), srv.URL+"/v1", APIOpenAICompletions, "sk-typed", "the-model")
			var typed *Error
			if !errors.As(err, &typed) || typed.Kind != tc.kind {
				t.Fatalf("err = %v want kind %s", err, tc.kind)
			}
			for _, want := range tc.in {
				if !strings.Contains(typed.Detail, want) {
					t.Fatalf("detail %q is missing %q", typed.Detail, want)
				}
			}
			if strings.Contains(typed.Detail, "sk-typed") {
				t.Fatalf("the key leaked into the detail: %q", typed.Detail)
			}
		})
	}
}

func TestProbeRefusesInput(t *testing.T) {
	if _, err := Probe(context.Background(), "https://x.com/v1", APIOpenAICompletions, "k", ""); err == nil {
		t.Fatal("an empty model must be refused")
	} else {
		var typed *Error
		if !errors.As(err, &typed) || typed.Kind != KindInput {
			t.Fatalf("err = %v", err)
		}
	}
	if _, err := Probe(context.Background(), "notaurl", APIOpenAICompletions, "k", "m"); err == nil {
		t.Fatal("a bad base URL must be refused")
	}
	if _, err := Probe(context.Background(), "https://x.com/v1", "telepathy", "k", "m"); err == nil {
		t.Fatal("an unknown API type must be refused")
	}
}

// Nothing but timing, model and usage leaves the probe: no completion text.
func TestProbeResultCarriesNoText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"SECRET-COMPLETION"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer srv.Close()

	res, err := Probe(context.Background(), srv.URL+"/v1", APIOpenAICompletions, "k", "m")
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(res)
	if strings.Contains(string(encoded), "SECRET-COMPLETION") {
		t.Fatalf("the answer leaked: %s", encoded)
	}
}

// A transport failure names the host, like the listing does.
func TestProbeUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	host := srv.URL
	srv.Close()

	_, err := Probe(context.Background(), host, APIOpenAICompletions, "k", "m")
	var typed *Error
	if !errors.As(err, &typed) || typed.Kind != KindTransport || typed.Host == "" {
		t.Fatalf("err = %v", err)
	}
}
