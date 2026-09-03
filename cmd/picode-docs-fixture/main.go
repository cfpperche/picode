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
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cfpperche/picode/internal/apps"
	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/presence"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/server"
	"github.com/cfpperche/picode/internal/share"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

const (
	// 18740: outside the daemon own port-climb range (8445+, seen at 8490
	// on this machine), so the fixture never collides with a real one.
	addr    = "127.0.0.1:18740"
	dataDir = "/tmp/picode-docs-fixture"
)

func main() {
	// Synthetic HOME before anything resolves a path: the app reads real pi
	// sessions (~/.pi/agent/sessions) for usage stats, and the fixture must
	// never publish the user's real spend — parity means synthetic data only.
	if err := os.Setenv("HOME", filepath.Join(dataDir, "home")); err != nil {
		log.Fatalf("fixture: home: %v", err)
	}
	// Deterministic: every run seeds the same synthetic world.
	if err := os.RemoveAll(dataDir); err != nil {
		log.Fatalf("fixture: clean %s: %v", dataDir, err)
	}
	for _, d := range []string{
		dataDir,
		filepath.Join(dataDir, "home", ".pi", "agent"),
		filepath.Join(dataDir, "work", "picode"),
		filepath.Join(dataDir, "work", "website"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Fatalf("fixture: mkdir %s: %v", d, err)
		}
	}

	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		log.Fatalf("fixture: store: %v", err)
	}
	defer st.Close()
	if err := seed(st); err != nil {
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

	deps := server.Deps{
		Store:    st,
		Auth:     nil, // ungated: the capture browser walks straight in
		Tmux:     tmux.New(),
		Runtime:  runtime,
		AgentCmd: "pi", // ADR-0003; never spawned by the fixture
		DataDir:  dataDir,
		Presence: devices,
		Feed:     changes,
		// The mobile inbox screen is an apps-host surface (ADR-0036):
		// without the registry its iframe dies and the screen shows
		// "Reconnecting" forever — the capture gate caught exactly that.
		Apps: apps.NewRegistry(apps.BuiltIns(false)...),
	}

	srv := server.New(addr, deps)
	log.Printf("fixture: synthetic PiCode on http://%s (data: %s)", addr, dataDir)
	log.Fatal(http.ListenAndServe(addr, srv.Handler))
}

// seed fills the store with a fixed, synthetic world. Names, states and
// numbers are invented; nothing here is a real person, inbox or spend.
func seed(st *store.Store) error {
	wsMain, err := st.AddWorkspace("picode", filepath.Join(dataDir, "work", "picode"))
	if err != nil {
		return err
	}
	wsSite, err := st.AddWorkspace("website", filepath.Join(dataDir, "work", "website"))
	if err != nil {
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

	return nil
}
