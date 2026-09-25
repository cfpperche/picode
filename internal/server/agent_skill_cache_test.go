package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// The sweep keeps a copy a live agent names, one a restorable exit names,
// and a fresh one; it removes an old copy nobody names, and one only a
// forgotten exit named.
func TestSweepAgentSkillCache(t *testing.T) {
	st := testStore(t)
	data := t.TempDir()
	deps := Deps{Store: st, DataDir: data}
	root := filepath.Join(data, "skills", "cache")
	dig := func(c string) string { return strings.Repeat(c, 64) }
	mk := func(d string, old bool) string {
		dir := filepath.Join(root, d, "s")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if old {
			past := time.Now().Add(-2 * time.Hour)
			_ = os.Chtimes(filepath.Join(root, d), past, past)
		}
		return dir
	}
	live, restorable, forgotten, orphan, fresh := dig("a"), dig("b"), dig("c"), dig("d"), dig("e")
	liveDir := mk(live, true)
	restDir := mk(restorable, true)
	forgotDir := mk(forgotten, true)
	mk(orphan, true)
	mk(fresh, false)

	a, err := st.AddAgent(store.FreeWorkspaceID, "live", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetAgentSkills(a.ID, []store.AgentSkill{{Name: "s", Digest: live, Dir: liveDir}}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, digest, dir string
		forget            bool
	}{{"gone", restorable, restDir, false}, {"forgot", forgotten, forgotDir, true}} {
		b, err := st.AddAgent(store.FreeWorkspaceID, c.name, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.SetAgentSkills(b.ID, []store.AgentSkill{{Name: "s", Digest: c.digest, Dir: c.dir}}); err != nil {
			t.Fatal(err)
		}
		ex, err := st.RemoveAgentWithExit(b.ID, store.ExitInput{Origin: store.ExitFromAPI})
		if err != nil {
			t.Fatal(err)
		}
		if c.forget {
			if _, err := st.ForgetAgentExit(ex.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	n, err := sweepAgentSkillCache(deps, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("removed %d, want 2 (orphan, forgotten)", n)
	}
	for d, want := range map[string]bool{live: true, restorable: true, fresh: true, forgotten: false, orphan: false} {
		_, err := os.Stat(filepath.Join(root, d))
		if (err == nil) != want {
			t.Errorf("%s… kept=%v, want %v", d[:6], err == nil, want)
		}
	}
}
