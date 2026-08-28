package pipkg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNpmName(t *testing.T) {
	if NpmName("npm:pi-mcp-adapter") != "pi-mcp-adapter" {
		t.Fatal(NpmName("npm:pi-mcp-adapter"))
	}
	if NpmName("npm:@foo/bar@1.2.3") != "@foo/bar" {
		t.Fatal(NpmName("npm:@foo/bar@1.2.3"))
	}
	if NpmName("npm:foo@1.2.3") != "foo" {
		t.Fatal(NpmName("npm:foo@1.2.3"))
	}
	if NpmName("git:github.com/x/y") != "" {
		t.Fatal("git")
	}
}

func TestNpmPinned(t *testing.T) {
	if NpmPinned("npm:pi-mcp-adapter") {
		t.Fatal("unpinned")
	}
	if !NpmPinned("npm:foo@1.2.3") {
		t.Fatal("exact")
	}
	if !NpmPinned("npm:@foo/bar@2.0.0") {
		t.Fatal("scoped exact")
	}
	if NpmPinned("npm:foo@^1.2.3") {
		t.Fatal("range")
	}
	if NpmPinned("./local") {
		t.Fatal("path")
	}
}

func TestNewer(t *testing.T) {
	if !Newer("2.0.0", "1.9.9") {
		t.Fatal("major")
	}
	if !Newer("1.3.2", "1.3.1") {
		t.Fatal("patch")
	}
	if Newer("1.3.1", "1.3.1") {
		t.Fatal("equal")
	}
	if Newer("1.0.0", "1.2.0") {
		t.Fatal("older")
	}
	if Newer("nope", "1.0.0") {
		t.Fatal("garbage")
	}
}

func TestUpdateArgs(t *testing.T) {
	got := strings.Join(UpdateArgs("npm:foo"), " ")
	if got != "update --extension npm:foo --no-approve" {
		t.Fatal(got)
	}
}

func TestCheckUpdates(t *testing.T) {
	user := t.TempDir()
	writeInstalled(t, user, "behind", "1.0.0")
	writeInstalled(t, user, "current", "2.0.0")
	writeInstalled(t, user, "pinned-pkg", "1.0.0")
	body := `{
	  "packages": [
	    "npm:behind",
	    "npm:current",
	    "npm:pinned-pkg@1.0.0",
	    "./local-path",
	    "npm:missing"
	  ]
	}`
	if err := os.WriteFile(filepath.Join(user, "settings.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/behind/latest"):
			_, _ = w.Write([]byte(`{"version":"1.2.0"}`))
		case strings.Contains(r.URL.Path, "/current/latest"):
			_, _ = w.Write([]byte(`{"version":"2.0.0"}`))
		case strings.Contains(r.URL.Path, "/pinned-pkg/latest"):
			t.Error("pinned should not hit registry")
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	rep, err := checkUpdates(context.Background(), ts.Client(), ts.URL, user, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Updates) != 1 {
		t.Fatalf("%+v", rep.Updates)
	}
	u := rep.Updates[0]
	if u.Source != "npm:behind" || u.Scope != "user" || u.Current != "1.0.0" || u.Latest != "1.2.0" {
		t.Fatalf("%+v", u)
	}
}

func TestCheckUpdatesEmpty(t *testing.T) {
	rep, err := checkUpdates(context.Background(), nil, "", t.TempDir(), "")
	if err != nil || len(rep.Updates) != 0 {
		t.Fatalf("%v %+v", err, rep)
	}
}

func TestCheckUpdatesRegistryMiss(t *testing.T) {
	user := t.TempDir()
	writeInstalled(t, user, "ok", "1.0.0")
	writeInstalled(t, user, "gone", "1.0.0")
	body := `{"packages":["npm:ok","npm:gone"]}`
	if err := os.WriteFile(filepath.Join(user, "settings.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/ok/latest") {
			_, _ = w.Write([]byte(`{"version":"1.1.0"}`))
			return
		}
		http.Error(w, "no", 500)
	}))
	defer ts.Close()
	rep, err := checkUpdates(context.Background(), ts.Client(), ts.URL, user, "")
	if err != nil || len(rep.Updates) != 1 || rep.Updates[0].Source != "npm:ok" {
		t.Fatalf("%v %+v", err, rep)
	}
}

func writeInstalled(t *testing.T, userDir, name, ver string) {
	t.Helper()
	dir := filepath.Join(userDir, "npm", "node_modules", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"name":"` + name + `","version":"` + ver + `"}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
