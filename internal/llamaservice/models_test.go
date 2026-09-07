package llamaservice

import (
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

func TestDownloadOwnershipMatrix(t *testing.T) {
	for _, scenario := range []string{"new", "existing", "untracked", "outside", "symlink", "alias", "failed"} {
		t.Run(scenario, func(t *testing.T) {
			s := testService(t)
			configure(t, s)
			path := filepath.Join(s.root, "models", "model.gguf")
			if scenario == "outside" {
				path = filepath.Join(t.TempDir(), "model.gguf")
			}
			if err := os.WriteFile(path, []byte("small synthetic download"), 0600); err != nil {
				t.Fatal(err)
			}
			if scenario == "symlink" {
				target := path
				path = filepath.Join(s.root, "models", "link.gguf")
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			}
			router := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": "test-model", "path": path, "status": map[string]string{"value": "unloaded"}}}})
			}))
			defer router.Close()
			port, _ := strconv.Atoi(strings.TrimPrefix(router.URL, "http://127.0.0.1:"))
			s.doc.Applied.Port = port
			s.process = &exec.Cmd{}
			defer func() { s.process = nil }()
			j := store.LlamaJob{ID: "download", Operation: "download", State: "succeeded", Endpoint: router.URL, Model: "test-model"}
			s.doc.Downloads = map[string][]string{j.ID: {}}
			if scenario == "existing" {
				s.doc.Downloads[j.ID] = []string{"models/model.gguf"}
			}
			if scenario == "untracked" {
				delete(s.doc.Downloads, j.ID)
			}
			if scenario == "alias" {
				j.Endpoint = strings.Replace(j.Endpoint, "127.0.0.1", "localhost", 1)
			}
			if scenario == "failed" {
				j.State = "failed"
			}
			if err := s.save(); err != nil {
				t.Fatal(err)
			}
			s.ObserveDownload(j)
			_, recorded := s.doc.Models["models/model.gguf"]
			if recorded != (scenario == "new") {
				t.Fatalf("ownership recorded=%v", recorded)
			}
		})
	}
}

func TestModelCleanupMatrix(t *testing.T) {
	for _, scenario := range []string{"owned", "unknown", "changed", "referenced"} {
		t.Run(scenario, func(t *testing.T) {
			s := testService(t)
			configure(t, s)
			path := filepath.Join(s.root, "models", "model.gguf")
			if err := os.WriteFile(path, []byte("disposable model file"), 0600); err != nil {
				t.Fatal(err)
			}
			hash, _, _ := hashRegular(path)
			info, _ := os.Stat(path)
			if scenario != "unknown" {
				s.doc.Models = map[string]OwnedModel{"models/model.gguf": {Model: "test-model", SHA256: hash, Job: "owned-download", Size: info.Size(), Modified: info.ModTime().UnixNano()}}
			}
			if scenario == "changed" {
				if err := os.WriteFile(path, []byte("different model"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "referenced" {
				w, err := s.st.AddWorkspace("test", t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				if _, err = s.st.AddAgent(w.ID, "agent", ""); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.save(); err != nil {
				t.Fatal(err)
			}
			files := s.Cache()
			if len(files) != 1 || files[0].Eligible != (scenario == "owned") {
				t.Fatal(files)
			}
			p, err := s.Preview(Request{Action: "cleanup", Files: []string{"models/model.gguf"}, Revision: s.rev})
			if scenario != "owned" {
				if err == nil {
					t.Fatal("unsafe model cleanup allowed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Execute(p.Token, false); err != nil {
				t.Fatal(err)
			}
			if j := finish(t, s); j.State != "succeeded" {
				t.Fatal(j)
			}
			if _, err = os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("eligible model was not deleted")
			}
		})
	}
}
func TestEndpointAliases(t *testing.T) {
	s := testService(t)
	s.doc.Config.Port = 18080
	s.doc.Applied.Port = 18081
	for _, endpoint := range []string{"http://localhost:18080", "http://127.0.0.1:18080", "http://[::1]:18081"} {
		if !s.ownsEndpoint(endpoint) {
			t.Fatal(endpoint)
		}
	}
	if s.ownsEndpoint("http://remote:18080") {
		t.Fatal("remote inferred as owned")
	}
}
