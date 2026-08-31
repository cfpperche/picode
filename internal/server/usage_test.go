package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestProviderUsageUnsigned(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ts := newTestServer(t, "cat")
	res, err := ts.Client().Get(ts.URL + "/api/providers/anthropic/usage")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "unsupported" {
		t.Fatalf("%v", body)
	}
	if body["provider"] != "anthropic" {
		t.Fatalf("provider %v", body["provider"])
	}
}

func TestAccountUsageUnknown(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ts := newTestServer(t, "cat")
	res, err := ts.Client().Get(ts.URL + "/api/providers/anthropic/accounts/missing/usage")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "auth_required" {
		t.Fatalf("%v", body)
	}
}
