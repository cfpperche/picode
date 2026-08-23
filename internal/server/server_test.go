package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	ts := httptest.NewServer(New("127.0.0.1:0").Handler)
	defer ts.Close()

	res, err := ts.Client().Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %v, want ok", body["status"])
	}
}

func TestVersionEndpoint(t *testing.T) {
	ts := httptest.NewServer(New("127.0.0.1:0").Handler)
	defer ts.Close()

	res, err := ts.Client().Get(ts.URL + "/api/version")
	if err != nil {
		t.Fatalf("GET /api/version: %v", err)
	}
	defer res.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["name"] != "picode" {
		t.Errorf("name field = %v, want picode", body["name"])
	}
	if v, ok := body["version"].(string); !ok || v == "" {
		t.Errorf("version field = %v, want non-empty string", body["version"])
	}
}

func TestIndexServed(t *testing.T) {
	ts := httptest.NewServer(New("127.0.0.1:0").Handler)
	defer ts.Close()

	res, err := ts.Client().Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if ct := res.Header.Get("Content-Type"); ct == "" {
		t.Error("Content-Type header missing for index")
	}
}
