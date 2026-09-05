package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/webhooks"
)

func TestWebhookAPI(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	defer receiver.Close()
	mux := http.NewServeMux()
	registerWebhookRoutes(mux, Deps{Store: st, Webhooks: webhooks.New(st)})
	request := func(method, path, body string, status int) []byte {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		if w.Code != status {
			t.Fatalf("%s %s: %d want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		return w.Body.Bytes()
	}
	request("GET", "/api/webhooks", "", 200)
	request("POST", "/api/webhooks", `{"url":"https://example.com","types":[]}`, 400)
	request("POST", "/api/webhooks", `{"url":"https://example.com","types":["agent."],"secret":"chosen"}`, 400)
	request("POST", "/api/webhooks", `{} {}`, 400)
	request("POST", "/api/webhooks", strings.Repeat("x", 17000), 400)
	body, _ := json.Marshal(map[string]any{"url": receiver.URL, "types": []string{"agent."}})
	raw := request("POST", "/api/webhooks", string(body), 201)
	var made struct {
		Webhook store.Webhook `json:"webhook"`
		Secret  string        `json:"secret"`
	}
	if err := json.Unmarshal(raw, &made); err != nil || made.Secret == "" {
		t.Fatalf("create: %s %v", raw, err)
	}
	path := "/api/webhooks/" + made.Webhook.ID
	raw = request("GET", "/api/webhooks", "", 200)
	if strings.Contains(string(raw), made.Secret) || strings.Contains(string(raw), `"secret"`) {
		t.Fatal("secret leaked on list")
	}
	request("PATCH", path, `{"revision":1,"url":"https://"}`, 400)
	raw = request("PATCH", path, `{"revision":1,"enabled":false}`, 200)
	if strings.Contains(string(raw), made.Secret) {
		t.Fatal("secret leaked on edit")
	}
	request("PATCH", path, `{"revision":1,"enabled":true}`, 409)
	raw = request("POST", path+"/test", "", 200)
	if !strings.Contains(string(raw), `"delivered":true`) {
		t.Fatalf("test not delivered: %s", raw)
	}
	request("POST", path+"/secret", `{"revision":1}`, 409)
	raw = request("POST", path+"/secret", `{"revision":2}`, 200)
	var rotated map[string]json.RawMessage
	_ = json.Unmarshal(raw, &rotated)
	if string(rotated["secret"]) == "" || strings.Contains(string(raw), made.Secret) {
		t.Fatal("rotation did not replace secret")
	}
	request("DELETE", path, "", 204)
	request("POST", path+"/test", "", 404)
	request("DELETE", path, "", 404)
}

func TestWebhookTestFailureAndUnavailable(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		io.WriteString(w, "private receiver response")
	}))
	defer receiver.Close()
	row, err := st.AddWebhook(receiver.URL, []string{"agent."})
	if err != nil {
		t.Fatal(err)
	}
	for _, available := range []bool{false, true} {
		d := Deps{Store: st}
		if available {
			d.Webhooks = webhooks.New(st)
		}
		mux := http.NewServeMux()
		registerWebhookRoutes(mux, d)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/webhooks/"+row.ID+"/test", nil))
		if !available && w.Code != 503 {
			t.Fatal(w.Code)
		}
		if available && (w.Code != 200 || !strings.Contains(w.Body.String(), `"delivered":false`) || strings.Contains(w.Body.String(), "private receiver")) {
			t.Fatal(w.Body.String())
		}
	}
}
