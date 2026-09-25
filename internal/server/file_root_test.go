package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/cfpperche/picode/internal/store"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

func TestFileRootPreconditionAcrossOwners(t *testing.T) {
	repo := gitRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "a"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "image.png"), []byte("preview bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := testStore(t)
	wk, err := st.AddWorkspace("Root", repo)
	if err != nil {
		t.Fatal(err)
	}
	ag, err := st.AddAgent(wk.ID, "Reader", "")
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(wk.ID, "Files", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	oldReveal := revealFn
	reveals := 0
	revealFn = func(string) error { reveals++; return nil }
	t.Cleanup(func() { revealFn = oldReveal })
	for _, base := range []string{"/api/workspaces/" + wk.ID, "/api/agents/" + ag.ID, "/api/terminals/" + term.ID} {
		for _, root := range []struct {
			name, value string
			blocked     bool
		}{
			{"legacy", "", false}, {"matching", canonDir(repo), false}, {"different", t.TempDir(), true},
		} {
			for _, route := range []struct {
				method, path string
				allowed      int
			}{
				{"GET", "/browse?dir=", 200}, {"GET", "/text?path=a", 200},
				{"GET", "/blob?path=image.png", 200}, {"GET", "/gitstatus?", 200},
				{"GET", "/gitdiff?path=a", 200}, {"GET", "/git/blob?hash=HEAD&path=a", 415},
				{"PUT", "/text?", 200}, {"POST", "/reveal?", 200},
			} {
				t.Run(base+"/"+root.name+route.method+route.path, func(t *testing.T) {
					before, err := os.ReadFile(filepath.Join(repo, "a"))
					if err != nil {
						t.Fatal(err)
					}
					info, err := os.Stat(filepath.Join(repo, "a"))
					if err != nil {
						t.Fatal(err)
					}
					text := "changed\n"
					if root.blocked {
						text = "must never be written\n"
					}
					payload, _ := json.Marshal(map[string]any{"path": "a", "text": text, "mtime": info.ModTime().UnixMilli()})
					if route.method == "POST" {
						payload = []byte("{}")
					}
					req, _ := http.NewRequest(route.method, ts.URL+base+route.path+"&root="+url.QueryEscape(root.value), bytes.NewReader(payload))
					count := reveals
					res := do(t, ts.Client(), req)
					defer res.Body.Close()
					want := route.allowed
					if root.blocked {
						want = http.StatusConflict
					}
					if res.StatusCode != want {
						t.Fatalf("status %d, want %d", res.StatusCode, want)
					}
					if root.blocked {
						after, _ := os.ReadFile(filepath.Join(repo, "a"))
						if !bytes.Equal(before, after) || count != reveals {
							t.Fatal("a failed root precondition had side effects")
						}
					}
				})
			}
		}
	}
	// A caller cannot turn a matching root into permission to escape the cwd.
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wk.ID+"/text?path=../outside&root="+url.QueryEscape(canonDir(repo))))
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("escape = %d", res.StatusCode)
	}
}

func TestFileRootRejectsTerminalCD(t *testing.T) {
	tm := tmux.New()
	if !tm.Available() {
		t.Skip("tmux not installed")
	}
	st := testStore(t)
	before, after := t.TempDir(), t.TempDir()
	for _, dir := range []string{before, after} {
		if err := os.WriteFile(filepath.Join(dir, "same.txt"), []byte("original"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	term, err := st.CreateTerminalIn(store.FreeWorkspaceID, "root precondition", before)
	if err != nil {
		t.Fatal(err)
	}
	session := tmux.ShellSessionName(term.ID)
	if err := tm.NewSession(t.Context(), session, before, "sh"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tm.KillSession(ctx, session)
	})
	ts := graphServer(t, st)
	base := ts.URL + "/api/terminals/" + term.ID
	if err := tm.SendKeys(t.Context(), session, "cd '"+strings.ReplaceAll(after, "'", "'\\''")+"'", "Enter"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		cwd, _ := tm.PaneCwd(t.Context(), session)
		if canonDir(cwd) == canonDir(after) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("terminal did not change directory")
		}
		time.Sleep(20 * time.Millisecond)
	}
	for _, method := range []string{"GET", "PUT"} {
		req, _ := http.NewRequest(method, base+"/text?path=same.txt&root="+url.QueryEscape(canonDir(before)), strings.NewReader(`{"path":"same.txt","text":"wrong folder","mtime":0}`))
		res := do(t, ts.Client(), req)
		res.Body.Close()
		if res.StatusCode != http.StatusConflict {
			t.Fatalf("%s stale root = %d", method, res.StatusCode)
		}
	}
	got, _ := os.ReadFile(filepath.Join(after, "same.txt"))
	if string(got) != "original" {
		t.Fatal("write reached the new cwd")
	}
	res := do(t, ts.Client(), mustGet(t, base+"/text?path=same.txt&root="+url.QueryEscape(canonDir(after))))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("refreshed root = %d", res.StatusCode)
	}
}

func TestFileRootWithWorktreeScope(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	if err := os.WriteFile(filepath.Join(side, "dot.png"), pngBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, side, "add", "dot.png")
	gitRun(t, side, "commit", "-m", "asset")
	if err := os.WriteFile(filepath.Join(side, "messy.txt"), []byte("sibling work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(store.FreeWorkspaceID, "Scope", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	for _, base := range []string{"/api/agents/" + agent.ID, "/api/terminals/" + term.ID} {
		routes := []string{"/gitstatus?", "/gitdiff?path=messy.txt", "/git/blob?hash=HEAD&path=dot.png"}
		if strings.Contains(base, "/agents/") {
			routes = append(routes, "/blob?path=dot.png")
		}
		for _, route := range routes {
			for _, root := range []struct {
				value string
				want  int
			}{{"", 200}, {canonDir(side), 200}, {canonDir(repo), 409}} {
				t.Run(base+route+root.value, func(t *testing.T) {
					code, _ := getBody(t, ts, base+route+"&worktree=side&root="+url.QueryEscape(root.value))
					if code != root.want {
						t.Fatalf("scoped root = %d, want %d", code, root.want)
					}
				})
			}
		}
	}
}
