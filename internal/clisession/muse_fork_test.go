package clisession

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// fakeMuseServe writes a `muse` that answers MSP the way 1.3.0 does:
// initialize, then (after `initialized`) session/fork with a notification
// first, which the client must skip. reply is the fork's result object.
func fakeMuseServe(t *testing.T, reply string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "muse")
	script := `#!/bin/sh
[ "$1" = serve ] || exit 9
read init
printf '%s\n' '{"id":1,"jsonrpc":"2.0","result":{"serverInfo":{"name":"muse","version":"1.3.0"}}}'
read initialized
case "$initialized" in *'"initialized"'*) ;; *) exit 3 ;; esac
read fork
case "$fork" in *'"session/fork"'*'"src-1"'*) ;; *) printf '%s\n' '{"id":2,"jsonrpc":"2.0","error":{"code":-32600,"message":"bad fork"}}'; exit 0 ;; esac
printf '%s\n' '{"jsonrpc":"2.0","method":"session/started","params":{}}'
printf '%s\n' '` + reply + `'
read eof
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func startWith(bin string) func(context.Context, ...string) *exec.Cmd {
	return func(ctx context.Context, args ...string) *exec.Cmd { return exec.CommandContext(ctx, bin, args...) }
}

func TestMuseForkSession(t *testing.T) {
	bin := fakeMuseServe(t, `{"id":2,"jsonrpc":"2.0","result":{"session":{"sessionId":"new-1","forkedFrom":{"sessionId":"src-1"}}}}`)
	got, err := MuseSource{}.ForkSession(context.Background(), Ref{ID: "src-1"}, startWith(bin))
	if err != nil {
		t.Fatal(err)
	}
	want := Fork{Args: []string{"resume", "new-1"}, ID: "new-1", ResumeArgs: []string{"resume", "new-1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fork = %+v", got)
	}

	// A copy that does not name this conversation as its source is refused.
	bin = fakeMuseServe(t, `{"id":2,"jsonrpc":"2.0","result":{"session":{"sessionId":"new-1","forkedFrom":{"sessionId":"other"}}}}`)
	if _, err := (MuseSource{}).ForkSession(context.Background(), Ref{ID: "src-1"}, startWith(bin)); err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("wrong source: %v", err)
	}
	// An error answer surfaces as Muse's refusal.
	if _, err := (MuseSource{}).ForkSession(context.Background(), Ref{ID: "src-2"}, startWith(fakeMuseServe(t, `{}`))); err == nil || !strings.Contains(err.Error(), "bad fork") {
		t.Fatalf("refusal: %v", err)
	}
	if _, err := (MuseSource{}).ForkSession(context.Background(), Ref{}, startWith(bin)); err == nil {
		t.Fatal("a fork without a session id must refuse")
	}
	if !CapabilitiesOf("muse").Fork {
		t.Fatal("muse advertises its fork")
	}
}

func TestUUIDv7Shape(t *testing.T) {
	id := uuidV7()
	if len(id) != 36 || id[14] != '7' || !strings.ContainsAny(id[19:20], "89ab") {
		t.Fatalf("uuidv7 = %s", id)
	}
}

// A running Muse TUI's session is not indexed yet: the candidates are the
// log directories touched since the TUI started, and session/read names
// each one's folder. The newest in the terminal's folder wins.
func TestMuseLiveSession(t *testing.T) {
	home := t.TempDir()
	day := filepath.Join(home, "sessions", "2026", "09", "24")
	for _, id := range []string{"other-folder", "mine", "too-old"} {
		if err := os.MkdirAll(filepath.Join(day, id), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	_ = os.Chtimes(filepath.Join(day, "too-old"), now.Add(-2*time.Hour), now.Add(-2*time.Hour))
	_ = os.Chtimes(filepath.Join(day, "mine"), now.Add(-time.Minute), now.Add(-time.Minute))
	_ = os.Chtimes(filepath.Join(day, "other-folder"), now, now)
	bin := filepath.Join(t.TempDir(), "muse")
	script := `#!/bin/sh
[ "$1" = serve ] || exit 9
read init
printf '%s\n' '{"id":1,"jsonrpc":"2.0","result":{"museHome":"` + home + `"}}'
read initialized
while read line; do
  id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9]*\).*/\1/p')
  case "$line" in
    *'"sessionId":"mine"'*) root='/work/proj' ;;
    *'"sessionId":"other-folder"'*) root='/work/elsewhere' ;;
    *'"sessionId":"too-old"'*) root='/work/proj' ;;
    *) root='' ;;
  esac
  sid=$(printf '%s' "$line" | sed -n 's/.*"sessionId":"\([^"]*\)".*/\1/p')
  printf '{"id":%s,"jsonrpc":"2.0","result":{"session":{"sessionId":"%s","workspaceRoot":"%s"}}}\n' "$id" "$sid" "$root"
done
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	id, resume, err := MuseSource{}.LiveSession(context.Background(), "/work/proj", now.Add(-10*time.Minute), nil, startWith(bin))
	if err != nil || id != "mine" || !reflect.DeepEqual(resume, []string{"resume", "mine"}) {
		t.Fatalf("live = %q %v %v", id, resume, err)
	}
	if id, _, _ := (MuseSource{}).LiveSession(context.Background(), "/work/none", now.Add(-10*time.Minute), nil, startWith(bin)); id != "" {
		t.Fatalf("no session in that folder, got %q", id)
	}
	// A newer session in the same folder that belongs elsewhere (a fork's
	// copy, another terminal's pin) is skipped.
	if err := os.MkdirAll(filepath.Join(day, "a-fork-copy"), 0o755); err != nil {
		t.Fatal(err)
	}
	script2 := strings.Replace(script, `*'"sessionId":"mine"'*) root='/work/proj' ;;`, `*'"sessionId":"mine"'*) root='/work/proj' ;;
    *'"sessionId":"a-fork-copy"'*) root='/work/proj' ;;`, 1)
	if err := os.WriteFile(bin, []byte(script2), 0o755); err != nil {
		t.Fatal(err)
	}
	if id, _, _ := (MuseSource{}).LiveSession(context.Background(), "/work/proj", now.Add(-10*time.Minute), nil, startWith(bin)); id != "a-fork-copy" {
		t.Fatalf("without exclusions the newest wins, got %q", id)
	}
	taken := func(id string) bool { return id == "a-fork-copy" }
	if id, _, _ := (MuseSource{}).LiveSession(context.Background(), "/work/proj", now.Add(-10*time.Minute), taken, startWith(bin)); id != "mine" {
		t.Fatalf("taken sessions are skipped, got %q", id)
	}
}
