package install

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	rows := []struct {
		cur, lat string
		want     bool
	}{
		{"0.1.0", "0.1.0", false},
		{"v0.1.0", "0.1.0", false},
		{"0.1.0", "0.1.1", true},
		{"0.1.1", "0.1.0", false},
		{"0.1.0", "0.2.0", true},
		{"1.0.0", "0.9.9", false},
	}
	for _, r := range rows {
		if got := Newer(r.cur, r.lat); got != r.want {
			t.Errorf("Newer(%q,%q)=%v want %v", r.cur, r.lat, got, r.want)
		}
	}
}

func TestStripV(t *testing.T) {
	if stripV("v1.2.3") != "1.2.3" {
		t.Fatal(stripV("v1.2.3"))
	}
}

func TestLatestReleaseNone(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()
	oldRoot, oldC := APIRoot, HTTPClient
	APIRoot, HTTPClient = ts.URL, ts.Client()
	defer func() { APIRoot, HTTPClient = oldRoot, oldC }()
	_, err := LatestRelease()
	if err == nil || err.Error() != "no published release" {
		t.Fatalf("got %v", err)
	}
}

func TestLatestReleaseOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v0.2.0",
			"html_url": "https://github.com/cfpperche/picode/releases/tag/v0.2.0",
			"assets":   []any{},
		})
	}))
	defer ts.Close()
	oldRoot, oldC := APIRoot, HTTPClient
	APIRoot, HTTPClient = ts.URL, ts.Client()
	defer func() { APIRoot, HTTPClient = oldRoot, oldC }()
	rel, err := LatestRelease()
	if err != nil || rel.Tag != "0.2.0" {
		t.Fatalf("%+v %v", rel, err)
	}
}

func TestVerifySHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "picode-linux-amd64")
	if err := os.WriteFile(path, []byte("binary bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("binary bytes"))
	good := hex.EncodeToString(sum[:])
	sums := []byte("deadbeef  picode-darwin-arm64\n" + good + " *picode-linux-amd64\n")
	if err := VerifySHA256(path, sums, "picode-linux-amd64"); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := VerifySHA256(path, []byte(strings.Repeat("0", 64)+"  picode-linux-amd64\n"), "picode-linux-amd64"); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("tampered: %v", err)
	}
	if err := VerifySHA256(path, sums, "picode-windows-amd64.exe"); err == nil || !strings.Contains(err.Error(), "not listed") {
		t.Fatalf("missing: %v", err)
	}
}
