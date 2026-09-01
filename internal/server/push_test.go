package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/push"
	"github.com/cfpperche/picode/internal/store"
)

func pushTestServer(t *testing.T) (*httptest.Server, *store.Store, *httptest.Server) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	keys, err := push.LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// A stand-in push service that records the headers it was posted.
	var last http.Header
	svc := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last = r.Header.Clone()
		if strings.HasSuffix(r.URL.Path, "/gone") {
			w.WriteHeader(http.StatusGone)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(svc.Close)
	_ = last
	n := &push.Notifier{Store: st, Sender: &push.Sender{Keys: keys, Subject: "mailto:t@example.com", Client: svc.Client()}}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Push: n}).Handler)
	t.Cleanup(ts.Close)
	return ts, st, svc
}

func TestPushEndpoints(t *testing.T) {
	ts, st, svc := pushTestServer(t)
	var v struct{ PublicKey string }
	if code := getJSON(t, ts, "/api/push/vapid", &v); code != http.StatusOK {
		t.Fatalf("vapid = %d", code)
	}
	if pub, err := base64.RawURLEncoding.DecodeString(v.PublicKey); err != nil || len(pub) != 65 {
		t.Fatalf("public key %q", v.PublicKey)
	}
	// Keys shaped like a browser's: 65-byte P-256 point and 16-byte auth.
	p256 := base64.RawURLEncoding.EncodeToString(append([]byte{4}, make([]byte, 64)...))
	auth := base64.RawURLEncoding.EncodeToString(make([]byte, 16))
	sub := map[string]any{"endpoint": svc.URL + "/sub/1", "keys": map[string]string{"p256dh": p256, "auth": auth}, "deviceId": "d1", "prefs": map[string]bool{"actions": true, "finished": false}}
	res := postJSON(t, ts, "/api/push/subscriptions", sub)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("subscribe = %d", res.StatusCode)
	}
	var got store.PushSubscription
	_ = json.NewDecoder(res.Body).Decode(&got)
	if got.Endpoint != svc.URL+"/sub/1" || got.Prefs.Finished || !got.Prefs.Actions {
		t.Fatalf("sub = %+v", got)
	}
	bad := postJSON(t, ts, "/api/push/subscriptions", map[string]any{"endpoint": "http://plain/x", "keys": map[string]string{"p256dh": p256, "auth": auth}})
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("http endpoint = %d", bad.StatusCode)
	}
	patch := postJSONMethod(t, ts, http.MethodPatch, "/api/push/subscriptions", map[string]any{"endpoint": svc.URL + "/sub/1", "prefs": map[string]bool{"actions": false, "finished": true}})
	if patch.StatusCode != http.StatusOK {
		t.Fatalf("patch = %d", patch.StatusCode)
	}
	list, _ := st.ListPushSubscriptions()
	if len(list) != 1 || list[0].Prefs.Actions || !list[0].Prefs.Finished {
		t.Fatalf("prefs after patch = %+v", list)
	}
	// A test push goes to the stand-in service even with an invalid point:
	// the point (all zeros) is not on the curve, so encryption must refuse
	// it with 502, never crash.
	test := postJSON(t, ts, "/api/push/test", map[string]string{"endpoint": svc.URL + "/sub/1"})
	if test.StatusCode != http.StatusBadGateway {
		t.Fatalf("test with bogus key = %d", test.StatusCode)
	}
	del := postJSONMethod(t, ts, http.MethodDelete, "/api/push/subscriptions", map[string]string{"endpoint": svc.URL + "/sub/1"})
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", del.StatusCode)
	}
	if list, _ := st.ListPushSubscriptions(); len(list) != 0 {
		t.Fatalf("after delete = %+v", list)
	}
}

func TestPushEndpointsWithoutNotifier(t *testing.T) {
	// A store is required: New starts the session sweep, which reads
	// settings. Only Push is absent here.
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st}).Handler)
	defer ts.Close()
	res, err := http.Get(ts.URL + "/api/push/vapid")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("no notifier = %d", res.StatusCode)
	}
}
