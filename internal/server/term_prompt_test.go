package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func TestDropFileNameAndPaste(t *testing.T) {
	got := dropFileName("../etc/passwd")
	if strings.Contains(got, "/") || strings.Contains(got, "..") {
		t.Fatalf("name = %q", got)
	}
	paste := buildPromptPaste("look", []string{".picode/drop/a.png", "notes.pdf"})
	if paste != "look\n@.picode/drop/a.png\n@notes.pdf" {
		t.Fatalf("paste = %q", paste)
	}
	if buildPromptPaste("", []string{"shot.png"}) != "@shot.png" {
		t.Fatal("paths only")
	}
}

func TestCheckPromptPaths(t *testing.T) {
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "ok.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(cwd, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "big.bin"), []byte(strings.Repeat("a", maxDropBytes+1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := checkPromptPaths(cwd, []string{"ok.png"}); err != nil {
		t.Fatal(err)
	}
	if _, err := checkPromptPaths(cwd, []string{"../ok.png"}); err == nil {
		t.Fatal("escape")
	}
	if _, err := checkPromptPaths(cwd, []string{"dir"}); err == nil {
		t.Fatal("dir")
	}
	if _, err := checkPromptPaths(cwd, []string{"missing.png"}); err == nil {
		t.Fatal("missing")
	}
	if _, err := checkPromptPaths(cwd, []string{"big.bin"}); err == nil || !strings.Contains(err.Error(), "4 MB") {
		t.Fatalf("huge: %v", err)
	}
	if _, err := checkPromptPaths(cwd, []string{"a", "b", "c", "d", "e"}); err == nil || !strings.Contains(err.Error(), "at most 4") {
		t.Fatalf("five: %v", err)
	}
}

func TestDecodeDropData(t *testing.T) {
	raw, err := decodeDropData(base64.StdEncoding.EncodeToString([]byte("hi")))
	if err != nil || string(raw) != "hi" {
		t.Fatalf("%q %v", raw, err)
	}
	if _, err := decodeDropData(""); err == nil {
		t.Fatal("empty")
	}
	if _, err := decodeDropData(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", maxDropBytes+1)))); err == nil {
		t.Fatal("big")
	}
}

func TestWriteDropFileGitignore(t *testing.T) {
	cwd := t.TempDir()
	// A project's own .gitignore, tracked in git, must come out untouched:
	// the staging folder used to gain a silent line in exactly this file.
	if err := os.WriteFile(filepath.Join(cwd, ".gitignore"), []byte("node_modules\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := writeDropFile(cwd, "shot.png", []byte("png"))
	if err != nil {
		t.Fatal(err)
	}
	path, _ := out["path"].(string)
	if !strings.HasPrefix(path, ".picode/drop/") || !strings.HasSuffix(path, ".png") {
		t.Fatalf("path = %q", path)
	}
	raw, err := os.ReadFile(filepath.Join(cwd, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "node_modules\n" {
		t.Fatalf("project .gitignore was rewritten: %q", raw)
	}
	nested, err := os.ReadFile(filepath.Join(cwd, ".picode", ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(nested) != "drop/\n" {
		t.Fatalf("nested gitignore = %q", nested)
	}
}

func TestWriteDropFileNoProjectGitignore(t *testing.T) {
	// A project with no .gitignore at all used to leave staged attachments
	// fully untracked and visible in `git status` — nothing ever covered
	// them. The nested one must exist regardless.
	cwd := t.TempDir()
	if _, err := writeDropFile(cwd, "shot.png", []byte("png")); err != nil {
		t.Fatal(err)
	}
	nested, err := os.ReadFile(filepath.Join(cwd, ".picode", ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(nested) != "drop/\n" {
		t.Fatalf("nested gitignore = %q", nested)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".gitignore")); err == nil {
		t.Fatal("no project .gitignore should have been created")
	}
}

func TestSweepDropDirAge(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.png")
	fresh := filepath.Join(dir, "fresh.png")
	for _, p := range []string{old, fresh} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(old, time.Now().Add(-8*24*time.Hour), time.Now().Add(-8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	sweepDropDir(dir, 7*24*time.Hour)
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old file should be gone, err = %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatalf("fresh file should survive: %v", err)
	}
}

func TestWriteDropFileSweepsStaleSiblings(t *testing.T) {
	// The sweep runs opportunistically on the next drop into the same
	// project — nothing else ever calls it.
	cwd := t.TempDir()
	dropDir := filepath.Join(cwd, ".picode", "drop")
	if err := os.MkdirAll(dropDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dropDir, "ancient-shot.png")
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(stale, time.Now().Add(-30*24*time.Hour), time.Now().Add(-30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := writeDropFile(cwd, "new.png", []byte("png")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale attachment should have been swept, err = %v", err)
	}
}

func fakeTmuxBin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	paste := filepath.Join(dir, "paste")
	script := "#!/bin/sh\ncmd=\"$1\"; shift\n" +
		"case \"$cmd\" in\n" +
		"has-session) exit 0 ;;\n" +
		"load-buffer) cat > \"" + dir + "/buffer\"; exit 0 ;;\n" +
		"paste-buffer) cat \"" + dir + "/buffer\" >> \"" + paste + "\"; exit 0 ;;\n" +
		"delete-buffer) exit 0 ;;\n" +
		"send-keys) printf '\\n' >> \"" + paste + "\"; exit 0 ;;\n" +
		"-V) echo tmux 3.6; exit 0 ;;\n" +
		"*) exit 0 ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(dir, "tmux"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return paste
}

func promptHarness(t *testing.T) (*store.Store, *httptest.Server, store.Terminal) {
	t.Helper()
	t.Cleanup(resetPromptInFlight)
	cwd := t.TempDir()
	st := testStore(t)
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
	}).Handler)
	t.Cleanup(ts.Close)
	term, err := st.CreateTerminal("cli", cwd)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetTerminalLaunch(term.ID, "pi", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	return st, ts, term
}

func TestTerminalPromptDecisionTable(t *testing.T) {
	png := base64.StdEncoding.EncodeToString([]byte("PNG"))

	t.Run("missing terminal", func(t *testing.T) {
		_, ts, _ := promptHarness(t)
		code, _ := postRaw(t, ts, "/api/terminals/nope/drop", `{"name":"a.png","data":"`+png+`"}`)
		if code != http.StatusNotFound {
			t.Fatalf("code=%d", code)
		}
	})

	t.Run("plain shell refused", func(t *testing.T) {
		st, ts, _ := promptHarness(t)
		shell, err := st.CreateTerminal("sh", t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		code, page := postRaw(t, ts, "/api/terminals/"+shell.ID+"/drop", `{"name":"a.png","data":"`+png+`"}`)
		if code != http.StatusConflict || !strings.Contains(page["error"].(string), "Agent CLI") {
			t.Fatalf("%d %v", code, page)
		}
	})

	t.Run("drop writes under cwd", func(t *testing.T) {
		_, ts, term := promptHarness(t)
		code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/drop", `{"name":"shot.png","data":"`+png+`"}`)
		if code != http.StatusOK {
			t.Fatalf("%d %v", code, page)
		}
		rel, _ := page["path"].(string)
		abs := filepath.Join(term.Cwd, filepath.FromSlash(rel))
		raw, err := os.ReadFile(abs)
		if err != nil || string(raw) != "PNG" {
			t.Fatalf("file %v %q", err, raw)
		}
	})

	t.Run("drop too large", func(t *testing.T) {
		_, ts, term := promptHarness(t)
		big := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", maxDropBytes+1)))
		code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/drop", `{"name":"a.png","data":"`+big+`"}`)
		if code != http.StatusBadRequest || !strings.Contains(page["error"].(string), "4 MB") {
			t.Fatalf("%d %v", code, page)
		}
	})

	t.Run("path escape", func(t *testing.T) {
		_, ts, term := promptHarness(t)
		code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/prompt", `{"message":"x","paths":["../secret"]}`)
		if code != http.StatusBadRequest || !strings.Contains(page["error"].(string), "outside") {
			t.Fatalf("%d %v", code, page)
		}
	})

	t.Run("too many paths", func(t *testing.T) {
		_, ts, term := promptHarness(t)
		code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/prompt", `{"paths":["a","b","c","d","e"]}`)
		if code != http.StatusBadRequest || !strings.Contains(page["error"].(string), "at most 4") {
			t.Fatalf("%d %v", code, page)
		}
	})

	t.Run("pane dead", func(t *testing.T) {
		shimTmux(t)
		_, ts, term := promptHarness(t)
		if err := os.WriteFile(filepath.Join(term.Cwd, "a.png"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/prompt", `{"message":"hi","paths":["a.png"]}`)
		if code != http.StatusConflict {
			t.Fatalf("%d %v", code, page)
		}
	})

	t.Run("send pastes", func(t *testing.T) {
		paste := fakeTmuxBin(t)
		_, ts, term := promptHarness(t)
		if err := os.WriteFile(filepath.Join(term.Cwd, "shot.png"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/prompt", `{"message":"look","paths":["shot.png"]}`)
		if code != http.StatusOK || page["typed"] != true {
			t.Fatalf("%d %v", code, page)
		}
		got, err := os.ReadFile(paste)
		if err != nil || !strings.Contains(string(got), "@shot.png") || !strings.Contains(string(got), "look") {
			t.Fatalf("paste %q %v", got, err)
		}
	})

	t.Run("in-flight busy", func(t *testing.T) {
		_, ts, term := promptHarness(t)
		if !tryLockPrompt(term.ID) {
			t.Fatal("lock")
		}
		if err := os.WriteFile(filepath.Join(term.Cwd, "a.png"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		fakeTmuxBin(t)
		code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/prompt", `{"message":"x","paths":["a.png"]}`)
		if code != http.StatusConflict || page["reason"] != "busy" {
			t.Fatalf("%d %v", code, page)
		}
	})

	t.Run("inspector type refused", func(t *testing.T) {
		_, ts, term := promptHarness(t)
		code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/type", `{"text":"git status"}`)
		if code != http.StatusConflict || page["reason"] != "cli" {
			t.Fatalf("%d %v", code, page)
		}
	})
}
