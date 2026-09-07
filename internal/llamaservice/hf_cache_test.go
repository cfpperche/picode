package llamaservice

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func TestHFCacheOwnershipMatrix(t *testing.T) {
	for _, scenario := range []string{"new", "existing-blob", "existing-link", "bad-hash", "incomplete", "unknown-version", "outside-link", "symlink-blob", "shared", "shared-after-review", "changed-link", "agent-alias"} {
		t.Run(scenario, func(t *testing.T) {
			s := testService(t)
			configure(t, s)
			data := []byte("synthetic GGUF payload")
			hash := sha256.Sum256(data)
			digest := hex.EncodeToString(hash[:])
			base := filepath.Join(s.root, "models", "models--owner--test")
			snapshotDir := filepath.Join(base, "snapshots", strings.Repeat("a", 40))
			for _, dir := range []string{filepath.Join(base, "blobs"), snapshotDir} {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			blob := filepath.Join(base, "blobs", digest)
			link := filepath.Join(snapshotDir, "model.gguf")
			if err := os.WriteFile(blob, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("../../blobs/"+digest, link); err != nil {
				t.Fatal(err)
			}
			name, _ := filepath.Rel(s.root, blob)
			linkName, _ := filepath.Rel(s.root, link)
			j := store.LlamaJob{ID: "download", Model: "owner/test:Q4_0", Operation: "download", State: "succeeded", Progress: []store.LlamaProgress{{File: "model.gguf", Done: int64(len(data)), Total: int64(len(data))}}}
			s.doc.Current = &Release{Version: "b10809"}
			s.doc.Downloads = map[string][]string{j.ID: {}}
			switch scenario {
			case "existing-blob":
				s.doc.Downloads[j.ID] = []string{name}
			case "existing-link":
				s.doc.Downloads[j.ID] = []string{linkName}
			case "bad-hash":
				if err := os.WriteFile(blob, []byte("different"), 0600); err != nil {
					t.Fatal(err)
				}
			case "incomplete":
				j.Progress[0].Done--
			case "unknown-version":
				s.doc.Current.Version = "unknown"
			case "outside-link":
				if err := os.Remove(link); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(t.TempDir(), digest), link); err != nil {
					t.Fatal(err)
				}
			case "symlink-blob":
				other := filepath.Join(t.TempDir(), digest)
				if err := os.Rename(blob, other); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, blob); err != nil {
					t.Fatal(err)
				}
			}
			router := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": j.Model, "source": "cache", "status": map[string]string{"value": "unloaded"}}}})
			}))
			defer router.Close()
			j.Endpoint = router.URL
			s.doc.Applied.Port, _ = strconv.Atoi(strings.TrimPrefix(router.URL, "http://127.0.0.1:"))
			s.process = &exec.Cmd{}
			defer func() { s.process = nil }()
			if err := s.save(); err != nil {
				t.Fatal(err)
			}
			s.ObserveDownload(j)
			s.process = nil
			entry, recorded := s.doc.Models[name]
			want := scenario == "new" || scenario == "shared" || scenario == "shared-after-review" || scenario == "changed-link" || scenario == "agent-alias"
			if recorded != want {
				t.Fatalf("ownership=%v; want %v", recorded, want)
			}
			if !recorded {
				return
			}
			if entry.Snapshot != linkName || entry.SHA256 != digest {
				t.Fatal(entry)
			}
			switch scenario {
			case "shared":
				if err := os.Symlink(blob, filepath.Join(s.root, "models", "another.gguf")); err != nil {
					t.Fatal(err)
				}
			case "changed-link":
				if err := os.Remove(link); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("../../blobs/unknown", link); err != nil {
					t.Fatal(err)
				}
			case "agent-alias":
				w, err := s.st.AddWorkspace("test", t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				a, err := s.st.AddAgent(w.ID, "agent", "")
				if err != nil {
					t.Fatal(err)
				}
				provider, model := "llama.cpp", "owner/test:another-quant"
				if _, err = s.st.UpdateAgent(a.ID, store.AgentPatch{Provider: &provider, Model: &model}); err != nil {
					t.Fatal(err)
				}
			}
			p, err := s.Preview(Request{Action: "cleanup", Files: []string{name}, Revision: s.rev})
			if scenario != "new" && scenario != "shared-after-review" {
				if err == nil {
					t.Fatal("unsafe cleanup accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "shared-after-review" {
				if err := os.Symlink(blob, filepath.Join(s.root, "models", "later.gguf")); err != nil {
					t.Fatal(err)
				}
				if _, err = s.Execute(p.Token, false); err == nil {
					t.Fatal("stale reference review accepted")
				}
				if _, err = os.Stat(blob); err != nil {
					t.Fatal("shared blob removed", err)
				}
				return
			}
			if _, err = s.Execute(p.Token, false); err != nil {
				t.Fatal(err)
			}
			if done := finish(t, s); done.State != "succeeded" {
				t.Fatal(done)
			}
			for _, path := range []string{blob, link} {
				if _, err = os.Lstat(path); !os.IsNotExist(err) {
					t.Fatal("cache payload or owned link retained", err)
				}
			}
		})
	}
}
