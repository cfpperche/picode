package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakePi answers the two pi calls this surface makes: the model table the
// catalog parses, and `auth check --json`.
func fakePi(t *testing.T, authJSON string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake pi shell script is POSIX-only")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "pi")
	script := `#!/bin/sh
for a in "$@"; do
  if [ "$a" = "check" ]; then
    printf '%s\n' '` + authJSON + `'
    exit 0
  fi
done
echo "provider model context max-out thinking images"
echo "anthropic claude-sonnet-5 200k 64k yes yes"
echo "groq llama-4 128k 32k no no"
`
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUsageSummaryIsCacheOnlyAndSaysUnknown(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pi := fakePi(t, `{"status":"ready","provider":"anthropic","authType":"oauth"}`)
	ts := newTestServer(t, pi)

	res, err := ts.Client().Get(ts.URL + "/api/providers/usage")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body struct {
		Entries []struct {
			Provider string `json:"provider"`
			Status   string `json:"status"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	// Nothing is signed in on this HOME, so there is nothing to meter and
	// certainly nothing to invent.
	if len(body.Entries) != 0 {
		t.Fatalf("want no rows for a machine with no logins, got %+v", body.Entries)
	}
}

func TestProviderVerifyUsesPiAuthCheck(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pi := fakePi(t, `{"status":"ready","provider":"anthropic","authType":"oauth"}`)
	ts := newTestServer(t, pi)

	res, err := ts.Client().Post(ts.URL+"/api/providers/anthropic/verify", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["ok"] != true || body["status"] != "ready" || body["authType"] != "oauth" {
		t.Fatalf("verify said %v", body)
	}
}

func TestProviderVerifyReportsNotReady(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pi := fakePi(t, `{"status":"not_ready","provider":"groq","reason":"credentials_not_configured"}`)
	ts := newTestServer(t, pi)

	res, err := ts.Client().Post(ts.URL+"/api/providers/groq/verify", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["ok"] != false || body["reason"] != "credentials_not_configured" {
		t.Fatalf("verify said %v", body)
	}
}

func TestProviderVerifyRefusesAFlagAsAnID(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ts := newTestServer(t, fakePi(t, `{"status":"ready"}`))
	res, err := ts.Client().Post(ts.URL+"/api/providers/--credentials/verify", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("a provider id that is a flag must be refused, got %d", res.StatusCode)
	}
}

func TestProviderIDOK(t *testing.T) {
	for _, ok := range []string{"anthropic", "openai-codex", "zai-coding-cn", "llama.cpp", "qwen_1"} {
		if !providerIDOK(ok) {
			t.Fatalf("%q should be accepted", ok)
		}
	}
	for _, bad := range []string{"", "-flag", "--credentials", "a b", "a/b", "a;rm", strings.Repeat("x", 65)} {
		if providerIDOK(bad) {
			t.Fatalf("%q should be refused", bad)
		}
	}
}

func TestCatalogCarriesProviderRefCounts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ts := newTestServer(t, fakePi(t, `{"status":"ready"}`))
	res, err := ts.Client().Get(ts.URL + "/api/catalog")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var rep struct {
		Providers []struct {
			ID     string `json:"id"`
			Agents int    `json:"agents"`
		} `json:"providers"`
	}
	if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.Providers) == 0 {
		t.Fatal("catalog is empty")
	}
	for _, p := range rep.Providers {
		if p.Agents != 0 {
			t.Fatalf("%s claims %d agents on an empty store", p.ID, p.Agents)
		}
	}
}

func TestCatalogMarksAnEnvSuppliedProviderSignedIn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GROQ_API_KEY", "sk-from-the-environment")
	ts := newTestServer(t, fakePi(t, `{"status":"ready"}`))
	res, err := ts.Client().Get(ts.URL + "/api/catalog")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var rep struct {
		Providers []struct {
			ID       string `json:"id"`
			SignedIn bool   `json:"signedIn"`
			Source   string `json:"source"`
			EnvVar   string `json:"envVar"`
		} `json:"providers"`
	}
	if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, p := range rep.Providers {
		if p.ID != "groq" {
			continue
		}
		found = true
		if !p.SignedIn || p.Source != "environment" || p.EnvVar != "GROQ_API_KEY" {
			t.Fatalf("groq row is %+v; pi would use the variable, so the page must say so", p)
		}
	}
	if !found {
		t.Fatal("groq missing from the catalog")
	}
}
