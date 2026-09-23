package skills

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func writeSkill(t *testing.T, dir, name, header string) string {
	t.Helper()
	d := filepath.Join(dir, name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if header == "" {
		header = "---\nname: " + name + "\ndescription: does " + name + "\n---\nbody\n"
	}
	if err := os.WriteFile(filepath.Join(d, "SKILL.md"), []byte(header), 0o644); err != nil {
		t.Fatal(err)
	}
	return d
}

func writeFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixture: a HOME and a workspace inside a repository.
func fixture(t *testing.T) (homeDir, repo, wsDir string) {
	t.Helper()
	root := t.TempDir()
	homeDir = filepath.Join(root, "home")
	repo = filepath.Join(root, "repo")
	wsDir = filepath.Join(repo, "pkg", "app")
	for _, d := range []string{homeDir, filepath.Join(repo, ".git"), wsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return
}

func rowsBy(rep Report) map[string][]Row {
	m := map[string][]Row{}
	for _, r := range rep.Rows {
		m[r.Name] = append(m[r.Name], r)
	}
	return m
}

func TestPrecedenceAndShadowing(t *testing.T) {
	homeDir, _, wsDir := fixture(t)
	// Muse (measured): project .agents > project .claude > user .agents > user .claude.
	writeSkill(t, filepath.Join(wsDir, ".agents/skills"), "shared", "")
	writeSkill(t, filepath.Join(wsDir, ".claude/skills"), "shared", "")
	writeSkill(t, filepath.Join(homeDir, ".agents/skills"), "shared", "")
	writeSkill(t, filepath.Join(homeDir, ".claude/skills"), "only-claude", "")
	rep, err := Read(Query{CLI: "muse", Workspace: wsDir, Home: homeDir})
	if err != nil {
		t.Fatal(err)
	}
	got := rowsBy(rep)["shared"]
	if len(got) != 3 {
		t.Fatalf("shared rows = %d, want 3", len(got))
	}
	// Muse's project rows need trust PiCode cannot read.
	if got[0].Root != ".agents/skills" || got[0].Status != StatusIfTrusted {
		t.Fatalf("winner = %+v", got[0])
	}
	for _, r := range got[1:] {
		if r.Status != StatusShadowed || r.ShadowedBy != ".agents/skills" {
			t.Fatalf("loser = %+v", r)
		}
	}
	if r := rowsBy(rep)["only-claude"][0]; r.Status != StatusLoaded || r.Scope != Machine {
		t.Fatalf("only-claude = %+v", r)
	}
}

func TestClaudePersonalOutranksProject(t *testing.T) {
	homeDir, _, wsDir := fixture(t)
	writeSkill(t, filepath.Join(wsDir, ".claude/skills"), "deploy", "")
	writeSkill(t, filepath.Join(homeDir, ".claude/skills"), "deploy", "")
	rep, _ := Read(Query{CLI: "claude-code", Workspace: wsDir, Home: homeDir})
	rows := rowsBy(rep)["deploy"]
	if rows[0].Scope != Machine || rows[0].Status != StatusLoaded || rows[1].Status != StatusShadowed {
		t.Fatalf("rows = %+v", rows)
	}
	// Claude Code does not read .agents/skills.
	writeSkill(t, filepath.Join(wsDir, ".agents/skills"), "agents-only", "")
	rep, _ = Read(Query{CLI: "claude-code", Workspace: wsDir, Home: homeDir})
	if len(rowsBy(rep)["agents-only"]) != 0 {
		t.Fatal("claude-code must not read .agents/skills")
	}
}

func TestWalkUpStopsAtTheRepositoryRoot(t *testing.T) {
	homeDir, repo, wsDir := fixture(t)
	writeSkill(t, filepath.Join(repo, ".agents/skills"), "at-root", "")
	writeSkill(t, filepath.Join(filepath.Dir(repo), ".agents/skills"), "outside-repo", "")
	rep, _ := Read(Query{CLI: "codex", Workspace: wsDir, Home: homeDir})
	by := rowsBy(rep)
	if len(by["at-root"]) != 1 || by["at-root"][0].Root != "../../.agents/skills" {
		t.Fatalf("at-root = %+v", by["at-root"])
	}
	if len(by["outside-repo"]) != 0 {
		t.Fatal("codex walked past the repository root")
	}
	// agy does not walk: the repository root's folder is not its folder.
	rep, _ = Read(Query{CLI: "agy", Workspace: wsDir, Home: homeDir})
	if len(rowsBy(rep)["at-root"]) != 0 {
		t.Fatal("agy must read only the workspace's own folder")
	}
}

func TestTrustStatus(t *testing.T) {
	homeDir, _, wsDir := fixture(t)
	writeSkill(t, filepath.Join(wsDir, ".agents/skills"), "proj", "")
	no := false
	rep, _ := Read(Query{CLI: "pi", Workspace: wsDir, Home: homeDir, Trusted: func(string, string) *bool { return &no }})
	if r := rowsBy(rep)["proj"][0]; r.Status != StatusNeedsTrust || !rep.Trust.Needed || rep.Trust.Command != "/trust" {
		t.Fatalf("pi untrusted: %+v %+v", r, rep.Trust)
	}
	yes := true
	rep, _ = Read(Query{CLI: "pi", Workspace: wsDir, Home: homeDir, Trusted: func(string, string) *bool { return &yes }})
	if r := rowsBy(rep)["proj"][0]; r.Status != StatusLoaded {
		t.Fatalf("pi trusted: %+v", r)
	}
	rep, _ = Read(Query{CLI: "hermes", Workspace: wsDir, Home: homeDir})
	if r := rowsBy(rep)["proj"][0]; r.Status != StatusIfTrusted || rep.Trust.Command != "hermes skills trust" {
		t.Fatalf("hermes: %+v %+v", r, rep.Trust)
	}
	// Codex has no trust gate on skills.
	rep, _ = Read(Query{CLI: "codex", Workspace: wsDir, Home: homeDir})
	if r := rowsBy(rep)["proj"][0]; r.Status != StatusLoaded || rep.Trust.Needed {
		t.Fatalf("codex: %+v", r)
	}
}

func TestSpecProblems(t *testing.T) {
	homeDir, _, _ := fixture(t)
	dir := filepath.Join(homeDir, ".agents/skills")
	writeSkill(t, dir, "good", "")
	writeSkill(t, dir, "mismatch", "---\nname: other\ndescription: x\n---\n")
	writeSkill(t, dir, "Bad_Name", "---\nname: Bad_Name\ndescription: x\n---\n")
	writeSkill(t, dir, "nodesc", "---\nname: nodesc\n---\n")
	writeSkill(t, dir, "noheader", "# just markdown\n")
	writeSkill(t, dir, "loose", "---\nname: loose\ndescription: Use when: \"quoted\" things: happen\n---\n")
	rep, _ := Read(Query{CLI: "codex", Home: homeDir})
	cases := []struct {
		name, status, problem string
	}{
		{"good", StatusLoaded, ""},
		{"other", StatusLoaded, "does not match its folder"},
		{"Bad_Name", StatusLoaded, "lowercase"},
		{"nodesc", StatusInvalid, "no description"},
		{"noheader", StatusInvalid, "no YAML frontmatter"},
		{"loose", StatusLoaded, "formatting problem"},
	}
	by := rowsBy(rep)
	for _, c := range cases {
		rows := by[c.name]
		if len(rows) != 1 {
			t.Fatalf("%s: %d rows", c.name, len(rows))
		}
		r := rows[0]
		if r.Status != c.status {
			t.Errorf("%s: status %s, want %s", c.name, r.Status, c.status)
		}
		joined := strings.Join(r.Problems, "; ")
		if (c.problem == "") != (joined == "") || !strings.Contains(joined, c.problem) {
			t.Errorf("%s: problems %q, want %q", c.name, joined, c.problem)
		}
	}
	if by["loose"][0].Description != `Use when: "quoted" things: happen` {
		t.Errorf("loose description = %q", by["loose"][0].Description)
	}
}

func TestProvenanceFromLocks(t *testing.T) {
	homeDir, _, wsDir := fixture(t)
	projDir := writeSkill(t, filepath.Join(wsDir, ".agents/skills"), "pdf", "")
	writeSkill(t, filepath.Join(wsDir, ".agents/skills"), "edited", "")
	pdfHash, err := Digest(projDir)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(wsDir, "skills-lock.json"), `{"version":1,"skills":{
		"pdf":{"source":"anthropics/skills","sourceType":"github","computedHash":"`+pdfHash+`"},
		"edited":{"source":"acme/skills","sourceType":"github","computedHash":"0000"}}}`)
	writeSkill(t, filepath.Join(homeDir, ".agents/skills"), "hyperframes", "")
	writeFile(t, filepath.Join(homeDir, ".agents/.skill-lock.json"), `{"version":3,"skills":{
		"hyperframes":{"source":"heygen-com/hyperframes","sourceType":"github","sourceUrl":"https://github.com/heygen-com/hyperframes.git","skillFolderHash":"abc"}}}`)
	writeSkill(t, filepath.Join(homeDir, ".hermes/skills/mlops"), "llama-cpp", "")
	writeFile(t, filepath.Join(homeDir, ".hermes/skills/.hub/lock.json"), `{"version":1,"installed":{
		"llama-cpp":{"source":"official","identifier":"official/mlops/llama-cpp","trust_level":"builtin","content_hash":"sha256:43","install_path":"mlops/llama-cpp"}}}`)

	rep, _ := Read(Query{CLI: "codex", Workspace: wsDir, Home: homeDir})
	by := rowsBy(rep)
	if p := by["pdf"][0].Provenance; p == nil || p.Source != "anthropics/skills" || p.Modified {
		t.Fatalf("pdf provenance = %+v", p)
	}
	if p := by["edited"][0].Provenance; p == nil || !p.Modified {
		t.Fatalf("edited provenance = %+v", p)
	}
	if p := by["hyperframes"][0].Provenance; p == nil || p.Lock != "~/.agents/.skill-lock.json" || p.Source != "heygen-com/hyperframes" {
		t.Fatalf("hyperframes provenance = %+v", p)
	}
	rep, _ = Read(Query{CLI: "hermes", Home: homeDir})
	if p := rowsBy(rep)["llama-cpp"][0].Provenance; p == nil || p.Installer != "hermes" || p.Trust != "builtin" {
		t.Fatalf("hermes provenance = %+v", p)
	}
	// A broken lock is a note, never an empty report.
	writeFile(t, filepath.Join(wsDir, "skills-lock.json"), `{not json`)
	rep, _ = Read(Query{CLI: "codex", Workspace: wsDir, Home: homeDir})
	if len(rep.Notes) == 0 || len(rep.Rows) == 0 {
		t.Fatalf("broken lock: notes=%v rows=%d", rep.Notes, len(rep.Rows))
	}
}

func TestAlsoLoadedByAndLinks(t *testing.T) {
	homeDir, _, wsDir := fixture(t)
	canonical := writeSkill(t, filepath.Join(wsDir, ".agents/skills"), "shared", "")
	if err := os.MkdirAll(filepath.Join(wsDir, ".claude/skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(canonical, filepath.Join(wsDir, ".claude/skills/shared")); err != nil {
		t.Skip("no symlinks here")
	}
	rep, _ := Read(Query{CLI: "codex", Workspace: wsDir, Home: homeDir})
	also := rowsBy(rep)["shared"][0].AlsoLoadedBy
	sort.Strings(also)
	want := []string{"agy", "grok", "hermes", "muse", "omp", "opencode", "pi"}
	if !reflect.DeepEqual(also, want) {
		t.Fatalf("also loaded by = %v, want %v", also, want)
	}
	rep, _ = Read(Query{CLI: "claude-code", Workspace: wsDir, Home: homeDir})
	if r := rowsBy(rep)["shared"][0]; !r.Linked || r.Status != StatusLoaded {
		t.Fatalf("linked row = %+v", r)
	}
}

func TestRecursiveRoots(t *testing.T) {
	homeDir, _, _ := fixture(t)
	writeSkill(t, filepath.Join(homeDir, ".hermes/skills/devops/ci"), "deep", "")
	writeSkill(t, filepath.Join(homeDir, ".agents/skills/group"), "nested", "")
	rep, _ := Read(Query{CLI: "hermes", Home: homeDir})
	if len(rowsBy(rep)["deep"]) != 1 {
		t.Fatal("hermes reads its category folders")
	}
	rep, _ = Read(Query{CLI: "codex", Home: homeDir})
	if len(rowsBy(rep)["nested"]) != 0 {
		t.Fatal("codex reads <root>/<name>/SKILL.md only")
	}
}

func TestUnknownCLI(t *testing.T) {
	if _, err := Read(Query{CLI: "nope"}); err != ErrUnknownCLI {
		t.Fatalf("err = %v", err)
	}
}

func TestEveryCatalogCLIHasADeclaration(t *testing.T) {
	want := []string{"pi", "omp", "claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy"}
	if got := CLIs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("CLIs = %v", got)
	}
}

// TestLocaleOrderMatchesNode pins localeLess to V8's localeCompare, the
// order Vercel's computedHash uses (skipped where node is absent).
func TestLocaleOrderMatchesNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	in := []string{"SKILL.md", "scripts/x.py", "references/A.md", "references/a.md", "_x", "-y", "a-b", "a_b",
		"a.b", "a/b", "A", "b", "B1", "b0", "README.md", "assets/logo.png", "LICENSE.txt", "scripts/run.sh",
		"references/api-v2.md", "references/api_v2.md", "references/api.md", "x10", "x9", "Z", "z", "a b"}
	out, err := exec.Command(node, "-e", `const a=JSON.parse(process.argv[1]);console.log(JSON.stringify(a.sort((x,y)=>x.localeCompare(y))))`, mustJSON(in)).Output()
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(out))
	got := append([]string(nil), in...)
	sort.SliceStable(got, func(i, j int) bool { return localeLess(got[i], got[j]) })
	if mustJSON(got) != want {
		t.Fatalf("order\n got %s\nwant %s", mustJSON(got), want)
	}
}

func mustJSON(v []string) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, s := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`"` + strings.ReplaceAll(s, `"`, `\"`) + `"`)
	}
	b.WriteByte(']')
	return b.String()
}

// TestDigestMatchesTheSkillsCLI reproduces computeSkillFolderHash with Node's
// crypto over the same files (skipped where node is absent).
func TestDigestMatchesTheSkillsCLI(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	dir := t.TempDir()
	for p, body := range map[string]string{
		"SKILL.md": "---\nname: x\ndescription: y\n---\n", "scripts/run.sh": "echo hi\n",
		"references/A.md": "A", "references/a.md": "a", "assets/logo.png": "\x89PNG",
		"node_modules/skip/index.js": "skip", ".git/HEAD": "ref",
	} {
		writeFile(t, filepath.Join(dir, p), body)
	}
	js := `const fs=require('fs'),path=require('path'),c=require('crypto');const base=process.argv[1];const files=[];
function walk(d){for(const e of fs.readdirSync(d,{withFileTypes:true})){const f=path.join(d,e.name);if(e.isDirectory()){if(e.name==='.git'||e.name==='node_modules')continue;walk(f)}else if(e.isFile()){files.push({r:path.relative(base,f).split('\\').join('/'),c:fs.readFileSync(f)})}}}
walk(base);files.sort((a,b)=>a.r.localeCompare(b.r));const h=c.createHash('sha256');for(const f of files){h.update(f.r);h.update(f.c)}console.log(h.digest('hex'))`
	out, err := exec.Command(node, "-e", js, dir).Output()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Digest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.TrimSpace(string(out)); got != want {
		t.Fatalf("digest %s, want %s", got, want)
	}
}

// TestSkillsJSListMatchesTheDeclarations keeps the pane's list of CLIs and
// the Go declarations one list (web/shared/domain/cliSkills.js).
func TestSkillsJSListMatchesTheDeclarations(t *testing.T) {
	body, err := os.ReadFile("../../web/shared/domain/cliSkills.js")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`SKILLS_CLIS = \[([^\]]*)\]`).FindSubmatch(body)
	if m == nil {
		t.Fatal("SKILLS_CLIS not found in cliSkills.js")
	}
	var js []string
	for _, part := range strings.Split(string(m[1]), ",") {
		if id := strings.Trim(strings.TrimSpace(part), `"`); id != "" {
			js = append(js, id)
		}
	}
	if !reflect.DeepEqual(js, CLIs()) {
		t.Fatalf("js %v, go %v", js, CLIs())
	}
}
