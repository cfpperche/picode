package skills

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
)

// TestLiveMuseParity runs the real muse binary against a fixture and checks
// that the reader's loaded skills are the ones Muse says it loads, from the
// same folder (ADR-0196 slice 1's live gate). Gated like the other live
// suites: PICODE_SKILLS_LIVE=1 and PICODE_LIVE_SANDBOX set to a throwaway
// HOME, so nothing here reads or writes the owner's own.
func TestLiveMuseParity(t *testing.T) {
	if os.Getenv("PICODE_SKILLS_LIVE") != "1" {
		t.Skip("set PICODE_SKILLS_LIVE=1")
	}
	homeDir := os.Getenv("HOME")
	if sb := os.Getenv("PICODE_LIVE_SANDBOX"); sb == "" || sb != homeDir {
		t.Skip("PICODE_LIVE_SANDBOX must equal HOME")
	}
	muse, err := exec.LookPath("muse")
	if err != nil {
		t.Skip("muse not installed")
	}
	_, _, wsDir := fixture(t)
	writeSkill(t, filepath.Join(wsDir, ".agents/skills"), "both", "")
	writeSkill(t, filepath.Join(wsDir, ".claude/skills"), "both", "")
	writeSkill(t, filepath.Join(wsDir, ".claude/skills"), "claude-proj", "")
	writeSkill(t, filepath.Join(homeDir, ".agents/skills"), "both", "")
	writeSkill(t, filepath.Join(homeDir, ".agents/skills"), "user-agents", "")
	writeSkill(t, filepath.Join(homeDir, ".claude/skills"), "user-agents", "")
	writeSkill(t, filepath.Join(homeDir, ".claude/skills"), "user-claude", "")

	out, err := exec.Command(muse, "skills", "list", "--source", "all", "--workspace", wsDir, "--trust-workspace", "--json").Output()
	if err != nil {
		t.Fatalf("muse skills list: %v", err)
	}
	var roster struct {
		Skills []struct {
			Name  string `json:"name"`
			Scope string `json:"scope"`
			Path  string `json:"path"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(out, &roster); err != nil {
		t.Fatalf("muse output: %v\n%s", err, out)
	}
	var vendor []string
	for _, s := range roster.Skills {
		if s.Scope == "user" || s.Scope == "project" {
			vendor = append(vendor, s.Name+"@"+filepath.Dir(filepath.Dir(s.Path)))
		}
	}
	rep, err := Read(Query{CLI: "muse", Workspace: wsDir, Home: homeDir})
	if err != nil {
		t.Fatal(err)
	}
	var ours []string
	for _, r := range rep.Rows {
		if r.Status == StatusLoaded || r.Status == StatusIfTrusted {
			root := r.Root
			if r.Scope == Machine {
				root = "$HOME/" + root[2:]
			}
			ours = append(ours, r.Name+"@"+root)
		}
	}
	sort.Strings(vendor)
	sort.Strings(ours)
	if len(vendor) != 4 {
		t.Fatalf("muse listed %v; the fixture has four loadable names", vendor)
	}
	t.Logf("muse and the reader agree: %v", ours)
	if len(vendor) != len(ours) {
		t.Fatalf("muse loads %v\nreader says %v", vendor, ours)
	}
	for i := range vendor {
		if vendor[i] != ours[i] {
			t.Fatalf("muse loads %v\nreader says %v", vendor, ours)
		}
	}
}

// TestLiveSkillsCLIHash installs a fixture with the real `npx skills` (no
// telemetry, sandbox HOME) and requires its computedHash to equal Digest —
// ADR-0196 slice 2's interop gate. Needs npx and the network.
func TestLiveSkillsCLIHash(t *testing.T) {
	if os.Getenv("PICODE_SKILLS_LIVE") != "1" {
		t.Skip("set PICODE_SKILLS_LIVE=1")
	}
	homeDir := os.Getenv("HOME")
	if sb := os.Getenv("PICODE_LIVE_SANDBOX"); sb == "" || sb != homeDir {
		t.Skip("PICODE_LIVE_SANDBOX must equal HOME")
	}
	npx, err := exec.LookPath("npx")
	if err != nil {
		t.Skip("npx not installed")
	}
	src := filepath.Join(t.TempDir(), "interop-probe")
	for p, body := range map[string]string{
		"SKILL.md": "---\nname: interop-probe\ndescription: hashes the same\n---\n", "scripts/run.sh": "echo hi\n",
		"references/A.md": "A", "references/a.md": "a", "README.md": "x", "assets/x-y_z.1.txt": "1",
	} {
		writeFile(t, filepath.Join(src, p), body)
	}
	proj := t.TempDir()
	cmd := exec.Command(npx, "-y", "skills@latest", "add", src, "-a", "codex", "-y", "--copy")
	cmd.Dir = proj
	cmd.Env = append(os.Environ(), "DISABLE_TELEMETRY=1", "DO_NOT_TRACK=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("npx skills add: %v\n%s", err, out)
	}
	var lock struct {
		Skills map[string]struct {
			ComputedHash string `json:"computedHash"`
		} `json:"skills"`
	}
	b, err := os.ReadFile(filepath.Join(proj, "skills-lock.json"))
	if err != nil || json.Unmarshal(b, &lock) != nil {
		t.Fatalf("lock: %v %s", err, b)
	}
	got, err := Digest(src)
	if err != nil {
		t.Fatal(err)
	}
	if want := lock.Skills["interop-probe"].ComputedHash; want == "" || got != want {
		t.Fatalf("Digest %s, skills CLI %s", got, want)
	}
}
