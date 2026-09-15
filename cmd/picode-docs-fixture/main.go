// picode-docs-fixture boots the real server against a seeded, synthetic
// store on 127.0.0.1:8490, so the docs capture pipeline
// (scripts/docs-shots.mjs) photographs the current UI — never stale,
// hand-placed images (docs/benchmarks/2026-09-03-docs-harness.md,
// parity principle). Ungated (Auth nil = the tests/dev mode) and plain
// HTTP: localhost is a secure context, so the UI is fully functional.
//
// Not shipped in releases (the release workflow builds ./cmd/picode and
// ./cmd/picode-desktop only). Run from the repo root after `make web`:
//
//	go run ./cmd/picode-docs-fixture
package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cfpperche/picode/internal/apps"
	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/llamaservice"
	"github.com/cfpperche/picode/internal/presence"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/server"
	"github.com/cfpperche/picode/internal/share"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

var dataDir string

func main() {
	llamaservice.RunSupervisor()
	addr := flag.String("addr", "127.0.0.1:18740", "fixture listen address; use a separate port for concurrent worktrees")
	flag.Parse()
	// One directory per listen address, so the path the UI shows (the
	// inspector prints the workspace) is identical in every capture — a
	// random MkdirTemp suffix made each recapture a new PNG (2026-09-09).
	// Concurrent worktrees run on separate ports and never share it, and the
	// previous run's directory on this port is disposable: the port is
	// exclusive and `make docs-shots` already kills whatever holds it.
	dataDir = filepath.Join(os.TempDir(), "picode-docs-fixture-"+portSuffix(*addr))
	if err := os.RemoveAll(dataDir); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dataDir)
	// Synthetic HOME before anything resolves a path: the app reads real pi
	// sessions (~/.pi/agent/sessions) for usage stats, and the fixture must
	// never publish the user's real spend — parity means synthetic data only.
	if err := os.Setenv("HOME", filepath.Join(dataDir, "home")); err != nil {
		log.Fatalf("fixture: home: %v", err)
	}
	// Every run seeds the same synthetic world.
	for _, d := range []string{
		dataDir,
		filepath.Join(dataDir, "home", ".pi", "agent"),
		filepath.Join(dataDir, "work", "picode"),
		filepath.Join(dataDir, "work", "website"),
		filepath.Join(dataDir, "work", "sandbox"),
		filepath.Join(dataDir, "work", "fresh"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Fatalf("fixture: mkdir %s: %v", d, err)
		}
	}

	// The picode workspace is a small git repository with a commit behind it
	// and a dirty working tree, so the Inspector rail's Changes list (counts,
	// branch, totals) photographs real data. Without git on PATH the folder
	// stays plain and the rail shows its non-git state instead.
	seedRepo(filepath.Join(dataDir, "work", "picode"))

	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		log.Fatalf("fixture: store: %v", err)
	}
	defer st.Close()
	runtimes := server.NewTermRuntimes()
	if err := seed(st, runtimes); err != nil {
		log.Fatalf("fixture: seed: %v", err)
	}
	runtime := rpc.NewRuntime("pi", st, nil) // never started: no real spawns
	runtime.DataDir = dataDir

	// Same feed/presence wiring as cmd/picode (ADR-0048), so fleet pills
	// and badges render exactly like production.
	changes := &feed.Feed{Store: st}
	st.OnEvent = changes.Publish
	runtime.OnState = func(agentID string, streaming, waiting bool, dialog *rpc.UIDialog) {
		changes.Ephemeral("agent.state", map[string]any{"agentId": agentID, "streaming": streaming, "waiting": waiting, "dialog": dialog})
	}
	devices := presence.New(share.ReachableIPv4())
	devices.OnChange = func(d presence.Device) {
		if d.Online {
			changes.Ephemeral("device.online", d)
		}
	}

	// A dedicated socket per fixture instance (ADR-0139): capture sessions
	// land in the fixture's own server instead of the owner's tmux, and the
	// default-socket Manager rides along as the drain.
	fxTmux := tmux.NewWithSocket(filepath.Join(dataDir, "tmux.sock")).WithLegacy(tmux.New())
	deps := server.Deps{
		Store:        st,
		Auth:         nil, // ungated: the capture browser walks straight in
		Tmux:         fxTmux,
		Runtime:      runtime,
		AgentCmd:     "pi", // ADR-0003; never spawned by the fixture
		DataDir:      dataDir,
		Presence:     devices,
		Feed:         changes,
		TermRuntimes: runtimes,
		// The mobile inbox screen is an apps-host surface (ADR-0036):
		// without the registry its iframe dies and the screen shows
		// "Reconnecting" forever — the capture gate caught exactly that.
		Apps: apps.NewRegistry(apps.BuiltIns(false)...),
	}

	srv := server.New(*addr, deps)
	log.Printf("fixture: synthetic PiCode on http://%s (data: %s)", *addr, dataDir)
	if err := http.ListenAndServe(*addr, srv.Handler); err != nil {
		log.Printf("fixture: %v", err)
	}
}

// seed fills the store with a fixed, synthetic world. Names, states and
// numbers are invented; nothing here is a real person, inbox or spend.
func seed(st *store.Store, runtimes *server.TermRuntimes) error {
	wsMain, err := st.AddWorkspace("picode", filepath.Join(dataDir, "work", "picode"))
	if err != nil {
		return err
	}
	wsSite, err := st.AddWorkspace("website", filepath.Join(dataDir, "work", "website"))
	if err != nil {
		return err
	}
	wsSandbox, err := st.AddWorkspace("sandbox", filepath.Join(dataDir, "work", "sandbox"))
	if err != nil {
		return err
	}
	// fresh: no agents, no terminals — the sidebar's real empty state.
	if _, err := st.AddWorkspace("fresh", filepath.Join(dataDir, "work", "fresh")); err != nil {
		return err
	}

	atlasID := ""
	for _, a := range []struct{ ws, name, path string }{
		{wsMain.ID, "Atlas", filepath.Join(dataDir, "work", "picode")},
		{wsMain.ID, "Borealis", ""},
		{wsSite.ID, "Kepler", filepath.Join(dataDir, "work", "website")},
	} {
		created, err := st.AddAgent(a.ws, a.name, a.path)
		if err != nil {
			return err
		}
		if a.name == "Atlas" {
			atlasID = created.ID
		}
	}
	// Free agents (no workspace — the sidebar's ungrouped section).
	for _, name := range []string{"Nova", "Rigel"} {
		if _, err := st.AddAgent(store.FreeWorkspaceID, name, ""); err != nil {
			return err
		}
	}

	_, err = st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: atlasID,
		WorkspaceID: wsMain.ID,
		Reason:      "go.mod floor vs CI toolchain",
		Title:       "Bump the Go toolchain?",
		Body:        "go.mod pins 1.22 but CI builds with stable. Move the floor to 1.23 and drop the workaround comment?",
		Allowed:     []string{store.VerbAccept, store.VerbIgnore},
	})
	if err != nil {
		return err
	}
	_, err = st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxFYI, SourceKind: store.InboxFromAutomation, SourceID: "seed-auto",
		WorkspaceID: wsSite.ID,
		Reason:      "scheduled run finished",
		Title:       "Nightly link check finished",
		Body:        "42 pages checked, 0 dead links. Full log in the run detail.",
	})
	if err != nil {
		return err
	}

	if _, _, err := st.CreateAutomation(store.AutomationParams{
		Name: "Nightly link check", Action: "start", WorkspaceID: wsMain.ID,
		Prompt: "Check every page for dead links and report.",
		Cron:   "0 4 * * *", MaxCostUSD: 0.5,
	}); err != nil {
		return err
	}
	if _, _, err := st.CreateAutomation(store.AutomationParams{
		Name: "Changelog digest", Action: "start", WorkspaceID: wsSite.ID,
		Prompt:  "Summarize this week's commits into changelog candidates.",
		Webhook: true, MaxCostUSD: 0.25,
	}); err != nil {
		return err
	}

	// Project terminals (ADR-0026): plain shells plus agent-CLI terminals
	// with a live runtime lease (ADR-0062), so the sidebar's terminal rows
	// and the collapsed face strips photograph real states. No process is
	// spawned — the lease only proves presence to the live list.
	if _, err := st.CreateTerminalIn(wsMain.ID, "shell", filepath.Join(dataDir, "work", "picode")); err != nil {
		return err
	}
	if err := seedCLITerminal(st, runtimes, wsMain.ID, "claude", "claude-code", filepath.Join(dataDir, "work", "picode")); err != nil {
		return err
	}
	if _, err := st.CreateTerminalIn(wsSandbox.ID, "build", filepath.Join(dataDir, "work", "sandbox")); err != nil {
		return err
	}
	if err := seedCLITerminal(st, runtimes, wsSandbox.ID, "pi", "pi", filepath.Join(dataDir, "work", "sandbox")); err != nil {
		return err
	}

	if err := seedCanvas(st, wsMain.ID, atlasID); err != nil {
		return err
	}

	return nil
}

// seedCanvas lays out one canvas the public capture photographs: two agent
// CLIs side by side, the shell under them, and a text panel saying what the
// three are for. The rectangles are written here rather than packed, because
// the product no longer packs — a reader draws where a panel goes, and a
// capture that showed a neat automatic grid would be advertising a behaviour
// that is gone.
func seedCanvas(st *store.Store, wsID, atlasID string) error {
	c, err := st.CreateCanvas("Release day")
	if err != nil {
		return err
	}
	terms, err := st.ListTerminals()
	if err != nil {
		return err
	}
	byName := map[string]string{}
	for _, t := range terms {
		byName[t.Name] = t.ID
	}
	for _, p := range []struct {
		kind, ref  string
		x, y, w, h int
	}{
		{store.CanvasKindTerminal, byName["claude"], 0, 0, 56, 40},
		{store.CanvasKindAgent, atlasID, 60, 0, 56, 40},
		{store.CanvasKindTerminal, byName["shell"], 0, 44, 56, 32},
		{store.CanvasKindText, "", 60, 44, 56, 32},
	} {
		if p.ref == "" && p.kind != store.CanvasKindText {
			continue
		}
		added, err := st.AddCanvasPanel(c.ID, p.kind, p.ref, p.x, p.y, p.w, p.h)
		if err != nil {
			return err
		}
		if p.kind == store.CanvasKindText {
			if _, err := st.SetCanvasPanelContent(c.ID, added.Panel.ID, "Release day\n\nClaude is cutting the changelog, Atlas is\nchecking the docs, and the shell below runs\nthe gates."); err != nil {
				return err
			}
		}
	}
	return nil
}

// seedCLITerminal creates an agent-CLI terminal in ws and marks its CLI
// runtime live, the way the wrapper would when a real TUI attaches.
func seedCLITerminal(st *store.Store, runtimes *server.TermRuntimes, wsID, name, cli, cwd string) error {
	term, err := st.CreateTerminalIn(wsID, name, cwd)
	if err != nil {
		return err
	}
	runtimes.Start(term.ID, server.TermRuntime{
		CLI: cli, Source: "wrapper", RunID: "fixture-" + name,
		StartedAt: time.Now().UTC(),
	})
	return nil
}

// seedRepo turns dir into a synthetic repository: one commit of invented
// files, then an edit, a deletion and an untracked file, so a working-tree
// reader has something to count. Synthetic author, no signing, no global
// config — the capture must not depend on the machine's git identity.
func seedRepo(dir string) {
	if _, err := exec.LookPath("git"); err != nil {
		log.Printf("fixture: git missing, %s stays a plain folder", dir)
		return
	}
	files := map[string]string{
		"README.md":                 "# picode\n\nSynthetic fixture project for the docs captures.\n",
		"cmd/picode/main.go":        "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"picode\")\n}\n",
		"web/desktop/src/App.jsx":   "export default function App() {\n  return <main id=\"main\" />;\n}\n",
		"web/desktop/src/app.css":   "#app { display: flex; }\n",
		"docs/handoff.md":           "# Handoff\n\nNothing in flight.\n",
		"internal/server/server.go": "package server\n",
	}
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			log.Printf("fixture: seed repo: %v", err)
			return
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			log.Printf("fixture: seed repo: %v", err)
			return
		}
	}
	git := func(args ...string) bool {
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=PiCode Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("fixture: git %v: %v\n%s", args, err, out)
			return false
		}
		return true
	}
	if !git("init", "-q", "-b", "main") || !git("add", ".") || !git("commit", "-q", "-m", "seed: synthetic project") {
		return
	}
	// A bare "origin" beside the folder gives the branch an upstream, and one
	// more local commit leaves it one ahead — the chip reads `main ↑1`.
	origin := filepath.Join(filepath.Dir(dir), "origin.git")
	if git("init", "-q", "--bare", "-b", "main", origin) && git("remote", "add", "origin", origin) && git("push", "-q", "-u", "origin", "main") {
		if err := os.WriteFile(filepath.Join(dir, "docs", "handoff.md"), []byte("# Handoff\n\nThe seed is one commit ahead of origin.\n"), 0o644); err == nil {
			_ = git("add", ".") && git("commit", "-q", "-m", "docs: handoff records the seed")
		}
	}
	// The dirty tree the rail lists: an edited file, a deletion and a new one.
	edits := map[string]string{
		"README.md":               "# picode\n\nSynthetic fixture project for the docs captures.\n\nThe Inspector rail lists this edit.\n",
		"web/desktop/src/App.jsx": "export default function App() {\n  return (\n    <div id=\"app\">\n      <main id=\"main\" />\n      <aside id=\"inspector\" />\n    </div>\n  );\n}\n",
		"docs/inspector-notes.md": "# Inspector\n\nUntracked notes the fixture leaves behind.\n",
	}
	for name, body := range edits {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			log.Printf("fixture: seed repo: %v", err)
			return
		}
	}
	if err := os.Remove(filepath.Join(dir, "internal", "server", "server.go")); err != nil {
		log.Printf("fixture: seed repo: %v", err)
	}
}

// portSuffix names the fixture directory after the listen address: the port
// alone when the address parses, otherwise the whole address reduced to
// [A-Za-z0-9._-] so it is always a single safe path element.
func portSuffix(addr string) string {
	s := addr
	if _, port, err := net.SplitHostPort(addr); err == nil && port != "" {
		s = port
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == '-':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "default"
	}
	return string(out)
}
