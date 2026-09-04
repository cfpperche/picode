package install

import (
	"path/filepath"
	"strings"
	"testing"
)

func env(pairs map[string]string) func(string) string {
	return func(k string) string { return pairs[k] }
}

func existing(paths ...string) func(string) bool {
	set := map[string]bool{}
	for _, p := range paths {
		set[p] = true
	}
	return func(p string) bool { return set[p] }
}

func TestSessionEnvFillsOnlyWhatIsMissing(t *testing.T) {
	dir := "/run/user/1000"
	bus := filepath.Join(dir, "bus")

	// A login shell already has both: add nothing.
	got := sessionEnv(env(map[string]string{
		runtimeDirEnv: dir, busEnv: "unix:path=" + bus,
	}), 1000, existing(dir, bus))
	if len(got) != 0 {
		t.Fatalf("nothing to fill in, got %v", got)
	}

	// A bare shell gets both, pointed at its own uid.
	got = sessionEnv(env(nil), 1000, existing(dir, bus))
	want := []string{runtimeDirEnv + "=" + dir, busEnv + "=unix:path=" + bus}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("got %v, want %v", got, want)
	}

	// The runtime dir is there but the socket is not: point at the dir only,
	// rather than at a bus that does not exist.
	got = sessionEnv(env(nil), 1000, existing(dir))
	if len(got) != 1 || got[0] != runtimeDirEnv+"="+dir {
		t.Fatalf("got %v", got)
	}
}

// The uid is the caller's own, so the guess can only ever land on its own
// runtime directory — never on another account's socket.
func TestSessionEnvUsesTheCallersOwnUid(t *testing.T) {
	got := sessionEnv(env(nil), 4242, existing("/run/user/4242", "/run/user/4242/bus"))
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "/run/user/4242") {
		t.Fatalf("got %v", got)
	}
	if strings.Contains(joined, "/run/user/1000") {
		t.Fatalf("reached for another uid: %v", got)
	}
}

func TestSessionEnvGivesUpWhenThereIsNothingToPointAt(t *testing.T) {
	if got := sessionEnv(env(nil), 1000, existing()); got != nil {
		t.Fatalf("nothing exists; got %v", got)
	}
}

func TestUserSessionMessageNamesTheFix(t *testing.T) {
	// Present in the environment: nothing to check.
	if err := userSession(env(map[string]string{runtimeDirEnv: "/run/user/1000"}), 1000, existing(), "goat"); err != nil {
		t.Fatalf("already set, got %v", err)
	}
	// Absent but the directory is there: sessionEnv will fill it in.
	if err := userSession(env(nil), 1000, existing("/run/user/1000"), "goat"); err != nil {
		t.Fatalf("recoverable, got %v", err)
	}
	// Neither: the message has to say what to do, not just what failed.
	err := userSession(env(nil), 1000, existing(), "goat")
	if err == nil {
		t.Fatal("expected a refusal")
	}
	for _, want := range []string{"loginctl enable-linger goat", "/run/user/1000", runtimeDirEnv} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("message must contain %q:\n%s", want, err)
		}
	}
}
