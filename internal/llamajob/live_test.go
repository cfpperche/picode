package llamajob

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// Opt-in real acceptance: docs/plans/llama-manager.md. The caller starts an
// isolated router/cache on port 18081; ordinary CI never downloads a model.
func TestLiveLlamaJobs(t *testing.T) {
	if os.Getenv("PICODE_LLAMA_LIVE") != "1" {
		t.Skip("opt-in real router acceptance: docs/plans/llama-manager.md")
	}
	target, _ := url.Parse("http://127.0.0.1:18081")
	proxy := httputil.NewSingleHostReverseProxy(target)
	var mu sync.Mutex
	posts := map[string]int{}
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			mu.Lock()
			posts[r.URL.Path]++
			mu.Unlock()
		}
		proxy.ServeHTTP(w, r)
	}))
	defer gateway.Close()
	path := filepath.Join(t.TempDir(), "jobs.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { st.Close() }()
	connection := func() (string, string) { return gateway.URL, "" }
	s, err := New(st, connection)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.Close() }()
	wait := func(id string, check func(store.LlamaJob) bool) store.LlamaJob {
		t.Helper()
		deadline := time.Now().Add(90 * time.Second)
		for time.Now().Before(deadline) {
			j, e := st.LlamaJob(id)
			if e != nil {
				t.Fatal(e)
			}
			if check(j) {
				return j
			}
			if !j.Active() {
				t.Fatalf("unexpected final state: %+v", j)
			}
			time.Sleep(100 * time.Millisecond)
		}
		j, _ := st.LlamaJob(id)
		t.Fatalf("real job timed out: %+v", j)
		return j
	}
	model := "Qwen3-4B-Q4_K_M"
	load, err := s.Start(model, "load", "live-load", false)
	if err != nil {
		t.Fatal(err)
	}
	observed := wait(load.ID, func(j store.LlamaJob) bool { return j.Observed == "loading" || j.State == "succeeded" })
	if observed.State == "succeeded" {
		t.Fatal("model loaded before restart injection; rerun with an unloaded router")
	}
	s.Close()
	st.Close()
	st, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err = New(st, connection)
	if err != nil {
		t.Fatal(err)
	}
	loaded := wait(load.ID, func(j store.LlamaJob) bool { return j.State == "succeeded" })
	mu.Lock()
	count := posts["/models/load"]
	mu.Unlock()
	if count != 1 {
		t.Fatal("load replayed", count)
	}
	unload, err := s.Start(model, "unload", "live-unload", false)
	if err != nil {
		t.Fatal(err)
	}
	unloaded := wait(unload.ID, func(j store.LlamaJob) bool { return j.State == "succeeded" })
	download, err := s.Start("ggml-org/Qwen3-0.6B-GGUF:Q4_0", "download", "live-download", false)
	if err != nil {
		t.Fatal(err)
	}
	downloading := wait(download.ID, func(j store.LlamaJob) bool {
		return j.Observed == "downloading" && len(j.Progress) > 0 && j.Progress[0].Done > 0 && j.Progress[0].Total > 0 && strings.HasSuffix(j.Progress[0].File, ".gguf")
	})
	if !downloading.CancelSupported {
		t.Fatal("expected verified build")
	}
	if _, err = s.Cancel(download.ID); err != nil {
		t.Fatal(err)
	}
	canceled := wait(download.ID, func(j store.LlamaJob) bool { return j.State == "canceled" })
	report := map[string]any{"loadAfterRestart": loaded, "unload": unloaded, "downloadProgress": downloading, "canceled": canceled, "posts": posts, "pass": true}
	raw, _ := json.MarshalIndent(report, "", "  ")
	t.Log(string(raw))
	if out := os.Getenv("PICODE_LLAMA_REPORT"); out != "" {
		if err = os.WriteFile(out, append(raw, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
