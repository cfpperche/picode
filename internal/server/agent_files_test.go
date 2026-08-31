package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSearchAgentFilesTable(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("README.md", "hi")
	mustWrite(".gitignore", "*")
	mustWrite("src/app.go", "pkg")
	mustWrite("src/util.go", "pkg")
	mustWrite("node_modules/left-pad/index.js", "nope")
	mustWrite(".git/HEAD", "ref")

	type row struct {
		name   string
		cwd    string
		q      string
		ok     bool
		want   []string
		forbid []string
	}
	rows := []row{
		{name: "1 missing cwd", cwd: filepath.Join(root, "nope"), q: "", ok: false},
		{name: "2 empty query top", cwd: root, q: "", ok: true, want: []string{"README.md"}, forbid: []string{"node_modules/left-pad/index.js", ".gitignore"}},
		{name: "2b query shows hidden", cwd: root, q: ".gitignore", ok: true, want: []string{".gitignore"}},
		{name: "3 match", cwd: root, q: "app", ok: true, want: []string{"src/app.go"}, forbid: []string{"README.md"}},
		{name: "4 zero hits", cwd: root, q: "zzzz-nope", ok: true, want: []string{}},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			hits, ok := searchAgentFiles(r.cwd, r.q)
			if ok != r.ok {
				t.Fatalf("ok=%v want %v", ok, r.ok)
			}
			got := map[string]bool{}
			for _, h := range hits {
				got[h.Path] = true
			}
			for _, w := range r.want {
				if !got[w] {
					t.Fatalf("missing %s in %+v", w, hits)
				}
			}
			for _, f := range r.forbid {
				if got[f] {
					t.Fatalf("leaked %s", f)
				}
			}
			if r.name == "4 zero hits" && len(hits) != 0 {
				t.Fatalf("hits=%+v", hits)
			}
		})
	}
}

func TestSearchAgentFilesRefusesEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	hits, ok := searchAgentFiles(root, "../secret")
	if !ok {
		t.Fatal("cwd should be ok")
	}
	for _, h := range hits {
		if h.Path == "../secret.txt" || h.Name == "secret.txt" {
			t.Fatalf("escaped: %+v", h)
		}
	}
}

func TestAgentFilesHTTP(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "main.go"), []byte("pkg"), 0o644); err != nil {
		t.Fatal(err)
	}
	wk := addWorkspaceWithAgent(t, ts, "App", proj)
	id := wk.Agent.ID
	if id == "" {
		t.Fatal("no agent")
	}
	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+id+"/files?q=main"))
	if got.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", got.StatusCode)
	}
	var body struct {
		Hits  []fileHit `json:"hits"`
		CwdOk bool      `json:"cwdOk"`
	}
	if err := json.NewDecoder(got.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.CwdOk || len(body.Hits) != 1 || body.Hits[0].Path != "main.go" {
		t.Fatalf("body=%+v", body)
	}

	miss := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/ag_nope/files"))
	if miss.StatusCode != http.StatusNotFound {
		t.Fatalf("missing agent = %d", miss.StatusCode)
	}
}

func TestReadAgentText(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ok.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin.dat"), []byte{'a', 0, 'b'}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	huge := filepath.Join(root, "huge.txt")
	f, err := os.Create(huge)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxAgentText + 1); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	out, code, err := readAgentText(root, "ok.go")
	if err != nil || code != http.StatusOK || out["text"] != "package x\n" {
		t.Fatalf("%d %v %+v", code, err, out)
	}
	_, code, err = readAgentText(root, filepath.Join(root, "ok.go"))
	if err != nil || code != http.StatusOK {
		t.Fatalf("abs %d %v", code, err)
	}
	_, code, err = readAgentText(root, "gone.go")
	if code != http.StatusNotFound || err == nil {
		t.Fatalf("gone %d %v", code, err)
	}
	_, code, err = readAgentText(root, "bin.dat")
	if code != http.StatusUnsupportedMediaType || err == nil {
		t.Fatalf("bin %d %v", code, err)
	}
	_, code, err = readAgentText(root, "huge.txt")
	if code != http.StatusRequestEntityTooLarge || err == nil {
		t.Fatalf("huge %d %v", code, err)
	}
	_, code, err = readAgentText(root, "dir")
	if code != http.StatusBadRequest || err == nil {
		t.Fatalf("dir %d %v", code, err)
	}
	_, code, err = readAgentText(root, "")
	if code != http.StatusBadRequest || err == nil {
		t.Fatalf("empty %d %v", code, err)
	}
	_, _, err = readAgentText(root, filepath.Join(t.TempDir(), "x"))
	if err == nil {
		t.Fatal("escape")
	}
}

func TestReadAgentBlob(t *testing.T) {
	root := t.TempDir()
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(filepath.Join(root, "a.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	mime, data, code, err := readAgentBlob(root, "a.png")
	if err != nil || code != http.StatusOK || mime != "image/png" || len(data) != len(png) {
		t.Fatalf("%s %d %v %d", mime, code, err, len(data))
	}
	_, _, code, err = readAgentBlob(root, "a.go")
	if code != http.StatusUnsupportedMediaType || err == nil {
		t.Fatalf("go %d %v", code, err)
	}
	_, _, _, err = readAgentBlob(root, filepath.Join(t.TempDir(), "x.png"))
	if err == nil {
		t.Fatal("escape")
	}
}

func TestAgentTextHTTP(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "main.go"), []byte("pkg main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wk := addWorkspaceWithAgent(t, ts, "App", proj)
	id := wk.Agent.ID
	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+id+"/text?path=main.go"))
	if got.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", got.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(got.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["text"] != "pkg main\n" || body["path"] != "main.go" {
		t.Fatalf("%+v", body)
	}
	if _, ok := body["mtime"].(float64); !ok || body["mtime"].(float64) == 0 {
		t.Fatalf("mtime %+v", body["mtime"])
	}
	miss := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+id+"/text?path=nope.go"))
	if miss.StatusCode != http.StatusNotFound {
		t.Fatalf("gone = %d", miss.StatusCode)
	}
}

func TestWriteAgentText(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "a.txt")
	if err := os.WriteFile(p, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code, err := readAgentText(root, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	mt := out["mtime"].(int64)
	got, code, err := writeAgentText(root, "a.txt", "two\n", mt)
	if err != nil || code != http.StatusOK || got["text"] != "two\n" {
		t.Fatalf("write %d %v %+v", code, err, got)
	}
	past := time.Now().Add(-2 * time.Second)
	if err := os.Chtimes(p, past, past); err != nil {
		t.Fatal(err)
	}
	_, code, err = writeAgentText(root, "a.txt", "three\n", got["mtime"].(int64))
	if code != http.StatusConflict || err == nil {
		t.Fatalf("stale %d %v", code, err)
	}
	_, code, err = writeAgentText(root, "a.txt", "x", 0)
	if code != http.StatusBadRequest || err == nil {
		t.Fatalf("mtime %d %v", code, err)
	}
	_, code, err = writeAgentText(root, "gone.txt", "x", 1)
	if code != http.StatusNotFound || err == nil {
		t.Fatalf("gone %d %v", code, err)
	}
}

func TestAgentTextPutHTTP(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wk := addWorkspaceWithAgent(t, ts, "App", proj)
	id := wk.Agent.ID
	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+id+"/text?path=a.txt"))
	var page map[string]any
	_ = json.NewDecoder(got.Body).Decode(&page)
	mt := int64(page["mtime"].(float64))
	put, _ := json.Marshal(map[string]any{"path": "a.txt", "text": "two\n", "mtime": mt})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/agents/"+id+"/text", bytes.NewReader(put))
	req.Header.Set("Content-Type", "application/json")
	resp := do(t, ts.Client(), req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put = %d", resp.StatusCode)
	}
	b, _ := os.ReadFile(filepath.Join(proj, "a.txt"))
	if string(b) != "two\n" {
		t.Fatalf("disk %q", b)
	}
}
