package browserhost

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoteWinsOverServerJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PICODE_DATA", dir)
	_ = os.WriteFile(filepath.Join(dir, "server.json"), []byte(`{"url":"https://localhost:8445"}`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "token"), []byte("localtoken\n"), 0o600)

	if u, err := ReadServerURL(); err != nil || u != "https://localhost:8445" {
		t.Fatalf("local: %q %v", u, err)
	}
	req, _ := http.NewRequest("GET", "https://localhost:8445/api/health", nil)
	authorize(req)
	if req.Header.Get("Authorization") != "Bearer localtoken" {
		t.Fatalf("local bearer: %q", req.Header.Get("Authorization"))
	}

	path, err := WriteRemote(Remote{URL: "https://box.tail.ts.net:8445/", Token: "remotetoken"})
	if err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
		t.Fatalf("remote.json mode %v", st.Mode().Perm())
	}
	if u, _ := ReadServerURL(); u != "https://box.tail.ts.net:8445" {
		t.Fatalf("remote: %q", u)
	}
	req, _ = http.NewRequest("GET", "https://box.tail.ts.net:8445/api/health", nil)
	authorize(req)
	if req.Header.Get("Authorization") != "Bearer remotetoken" {
		t.Fatalf("remote bearer: %q", req.Header.Get("Authorization"))
	}
	if _, err := WriteRemote(Remote{URL: "box:8445"}); err == nil {
		t.Fatal("bare host accepted")
	}
}

func TestVerifyTLSFor(t *testing.T) {
	for _, skip := range []string{"https://localhost:8445", "https://127.0.0.1:8445", "https://192.168.1.4:8445", "https://100.64.0.9:8445", "https://[::1]:8445"} {
		if VerifyTLSFor(skip) {
			t.Errorf("%s should accept the local cert", skip)
		}
	}
	for _, verify := range []string{"https://box.tail.ts.net:8445", "https://picode.example.com"} {
		if !VerifyTLSFor(verify) {
			t.Errorf("%s must verify", verify)
		}
	}
}
