package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// The measured table (docs/benchmarks/2026-09-23-attach-delivery-modes.md):
// a change here is a change to what PiCode types into a working CLI.
func TestDeliveryModesTable(t *testing.T) {
	want := map[string][]string{
		"pi":          {"prompt", "steer", "follow_up", "interrupt"},
		"omp":         {"prompt", "steer", "follow_up", "interrupt"},
		"hermes":      {"prompt", "steer", "follow_up", "interrupt"},
		"muse":        {"prompt", "steer", "follow_up", "interrupt"},
		"codex":       {"prompt", "steer", "follow_up", "interrupt"},
		"claude-code": {"prompt", "steer", "interrupt"},
		"opencode":    {"prompt", "steer", "interrupt"},
		"agy":         {"prompt", "follow_up", "interrupt"},
		"grok":        {"prompt", "follow_up", "interrupt"},
		"":            {"prompt"},
	}
	for cli, modes := range want {
		if got := deliveryModesFor(cli); !reflect.DeepEqual(got, modes) {
			t.Errorf("%q: got %v want %v", cli, got, modes)
		}
	}
	// Hermes gets exactly one Ctrl+C (a second within 2 s force-exits it);
	// Grok's Esc does not cancel; OpenCode confirms with a second Esc.
	want2 := map[string][]string{"hermes": {"C-c"}, "grok": {"C-c"}, "opencode": {"Escape", "Escape"}, "claude-code": {"Escape"}}
	for cli, keys := range want2 {
		if got := deliveryAdapters[cli].interrupt; !reflect.DeepEqual(got, keys) {
			t.Errorf("%s interrupt keys %v want %v", cli, got, keys)
		}
	}
	// Hermes never gets a bare Enter: with busy_input_mode "interrupt" it
	// cancels the turn. Both of its modes are slash commands.
	for _, m := range []string{deliverySteer, deliveryFollowUp} {
		if s := deliverySeqFor("hermes", m); s == nil || s.command == "" {
			t.Errorf("hermes %s must be a slash command, got %+v", m, s)
		}
	}
}

func TestMidTurnText(t *testing.T) {
	cases := []struct {
		seq     *deliverySeq
		payload string
		want    string
	}{
		{seqEnter, "look\n@a.png\n", "look\n@a.png"},
		{&deliverySeq{command: "/steer ", key: "Enter"}, "look  here\n@a.png", "/steer look here @a.png"},
		{&deliverySeq{command: "/queue ", key: "Enter"}, "one", "/queue one"},
	}
	for _, c := range cases {
		if got := midTurnText(c.seq, c.payload); got != c.want {
			t.Errorf("%q: got %q want %q", c.payload, got, c.want)
		}
	}
}

func TestMidTurnQueuedReceipt(t *testing.T) {
	head := deliveryHead("Also write BRAVO on its own line at the very end.\n@x.png")
	if head != "Also write BRAVO on its " {
		t.Fatalf("head %q", head)
	}
	snap := func(cursor int, lines ...string) tmux.InputSnapshot {
		return tmux.InputSnapshot{PaneID: "%1", CursorY: cursor, Lines: lines}
	}
	cases := []struct {
		name          string
		before, after tmux.InputSnapshot
		want          bool
	}{
		{"pi steering line", snap(2, "working", "", "Also write…"),
			snap(3, "working", " Steering: Also write BRAVO on its own line", "", ""), true},
		{"codex queue row with color", snap(1, "x", "›"),
			snap(2, "• Queued follow-up inputs", "  \x1b[2m↳ Also write BRAVO on its own line\x1b[0m", "›"), true},
		{"text only still in the input row", snap(0, "› Also write BRAVO on its own line"),
			snap(0, "› Also write BRAVO on its own line"), false},
		{"already on screen before, no new row", snap(1, "❯ Also write BRAVO on its own line", "❯"),
			snap(1, "❯ Also write BRAVO on its own line", "❯"), false},
		{"nothing rendered", snap(0, "❯"), snap(0, "❯"), false},
	}
	for _, c := range cases {
		if got := midTurnQueued(c.before, c.after, head); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
	if midTurnQueued(snap(0, "a"), snap(1, "a", "b"), "") {
		t.Error("an empty head never proves a queue")
	}
}

// deliveryHarness: a CLI terminal with a state registry the test drives.
func deliveryHarness(t *testing.T, cli string) (*httptest.Server, store.Terminal, *TermStates) {
	t.Helper()
	t.Cleanup(resetPromptInFlight)
	st := testStore(t)
	states := NewTermStates()
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
		TermStates: states,
	}).Handler)
	t.Cleanup(ts.Close)
	term, err := st.CreateTerminal("cli", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetTerminalLaunch(term.ID, cli, clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	return ts, term, states
}

// The ADR-0206 decision table on the wire. The mid-turn send itself needs a
// live TUI's queue render and is covered by the receipt test above plus the
// live measurement; every refusal row and the idle fallback are here.
func TestAttachDeliveryDecisionTable(t *testing.T) {
	now := time.Now()
	rows := []struct {
		name, cli, state, body string
		code                   int
		reason                 string
	}{
		{"bad mode", "claude-code", "", `{"message":"x","delivery":"later"}`, http.StatusBadRequest, ""},
		{"working prompt names the modes", "codex", TermWorking, `{"message":"x"}`, http.StatusConflict, TermWorking},
		{"working, mode the CLI lacks", "claude-code", TermWorking, `{"message":"x","delivery":"follow_up"}`, http.StatusConflict, "unsupported-mode"},
		{"idle, mode the CLI lacks", "agy", TermIdle, `{"message":"x","delivery":"steer"}`, http.StatusConflict, "unsupported-mode"},
		{"needs-you refuses steer", "pi", TermNeedsYou, `{"message":"x","delivery":"steer"}`, http.StatusConflict, TermNeedsYou},
		{"needs-you refuses follow-up", "codex", TermNeedsYou, `{"message":"x","delivery":"follow_up"}`, http.StatusConflict, TermNeedsYou},
		{"idle steer is a prompt", "opencode", TermIdle, `{"message":"x","delivery":"steer"}`, http.StatusOK, ""},
		{"needs-you refuses interrupt", "claude-code", TermNeedsYou, `{"message":"x","delivery":"interrupt"}`, http.StatusConflict, TermNeedsYou},
		{"idle interrupt is a prompt", "codex", TermIdle, `{"message":"x","delivery":"interrupt"}`, http.StatusOK, ""},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			fakeTmuxBin(t)
			ts, term, states := deliveryHarness(t, row.cli)
			if row.state != "" {
				states.Set(term.ID, row.state, row.cli, now)
			}
			code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/prompt", row.body)
			if code != row.code {
				t.Fatalf("code=%d want %d page=%v", code, row.code, page)
			}
			if row.reason != "" && page["reason"] != row.reason {
				t.Fatalf("reason=%v want %s page=%v", page["reason"], row.reason, page)
			}
		})
	}

	t.Run("working refusal names the other modes", func(t *testing.T) {
		fakeTmuxBin(t)
		ts, term, states := deliveryHarness(t, "codex")
		states.Set(term.ID, TermWorking, "codex", now)
		_, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/prompt", `{"message":"x"}`)
		if msg, _ := page["error"].(string); !strings.Contains(msg, "Steer or Follow-up") {
			t.Fatalf("error %q", msg)
		}
	})

	t.Run("modes endpoint", func(t *testing.T) {
		ts, term, states := deliveryHarness(t, "grok")
		states.Set(term.ID, TermWorking, "grok", now)
		res, err := http.Get(ts.URL + "/api/terminals/" + term.ID + "/prompt")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var page struct {
			CLI   string   `json:"cli"`
			Modes []string `json:"modes"`
			State string   `json:"state"`
		}
		if err := json.NewDecoder(res.Body).Decode(&page); err != nil {
			t.Fatal(err)
		}
		if page.CLI != "grok" || !reflect.DeepEqual(page.Modes, []string{"prompt", "follow_up", "interrupt"}) || page.State != TermWorking {
			t.Fatalf("%+v", page)
		}
	})
}

func TestPiDeliverAs(t *testing.T) {
	for mode, want := range map[string]string{"": "", "prompt": "", "steer": "steer", "follow_up": "followUp", "interrupt": "interrupt"} {
		if got := piDeliverAs(mode); got != want {
			t.Errorf("%q: %q want %q", mode, got, want)
		}
	}
	// The receiver maps anything but "steer" onto followUp, the behaviour
	// every sender had before ADR-0206.
	src, err := os.ReadFile("intercept/pi-inbox-reply.ts")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), `deliverAs: doc.deliverAs === "steer" ? "steer" : "followUp"`) {
		t.Fatal("receiver no longer honours doc.deliverAs")
	}
	if !strings.Contains(string(src), `doc.deliverAs === "interrupt"`) || !strings.Contains(string(src), "latestCtx.abort?.()") {
		t.Fatal("receiver no longer stops the turn for interrupt")
	}
}

// The mid-turn path end to end on the suite's isolated tmux: a stand-in
// CLI that renders what it takes on its own row earns "queued"; one that
// swallows input silently gets "unconfirmed" — the receipt never claims a
// queue the pane does not show.
func TestAttachDeliveryMidTurnOnTmux(t *testing.T) {
	m := tmux.New()
	if !m.Available() {
		t.Skip("tmux not available")
	}
	rows := []struct {
		name, script, want string
	}{
		{"renders the queue", `stty -echo; while IFS= read -r l; do printf 'QUEUED %s\n' "$l"; done`, "queued"},
		{"renders nothing", `stty -echo; cat >/dev/null`, "unconfirmed"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			ts, term, states := deliveryHarness(t, "opencode")
			name := tmux.ShellSessionName(term.ID)
			if err := m.NewSession(context.Background(), name, term.Cwd, "sh", "-c", row.script); err != nil {
				t.Fatalf("fixture session: %v", err)
			}
			t.Cleanup(func() {
				// A context of its own: t.Context() is already canceled here.
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = m.KillSession(ctx, name)
			})
			time.Sleep(300 * time.Millisecond)
			states.Set(term.ID, TermWorking, "opencode", time.Now())
			code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/prompt", `{"message":"also write BRAVO at the end","delivery":"steer"}`)
			if code != http.StatusOK || page["delivery"] != row.want {
				t.Fatalf("code=%d page=%v want delivery %s", code, page, row.want)
			}
		})
	}
}

func TestInterruptDetection(t *testing.T) {
	snap := func(cursor int, lines ...string) tmux.InputSnapshot {
		return tmux.InputSnapshot{PaneID: "%1", CursorY: cursor, Lines: lines}
	}
	// The measured stop lines, one per CLI.
	for _, line := range []string{
		"  ⎿  Interrupted · What should Claude do instead?",
		"■ Conversation interrupted - tell the model what to do differently.",
		" Operation aborted",
		"| Command aborted",
		"     ▣  Build · GLM-5.3-Flash · interrupted",
		"◆ Interrupted · Report issues with /feedback",
		"     Turn cancelled by user in 4.0s.",
		" Operation interrupted: waiting for model response (5.4s elapsed).",
	} {
		if markerRows(snap(0, line)) != 1 {
			t.Errorf("no stop marker in %q", line)
		}
	}
	if markerRows(snap(0, "reading README.md", "writing numbers")) != 0 {
		t.Error("ordinary rows are not stop markers")
	}

	cases := []struct {
		name          string
		before, after tmux.InputSnapshot
		refilled      bool
	}{
		{"claude empty", snap(0, "❯ "), snap(0, "❯ "), false},
		{"hermes italic suggestion", snap(0, "❯"),
			snap(0, "\x1b[38;5;230m❯ \x1b[3m\x1b[38;5;136mSummarize what's in this folder\x1b[0m"), false},
		{"codex dim placeholder", snap(0, "› \x1b[2mAsk Codex to do anything\x1b[0m"),
			snap(0, "› \x1b[2mAsk Codex to do anything\x1b[0m"), false},
		{"muse gave the prompt back", snap(0, "❯"), snap(0, "❯ Read the file README.md three times"), true},
		{"green text is not a placeholder", snap(0, "❯"), snap(0, "❯ \x1b[32mrestored\x1b[0m"), true},
		{"same framed row", snap(0, "│ ❯  │"), snap(0, "│ ❯  │"), false},
	}
	for _, c := range cases {
		if got := composerRefilled(c.before, c.after); got != c.refilled {
			t.Errorf("%s: refilled=%v want %v (text %q)", c.name, got, c.refilled, composerText(c.after))
		}
	}
}

// Stop and send end to end on the suite's isolated tmux: a stand-in CLI
// that prints "Interrupted" on Esc gets the message; one that ignores the
// key is refused with not-stopped and receives nothing.
func TestAttachInterruptOnTmux(t *testing.T) {
	m := tmux.New()
	if !m.Available() {
		t.Skip("tmux not available")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	prevSettle := interruptSettle
	interruptSettle = 1500 * time.Millisecond
	t.Cleanup(func() { interruptSettle = prevSettle })
	stops := `stty -echo -icanon min 1; while IFS= read -r -s -n1 -d '' c; do if [[ $c == $'\e' ]]; then printf 'Interrupted\n'; else printf '%s' "$c" >> got; fi; done`
	ignores := `stty -echo -icanon min 1; while IFS= read -r -s -n1 -d '' c; do printf '%s' "$c" >> got; done`
	rows := []struct {
		name, script string
		code         int
		reason       string
		sent         bool
	}{
		{"stops, then sends", stops, http.StatusOK, "", true},
		{"never stops, nothing sent", ignores, http.StatusConflict, "not-stopped", false},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			ts, term, states := deliveryHarness(t, "claude-code")
			name := tmux.ShellSessionName(term.ID)
			if err := m.NewSession(context.Background(), name, term.Cwd, "bash", "-c", row.script); err != nil {
				t.Fatalf("fixture session: %v", err)
			}
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = m.KillSession(ctx, name)
			})
			time.Sleep(300 * time.Millisecond)
			states.Set(term.ID, TermWorking, "claude-code", time.Now())
			code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/prompt", `{"message":"use the other file","delivery":"interrupt"}`)
			if code != row.code || (row.reason != "" && page["reason"] != row.reason) {
				t.Fatalf("code=%d page=%v want %d %s", code, page, row.code, row.reason)
			}
			time.Sleep(300 * time.Millisecond)
			got, _ := os.ReadFile(filepath.Join(term.Cwd, "got"))
			if sent := strings.Contains(string(got), "use the other file"); sent != row.sent {
				t.Fatalf("pane got %q, sent=%v want %v", got, sent, row.sent)
			}
		})
	}
}
