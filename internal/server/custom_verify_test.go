package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// verifyCustom posts to the roster's verify route for a custom provider.
func verifyCustom(t *testing.T, req func(string, string, string) (int, string), id, body string) (int, map[string]any) {
	t.Helper()
	status, raw := req(http.MethodPost, "/api/providers/"+id+"/verify", body)
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("body is not JSON: %s", raw)
	}
	return status, out
}

// A custom endpoint is verified with one real request — the row's old answer
// came from credential presence, which stayed green with a bogus key.
func TestCustomVerifySpendsARealRequest(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	var path, auth, method string
	var body map[string]any
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, auth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		w.Write([]byte(`{"choices":[{"message":{"content":"hi"}}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`))
	}))
	defer gw.Close()

	// Save the definition and its key the way the form does.
	if status, b := req(http.MethodPut, "/api/providers/custom/gw", `{
		"baseUrl":"`+gw.URL+`/v1","api":"openai-completions","key":"sk-saved-key",
		"models":[{"id":"deepseek-v4.1-flash","reasoning":true,"thinkingLevels":["high"]}]}`); status != http.StatusOK {
		t.Fatalf("PUT status %d: %s", status, b)
	}

	status, res := verifyCustom(t, req, "gw", "")
	if status != http.StatusOK || res["ok"] != true || res["verified"] != true {
		t.Fatalf("status %d res %v", status, res)
	}
	if method != http.MethodPost || path != "/v1/chat/completions" {
		t.Fatalf("the probe did not reach the endpoint: %s %s", method, path)
	}
	if auth != "Bearer sk-saved-key" {
		t.Fatalf("the saved key did not travel: %q", auth)
	}
	if body["max_tokens"] != float64(1) {
		t.Fatalf("the probe was not minimal: %v", body)
	}
	if res["model"] != "deepseek-v4.1-flash" {
		t.Fatalf("model = %v", res["model"])
	}
	if label, _ := res["label"].(string); !strings.Contains(label, "answered in") || !strings.Contains(label, "3 in, 1 out") {
		t.Fatalf("label = %q", label)
	}
	// The key never travels back, even though the endpoint just saw it.
	encoded, _ := json.Marshal(res)
	if strings.Contains(string(encoded), "sk-saved-key") {
		t.Fatalf("the key leaked into the verdict: %s", encoded)
	}
}

// The dialog verifies before saving: the body carries what the form holds.
func TestCustomVerifyFromTheFormBody(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	var got map[string]any
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key sk-typed-key"}}`))
	}))
	defer gw.Close()

	status, res := verifyCustom(t, req, "notsaved", `{"baseUrl":"`+gw.URL+`/v1","api":"openai-completions","key":"sk-typed-key","model":"glm-4.6"}`)
	if status != http.StatusOK || res["ok"] != false {
		t.Fatalf("status %d res %v", status, res)
	}
	if res["status"] != "refused" {
		t.Fatalf("status field = %v", res["status"])
	}
	reason, _ := res["reason"].(string)
	if !strings.Contains(reason, "refused the key") || !strings.Contains(reason, "(401)") {
		t.Fatalf("reason = %q", reason)
	}
	if strings.Contains(reason, "sk-typed-key") {
		t.Fatalf("the key leaked into the reason: %q", reason)
	}
	if got["model"] != "glm-4.6" {
		t.Fatalf("the probe used %v", got["model"])
	}
}

// A custom definition with no models is answered, not crashed on.
func TestCustomVerifyWithoutAModel(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer gw.Close()

	status, res := verifyCustom(t, req, "empty", `{"baseUrl":"`+gw.URL+`","api":"openai-completions","key":"sk-abcdefgh"}`)
	if status != http.StatusOK || res["ok"] != false {
		t.Fatalf("status %d res %v", status, res)
	}
	if reason, _ := res["reason"].(string); !strings.Contains(reason, "no model to verify") {
		t.Fatalf("reason = %q", reason)
	}
}

// The built-ins still answer through pi, unchanged.
func TestBuiltinVerifyStillAsksPi(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	status, res := verifyCustom(t, req, "cheaperinference", "")
	if status != http.StatusOK {
		t.Fatalf("status %d res %v", status, res)
	}
	if _, isProbe := res["verified"]; isProbe {
		t.Fatalf("a built-in was verified with a real request: %v", res)
	}
}
