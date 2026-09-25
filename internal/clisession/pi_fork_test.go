package clisession

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakePi stands in for pi's --fork: it writes <dir>/<time>_<id>.jsonl the
// way pi 0.87.1 does (measured 2026-09-24), records its arguments and its
// folder, or fails when told to.
func fakePi(t *testing.T, mode string) (func(ctx context.Context, args ...string) *exec.Cmd, string) {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "args")
	script := filepath.Join(dir, "pi")
	body := `#!/bin/sh
printf '%s\n' "$PWD" "$@" > "` + log + `"
case "` + mode + `" in
fail) echo "Error: No session found" >&2; exit 1 ;;
silent) exit 0 ;;
esac
id= ; sdir=
while [ $# -gt 0 ]; do
  case "$1" in
  --session-id) id=$2; shift ;;
  --session-dir) sdir=$2; shift ;;
  esac
  shift
done
printf '{"type":"session","id":"%s"}\n' "$id" > "$sdir/2026-09-25T01-24-00-992Z_$id.jsonl"
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	return func(ctx context.Context, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, script, args...)
		cmd.Dir = cwd
		return cmd
	}, log
}

func TestPiForkIntoDir(t *testing.T) {
	src := filepath.Join(t.TempDir(), "2026-09-15T19-20-42-681Z_src.jsonl")
	if err := os.WriteFile(src, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "agent-dir")
	start, log := fakePi(t, "ok")
	got, err := PISource{}.ForkIntoDir(context.Background(), Ref{Path: src}, dir, "new-id", start)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(got) != dir || !strings.HasSuffix(got, "_new-id.jsonl") {
		t.Fatalf("copy at %s", got)
	}
	raw, _ := os.ReadFile(log)
	args := strings.Join(strings.Split(strings.TrimSpace(string(raw)), "\n")[1:], " ")
	want := "--mode rpc --no-extensions --no-skills --no-prompt-templates --no-themes --fork " + src + " --session-id new-id --session-dir " + dir
	if args != want {
		t.Fatalf("args %q\nwant %q", args, want)
	}

	// Refusals name the reason: pi's own error, a run that wrote nothing,
	// a source that is gone, a ref with no file.
	for _, c := range []struct {
		mode, src, want string
	}{
		{"fail", src, "No session found"},
		{"silent", src, "wrote no copy"},
		{"ok", filepath.Join(t.TempDir(), "gone.jsonl"), "is gone"},
		{"ok", "", "no conversation file"},
	} {
		start, _ := fakePi(t, c.mode)
		_, err := PISource{}.ForkIntoDir(context.Background(), Ref{Path: c.src}, filepath.Join(t.TempDir(), "d"), "x", start)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: %v, want %q", c.mode, err, c.want)
		}
	}
}
