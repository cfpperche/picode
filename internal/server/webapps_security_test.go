package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebappInstallRefusesRedirectToPrivateHost(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://10.0.0.1/secret", http.StatusFound)
	}))
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", `{"url":"`+origin.URL+`"}`)
	if code != http.StatusBadGateway {
		t.Fatalf("private redirect install = %d %v", code, body)
	}
}

func TestWebappInstallRefusesRedirectToLocalName(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://localhost:5984/_utils", http.StatusFound)
	}))
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", `{"url":"`+origin.URL+`"}`)
	if code != http.StatusBadGateway {
		t.Fatalf("local-name redirect install = %d %v", code, body)
	}
}

func TestWebappInstallRefusesDirectPrivateLiteral(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><title>Direct</title></html>`))
	}))
	_ = origin
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", `{"url":"http://10.255.255.1/"}`)
	if code != http.StatusBadGateway {
		t.Fatalf("private literal install = %d %v", code, body)
	}
}

func TestWebappRedirectLeavingHTTPRefused(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "ftp://example.com/x", http.StatusFound)
	})
	origin := webappOrigin(t, handler)
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", `{"url":"`+origin.URL+`"}`)
	if code != http.StatusBadGateway {
		t.Fatalf("ftp redirect install = %d %v", code, body)
	}
}

func TestWebappRedirectToDifferentPublicHostAllowed(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><title>Moved App</title>`))
	}))
	defer target.Close()
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", `{"url":"`+origin.URL+`"}`)
	if code != http.StatusOK {
		t.Fatalf("public redirect install = %d %v", code, body)
	}
	if body["name"] != "Moved App" {
		t.Fatalf("redirect metadata = %v", body)
	}
}
