package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

const peerPointer = "PiCode: read pending messages and acknowledge handled ones."
const peerCLIPointer = "PiCode: run picode messages read; handle messages, then ack their IDs."
const peerActivationPointer = "PiCode activation: run picode messages read once to finish connecting, then reply activation ready. Do not change files."

var terminalSGR = regexp.MustCompile(`\x1b\[[0-9;:]*m`)

// Claude Code predicts a follow-up after an empty cursor and renders it dim.
// Recognize only that exact captured shape — a reset glyph, ❯ plus a
// non-breaking space, one dim run, no trailing content — so typed (bright)
// text and a cursor inside the text still refuse. Capture:
// testdata/claude-ghost-suggestion.json.
var claudeEmptySuggestion = regexp.MustCompile(`^\x1b\[39m❯\xa0\x1b\[2m([^\x1b\r\n\t]+)\x1b\[0m$`)

// Refusal reasons for the guarded input path. They are stable metadata: the
// log may carry the reason, never the pointer, message, token or screen.
var (
	errPeerSnapshotGone      = errors.New("pane snapshot unavailable")
	errPeerPaneChanged       = errors.New("pane changed")
	errPeerRuntimeChanged    = errors.New("native observation or runtime changed")
	errPeerConnectionChanged = errors.New("connection changed")
	errPeerPointerFit        = errors.New("pointer does not fit")
	errPeerNotRendered       = errors.New("paste not rendered yet")
	errPeerComposerChanged   = errors.New("composer changed or is not ready")
)

// Under load a TUI can take longer than one sample interval to render a
// paste (the 2026-09-14 matrix: Grok received every pointer, the Enter was
// withheld because the post-paste frame was sampled once, too early, and the
// attempt expired uncertain). The post-paste state is therefore polled
// inside a bounded window: Enter still goes only after one full-frame match
// of the same guards, and expiry refuses the attempt exactly as the single
// sample did — uncertain, never retried, no draft touched.
var (
	peerPasteSettleWindow   = 1200 * time.Millisecond
	peerPasteSettleInterval = 120 * time.Millisecond
)

// peerAwaitComposer samples the unchanged post-paste check until it passes or
// the bounded window expires. Structural refusals (pane, runtime, connection,
// fit) return immediately; only states a later render can settle keep polling.
func peerAwaitComposer(ctx context.Context, check func(string) error, expected string) error {
	deadline := time.Now().Add(peerPasteSettleWindow)
	for {
		err := check(expected)
		if err == nil || !peerUnsettled(err) {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().Add(peerPasteSettleInterval).After(deadline) {
			return err
		}
		timer := time.NewTimer(peerPasteSettleInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func peerUnsettled(err error) bool {
	return errors.Is(err, errPeerNotRendered) || errors.Is(err, errPeerComposerChanged) ||
		errors.Is(err, errPeerSnapshotGone)
}

var grokEmptySuggestion = regexp.MustCompile(`^  │ ❯ \x1b\[2;3m([^\x1b\r\n\t]+)\x1b\[0m( +)│ *$`)

// Grok 1.0.30 restyles the same unaccepted suggestion: the border, gutter and
// prompt glyph carry their own color, the text is italic plus a separate
// dim-gray foreground closed by \x1b[0m, the padding rows carry fg+bg colors and
// the closing border is styled too. Same requirements as the 1.0.25 shape:
// empty cursor, complete frame, exact suggestion footer. Capture:
// testdata/grok-bordered-suggestion-1030.json.
var grokEmptySuggestion1030 = regexp.MustCompile(`^  \x1b\[38;2;80;80;88m│\x1b\[38;2;225;225;225m \x1b\[38;2;200;200;200m❯ \x1b\[3m\x1b\[38;2;88;88;88m([^\x1b\r\n\t]+)\x1b\[0m\x1b\[38;2;225;225;225m\x1b\[48;2;20;20;20m( +)\x1b\[38;2;80;80;88m│\x1b\[39m *$`)

// grokSuggestionMatch recognizes either captured suggestion style and returns
// the suggestion text plus its padding so the row can be normalized to an
// empty editor.
func grokSuggestionMatch(line string) []string {
	if m := grokEmptySuggestion.FindStringSubmatch(line); m != nil {
		return m
	}
	return grokEmptySuggestion1030.FindStringSubmatch(line)
}

// The screen is an additional conservative input gate. A lifecycle hook and
// native session binding are mandatory independently of these cursor checks.
func peerInputMatches(cli string, s tmux.InputSnapshot, expected string) bool {
	if s.InMode || s.Width < 70 || s.CursorY < 0 || s.CursorY >= len(s.Lines) {
		return false
	}
	if cli == "pi" {
		if expected != "" || s.CursorX != 0 || s.CursorY < 1 || s.CursorY+3 >= len(s.Lines) {
			return false
		}
		clean := func(n int) string { return strings.TrimSpace(terminalSGR.ReplaceAllString(s.Lines[n], "")) }
		rule := strings.Repeat("─", s.Width)
		return clean(s.CursorY) == "" && clean(s.CursorY-1) == rule && clean(s.CursorY+1) == rule &&
			strings.HasPrefix(clean(s.CursorY+2), "/") && strings.Contains(clean(s.CursorY+3), "%/")
	}
	if cli == "opencode" {
		return peerOpenCodeInput(s, expected)
	}
	if cli == "grok" && strings.HasPrefix(strings.TrimSpace(terminalSGR.ReplaceAllString(s.Lines[s.CursorY], "")), "│") {
		return peerGrokBoxInput(s, expected)
	}
	raw := s.Lines[s.CursorY]
	line := strings.TrimRight(strings.ReplaceAll(terminalSGR.ReplaceAllString(raw, ""), "\u00a0", " "), " \t")
	if !peerSingleInputRow(cli, s) {
		return false
	}
	marker := "❯"
	if cli == "codex" {
		marker = "›"
	}
	switch cli {
	case "grok", "hermes", "claude-code", "codex":
	default:
		return false
	}
	if expected != "" {
		return line == marker+" "+expected && s.CursorX == 2+utf8.RuneCountInString(expected)
	}
	if s.CursorX != 2 {
		return false
	}
	if line == marker {
		return true
	}
	// Hermes marks its empty-input suggestion italic; normal user input does
	// not carry that marker. Never infer emptiness from cursor position alone.
	if (cli == "hermes" || cli == "codex") && strings.HasPrefix(line, marker+" ") && ((cli == "hermes" && strings.Contains(raw, "\x1b[3m")) || (cli == "codex" && strings.Contains(raw, "\x1b[2m"))) {
		return true
	}
	// Claude Code renders a predicted follow-up dim after the empty cursor.
	// The ghost is never submitted text: typed input is bright and parks the
	// cursor inside it, never at the ghost's start.
	if cli == "claude-code" && claudeEmptySuggestion.MatchString(raw) {
		return true
	}
	return false
}

// Grok's bordered composer places its cursor inside the frame. Match the whole
// one-line editor, model border and shortcut footer; never trim away draft text
// or accept a continuation row merely because the cursor is at its start.
func peerGrokBoxInput(s tmux.InputSnapshot, expected string) bool {
	y := s.CursorY
	if y < 1 || y+3 >= len(s.Lines) || s.CursorX != 6+utf8.RuneCountInString(expected) || strings.ContainsAny(expected, "\r\n\t\x1b") {
		return false
	}
	clean := func(n int) string { return strings.TrimRight(terminalSGR.ReplaceAllString(s.Lines[n], ""), " ") }
	if clean(y-1) != "  ╭"+strings.Repeat("─", s.Width-6)+"╮" {
		return false
	}
	line := clean(y)
	suggestion := false
	// Native Grok renders an unaccepted suggestion dim and italic at the
	// empty cursor. Require that exact style and footer together; typed or
	// partly accepted text must still fail the empty-editor check.
	if expected == "" {
		if match := grokSuggestionMatch(s.Lines[y]); match != nil {
			suggestion = true
			line = "  │ ❯ " + strings.Repeat(" ", utf8.RuneCountInString(match[1])+len(match[2])) + "│"
		}
	}
	padding := s.Width - 9 - utf8.RuneCountInString(expected)
	if padding < 1 || line != "  │ ❯ "+expected+strings.Repeat(" ", padding)+"│" {
		return false
	}
	bottom := clean(y + 1)
	if utf8.RuneCountInString(bottom) != s.Width-2 || !strings.HasPrefix(bottom, "  ╰─") || !strings.HasSuffix(bottom, " ─╯") {
		return false
	}
	model := strings.TrimLeft(strings.TrimSuffix(strings.TrimPrefix(bottom, "  ╰"), " ─╯"), "─")
	if !strings.HasPrefix(model, " Grok ") || !strings.Contains(model, " · ") || strings.ContainsAny(model, "│╭╮╰╯") {
		return false
	}
	footer := "  Shift+Tab:mode  │  Ctrl+x:shortcuts"
	if suggestion {
		footer = "  Tab/→:accept suggestion  │  Shift+Tab:mode  │  Ctrl+x:shortcuts"
	} else if expected != "" {
		footer = "  Enter:send  │  Shift+Tab:mode  │  Ctrl+x:shortcuts"
	}
	// Grok 1.0.34 appends a newline hint to the post-paste footer. Accept
	// that exact second shape for pasted text only; empty, suggestion and
	// welcome states keep their single footers.
	footers := []string{footer}
	if !suggestion && expected != "" {
		footers = append(footers, "  Enter:send  │  Shift+Enter/Alt+Enter:newline  │  Shift+Tab:mode  │  Ctrl+x:shortcuts")
	}
	// The first-turn welcome screen uses a right-aligned release channel.
	// Typing the pointer dismisses it and must produce the normal Enter footer.
	welcome := expected == "" && !suggestion && clean(y+3) == strings.Repeat(" ", s.Width-10)+"[stable]"
	matched := false
	for _, f := range footers {
		if clean(y+3) == f {
			matched = true
		}
	}
	if clean(y+2) != "" || (!matched && !welcome) {
		return false
	}
	for n := y + 4; n < len(s.Lines); n++ {
		if clean(n) != "" {
			return false
		}
	}
	return true
}

func peerPointerFits(cli string, s tmux.InputSnapshot, pointer string) bool {
	if cli == "opencode" {
		return 5+utf8.RuneCountInString(pointer) < peerOpenCodeWidth(s)
	}
	if cli != "grok" || s.CursorY < 0 || s.CursorY >= len(s.Lines) || !strings.HasPrefix(strings.TrimSpace(terminalSGR.ReplaceAllString(s.Lines[s.CursorY], "")), "│") {
		return true
	}
	return s.Width-9-utf8.RuneCountInString(pointer) >= 1
}

// Recognize the whole composer, not just its cursor row. Unknown layouts and
// multiline drafts remain pending. Native hooks separately gate permissions.
func peerSingleInputRow(cli string, s tmux.InputSnapshot) bool {
	clean := func(n int) string { return strings.TrimSpace(terminalSGR.ReplaceAllString(s.Lines[n], "")) }
	rule := func(n int) bool {
		if n < 0 || n >= len(s.Lines) {
			return false
		}
		line := clean(n)
		return utf8.RuneCountInString(line) >= 10 && strings.Trim(line, "─━") == ""
	}
	switch cli {
	case "codex":
		y := s.CursorY + 2
		if y >= len(s.Lines) || clean(y-1) != "" || !strings.Contains(clean(y), " · ") {
			return false
		}
		for n := y + 1; n < len(s.Lines); n++ {
			if clean(n) != "" {
				return false
			}
		}
		return true
	case "grok":
		// Grok's composer is its final prompt immediately before the status bar.
		// A continuation below the cursor or a prompt in history cannot match.
		y := s.CursorY + 1
		if y >= len(s.Lines) || !strings.Contains(clean(y), "ctrl+o transcript") {
			return false
		}
		for n := y + 1; n < len(s.Lines); n++ {
			if clean(n) != "" {
				return false
			}
		}
		return true
	case "hermes", "claude-code":
		if !rule(s.CursorY-1) || !rule(s.CursorY+1) {
			return false
		}
		// A rule-shaped user line cannot impersonate the lower border: the real
		// lower border and any remaining draft would still be visible below it.
		tail := []string{}
		for n := s.CursorY + 2; n < len(s.Lines); n++ {
			if line := clean(n); line != "" {
				tail = append(tail, line)
			}
		}
		// Claude moves its remote-control shortcut to a third footer row in
		// narrow panes. Accept that exact shortcut, not arbitrary draft text.
		if cli == "claude-code" && len(tail) == 3 && tail[2] == "/rc" {
			tail = tail[:2]
		}
		return len(tail) == 0 || (cli == "claude-code" && ((len(tail) == 1 && (strings.Contains(tail[0], "? for shortcuts") || strings.HasPrefix(tail[0], "⏸ manual mode on") || strings.Contains(tail[0], "(shift+tab to cycle)"))) || (len(tail) == 2 && strings.Contains(tail[0], " | ") && strings.Contains(tail[1], "shift+tab"))))
	default:
		return false
	}
}

// OpenCode's standard session composer has three input rows, its model row,
// a lower border, and a command footer. Require that entire native frame.
func peerOpenCodeWidth(s tmux.InputSnapshot) int {
	if s.CursorY < 2 || s.CursorY+4 >= len(s.Lines) {
		return 0
	}
	border := []rune(terminalSGR.ReplaceAllString(s.Lines[s.CursorY+3], ""))
	if len(border) < 20 || string(border[:3]) != "  ╹" {
		return 0
	}
	width := 3
	for width < len(border) && border[width] == '▀' {
		width++
	}
	if width < 20 || width > s.Width {
		return 0
	}
	for col := width; col < width+2 && col < len(border); col++ {
		if border[col] != ' ' {
			return 0
		}
	}
	return width
}

var openCodePathContinuation = regexp.MustCompile(`^   [A-Za-z0-9._~/\-]+$`)

func peerOpenCodeInput(s tmux.InputSnapshot, expected string) bool {
	y := s.CursorY
	if y < 2 || y+4 >= len(s.Lines) || s.CursorX != 5+utf8.RuneCountInString(expected) {
		return false
	}
	plain := func(n int) string { return terminalSGR.ReplaceAllString(s.Lines[n], "") }
	// The lower border defines the editor's column boundary. The native
	// sidebar can share these rows without becoming part of the composer.
	width := peerOpenCodeWidth(s)
	if width == 0 || s.CursorX >= width {
		return false
	}
	for n := y - 2; n < len(s.Lines); n++ {
		line := []rune(plain(n))
		for col := width; col < width+2 && col < len(line); col++ {
			if line[col] != ' ' {
				return false
			}
		}
	}
	clean := func(n int) string {
		line := []rune(plain(n))
		if len(line) > width {
			line = line[:width]
		}
		return strings.TrimRight(string(line), " ")
	}
	if clean(y-2) != "" || clean(y-1) != "  ┃" || clean(y+1) != "  ┃" {
		return false
	}
	want := "  ┃"
	if expected != "" {
		want += "  " + expected
	}
	if clean(y) != want || !strings.HasPrefix(clean(y+2), "  ┃  ") || !strings.Contains(clean(y+2), " · ") {
		return false
	}
	footer := clean(y + 4)
	if !strings.HasSuffix(footer, "ctrl+p commands") {
		return false
	}
	continuations := 0
	for n := y + 5; n < len(s.Lines); n++ {
		line := clean(n)
		if line == "" {
			continue
		}
		// The native footer wraps the cwd across as many rows as its length
		// needs — deep worktree paths at narrow widths wrap twice or more, and
		// every extra row is still below the lower border where a user draft
		// cannot be. Accept only those path-only rows, never arbitrary text.
		if !strings.HasPrefix(footer, "   /") || continuations == 8 || !openCodePathContinuation.MatchString(line) {
			return false
		}
		continuations++
	}

	return true
}

// Deliver through Pi's existing native receiver with the originally bound
// session file. The receiver compares it inside Pi immediately before submit.
func deliverPeerPointer(ctx context.Context, deps Deps, key, session string) error {
	return deliverPeerText(ctx, deps, key, session, peerPointer)
}
func deliverPeerText(ctx context.Context, deps Deps, key, session, text string) error {
	if deps.Replies == nil || deps.Replies.receiverSession(key) != session {
		return errors.New("receiver changed session")
	}
	nonce, err := newReplyNonce()
	if err != nil {
		return err
	}
	ack, done := deps.Replies.registerAck(nonce)
	defer done()
	file, err := writeReplyFile(deps.DataDir, key, replyFile{AttentionOnly: true, Nonce: nonce, SessionPath: session, Payload: text, CreatedAt: time.Now().UTC(), PID: deps.Replies.receiverPID(key)})
	if err != nil {
		return err
	}
	defer os.Remove(file)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case a := <-ack:
		if !a.OK {
			return errors.New(a.Reason)
		}
		return nil
	}
}

func peerLiveTerminal(deps Deps, p store.PeerConnection) (TermRuntime, bool) {
	rt, ok := deps.TermRuntimes.Get(p.OwnerID)
	if !ok || rt.SessionID != p.SessionKey || rt.CLI != p.CLI || rt.RunID == "" || !processAlive(rt) {
		return rt, false
	}
	observation, observationErr := readNativeObservation(deps.DataDir, p.OwnerID)
	if rt.Observation || observationErr == nil || (deps.DataDir != "" && !os.IsNotExist(observationErr)) {
		if observationErr != nil || !observationMatchesRuntime(observation, rt) || observation.State != TermIdle {
			return rt, false
		}
	}

	state, ok := deps.TermStates.Get(p.OwnerID)
	return rt, ok && state.SessionID == rt.SessionID && state.SessionSeq == rt.SessionSeq && state.RunID == rt.RunID && state.CLI == rt.CLI && state.State == TermIdle
}

func attentionFinish(deps Deps, m store.PeerMessage, err error) {
	status := "notified"
	if err != nil {
		status = "uncertain"
	}
	_, _ = deps.Store.SetPeerAttention(m.ID, m.RecipientID, "attempted", status)
}

func attemptPeerAttention(ctx context.Context, deps Deps, m store.PeerMessage) {
	p, err := deps.Store.PeerConnection(m.RecipientID)
	if err != nil {
		return
	}
	pointer := peerPointer
	if communication.NativeMessages(p.CLI) {
		pointer = peerCLIPointer
	}
	attemptPeerText(ctx, deps, p, pointer, func() bool {
		ok, e := deps.Store.SetPeerAttention(m.ID, p.ID, "pending", "attempted")
		return e == nil && ok
	}, func(e error) { attentionFinish(deps, m, e) })
}

// The sender of a connection test uses the same guarded native input path.
func attemptPeerText(ctx context.Context, deps Deps, p store.PeerConnection, pointer string, claim func() bool, finish func(error)) {
	complete := finish
	finish = func(err error) {
		if err != nil {
			// Metadata only: never log the pointer, message, token or screen.
			log.Printf("communication attention: owner=%s cli=%s connection=%s reason=%s", p.OwnerID, p.CLI, p.ID, peerAttentionReason(err))
		}
		complete(err)
	}
	if p.Kind == "agent" {
		if deps.Replies == nil {
			return
		}
		release, e := deps.Replies.Controls.TryBeginMutation(p.OwnerID)
		if e != nil {
			return
		}
		defer release()
		if deps.Runtime == nil {
			return
		}
		ma := deps.Runtime.Get(p.OwnerID)
		if ma == nil {
			return
		}
		state := ma.Snapshot()
		if state.Streaming || state.Waiting {
			return
		}
		native, err := ma.GetState(ctx)
		if err != nil {
			return
		}
		var v struct {
			SessionFile string `json:"sessionFile"`
		}
		if json.Unmarshal(native.Data, &v) != nil || v.SessionFile != p.SessionKey || deps.Replies == nil || deps.Replies.receiverSession(p.OwnerID) != p.SessionKey {
			return
		}
		if !claim() {
			return
		}
		finish(deliverPeerText(ctx, deps, p.OwnerID, p.SessionKey, pointer))
		return
	}
	if deps.Tmux == nil || deps.TermRuntimes == nil || deps.TermStates == nil {
		return
	}
	rt, ok := peerLiveTerminal(deps, p)
	if !ok {
		return
	}
	if p.CLI == "pi" {
		if deps.Replies == nil || !deps.Replies.receiverFresh(termReplyKey(p.OwnerID)) {
			return
		}
		terminal, err := deps.Store.GetTerminal(p.OwnerID)
		if err != nil {
			return
		}
		session, err := deps.terminalAskSession(p.OwnerID, terminal.Cwd)
		if err != nil || session != rt.SessionPath {
			return
		}
		// The receiver refuses drafts too, but claiming before that refusal
		// would turn safely deferred attention into a non-retryable attempt.
		snap, err := deps.Tmux.InputSnapshot(ctx, tmux.ShellSessionName(p.OwnerID))
		if err != nil || !peerInputMatches("pi", snap, "") {
			return
		}
		current, ready := peerLiveTerminal(deps, p)
		if !ready || current.RunID != rt.RunID || current.PID != rt.PID {
			return
		}
		if !claim() {
			return
		}
		err = deliverPeerText(ctx, deps, termReplyKey(p.OwnerID), session, pointer)
		finish(err)
		return
	}
	if !tryLockPrompt(p.OwnerID) {
		return
	}
	defer unlockPrompt(p.OwnerID)
	name := tmux.ShellSessionName(p.OwnerID)
	before, err := deps.Tmux.InputSnapshot(ctx, name)
	if err != nil || !peerInputMatches(p.CLI, before, "") || !peerPointerFits(p.CLI, before, pointer) {
		return
	}
	// Require the wrapper to remain in the exact pane's process ancestry.
	procs := readProcSnapshot()
	pid := rt.PID
	for pid > 0 && pid != before.PanePID {
		pid = procs.ppid[pid]
	}
	if pid != before.PanePID {
		return
	}
	check := func(expected string) error {
		current, e := deps.Store.PeerConnection(p.ID)
		if e != nil || current.SessionKey != p.SessionKey {
			return errPeerConnectionChanged
		}
		live, ok := peerLiveTerminal(deps, p)
		if !ok || live.RunID != rt.RunID || live.PID != rt.PID || live.ProcStart != rt.ProcStart {
			return errPeerRuntimeChanged
		}
		snap, e := deps.Tmux.InputSnapshot(ctx, name)
		return peerInputRecheck(p.CLI, before, snap, expected, pointer, e)
	}
	if check("") != nil {
		return
	}
	if !claim() {
		return
	}
	// Any loss of certainty after the durable claim is terminal for this
	// attempt. Never remove an editor draft or blindly send Enter on retry.
	if err := check(""); err != nil {
		finish(peerAttentionFailure("before paste: " + err.Error()))
		return
	}
	if err := deps.Tmux.PasteOnly(ctx, before.PaneID, pointer); err != nil {
		finish(peerAttentionFailure("paste failed"))
		return
	}
	// The recipient's TUI may lag the paste under load; sample the exact
	// post-paste frame inside the bounded settle window instead of trusting
	// one instant.
	if err := peerAwaitComposer(ctx, check, pointer); err != nil {
		finish(peerAttentionFailure("after paste: " + err.Error()))
		return
	}
	if err := deps.Tmux.SubmitPane(ctx, before.PaneID); err != nil {
		finish(peerAttentionFailure("submit failed"))
		return
	}
	finish(nil)
}

func peerInputRecheck(cli string, before, current tmux.InputSnapshot, expected, pointer string, err error) error {
	if err != nil {
		return errPeerSnapshotGone
	}
	if current.PaneID != before.PaneID || current.PanePID != before.PanePID {
		return errPeerPaneChanged
	}
	if !peerPointerFits(cli, current, pointer) {
		return errPeerPointerFit
	}
	if peerInputMatches(cli, current, expected) {
		return nil
	}
	// Classify without exposing screen content: an editor that still matches
	// the pre-paste empty frame means the TUI has not rendered the paste yet,
	// the one state a later sample can settle. Anything else stays refused.
	if expected != "" && peerInputMatches(cli, current, "") {
		return errPeerNotRendered
	}
	return errPeerComposerChanged
}

// One coalesced loop for the server, driven by the feed. The bounded tick also
// notices local terminal input changes that cannot produce browser feed events.
func StartPeerAttention(ctx context.Context, deps Deps) {
	if deps.Store == nil || deps.Feed == nil {
		return
	}
	wake := make(chan struct{}, 1)
	deps.Feed.Listen(func(e store.Event) {
		if strings.HasPrefix(e.Type, "peer.") || e.Type == "terminal.last_session" || e.Type == "terminal.state" || e.Type == "terminal.runtime" || e.Type == "agent.state" {
			select {
			case wake <- struct{}{}:
			default:
			}
		}
	})
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
		case <-tick.C:
		}
		reconcilePeerParticipants(ctx, deps)
		reconcilePeerChecks(ctx, deps)
		sweepPeerLaunchFiles(deps)
		messages, err := deps.Store.PendingPeerAttention()
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		for _, m := range messages {
			if seen[m.RecipientID] {
				continue
			}
			seen[m.RecipientID] = true
			call, cancel := context.WithTimeout(ctx, 3*time.Second)
			attemptPeerAttention(call, deps, m)
			cancel()
		}
	}
}

// Only bounded internal reasons reach logs; receiver error text may contain
// untrusted native output and is deliberately omitted.
type peerAttentionFailure string

func (e peerAttentionFailure) Error() string { return string(e) }
func peerAttentionReason(err error) string {
	var failure peerAttentionFailure
	if errors.As(err, &failure) {
		return string(failure)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "delivery timed out"
	}
	if errors.Is(err, context.Canceled) {
		return "delivery cancelled"
	}
	return "native receiver or control rejected delivery"
}

// peerComposerState classifies a CLI pane's input row for the prompt door
// (Fatia F follow-up, owner report 2026-09-20: clean composers were
// refused on every agent terminal). Three answers:
//
//   - "empty": the recognized empty-input row — delivery proceeds and can
//     be verified.
//   - "occupied": positive evidence of a draft (the row carries visible
//     text past the prompt marker) — delivery is refused.
//   - "unknown": the row cannot be classified (rendering drift, dialog,
//     narrow pane). Delivery proceeds — the pre-Fatia-F behaviour — with
//     the receipt demoted, because refusing every unclassifiable pane
//     blocked clean composers the moment a CLI shipped a new render.
//
// Falls back to the strict empty matcher for the known-empty answer, so
// the two never disagree about a pane either of them can classify.
func peerComposerState(cli string, s tmux.InputSnapshot) (string, bool) {
	if peerInputMatches(cli, s, "") {
		return "empty", true
	}
	if peerComposerHoldsDraft(cli, s) {
		return "occupied", true
	}
	return "unknown", false
}

func peerComposerHoldsDraft(cli string, s tmux.InputSnapshot) bool {
	if s.InMode || s.CursorY < 0 || s.CursorY >= len(s.Lines) {
		return false
	}
	raw := s.Lines[s.CursorY]
	line := strings.TrimRight(strings.ReplaceAll(terminalSGR.ReplaceAllString(raw, ""), "\u00a0", " "), " \t")
	switch cli {
	case "claude-code":
		// Positive draft: the prompt glyph followed by bright text. Dim
		// content is the vendor's own suggestion, never a draft (measured:
		// typed input carries no dim), so a dim row is unknown, not
		// occupied — rendering drift must deliver, not block.
		if strings.HasPrefix(line, "❯") {
			if strings.Contains(raw, "\x1b[2m") {
				return false
			}
			return strings.TrimSpace(strings.TrimPrefix(line, "❯")) != ""
		}
	case "codex", "hermes":
		for _, marker := range []string{"›", "❯"} {
			if strings.HasPrefix(line, marker) {
				if strings.Contains(raw, "\x1b[2m") {
					return false
				}
				return strings.TrimSpace(strings.TrimPrefix(line, marker)) != ""
			}
		}
	case "grok":
		// Bordered composer: content between the gutter and the border.
		if strings.HasPrefix(strings.TrimSpace(terminalSGR.ReplaceAllString(raw, "")), "│") {
			if strings.Contains(line, "❯") {
				body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "│ ❯"), "│"))
				return body != "" && !grokEmptySuggestion.MatchString(raw) && !grokEmptySuggestion1030.MatchString(raw)
			}
		}
	case "pi":
		// Frame region: content between the two rules at the cursor.
		y := s.CursorY
		if y < 1 || y+1 >= len(s.Lines) {
			return false
		}
		clean := func(n int) string { return strings.TrimSpace(terminalSGR.ReplaceAllString(s.Lines[n], "")) }
		rule := strings.Repeat("─", s.Width)
		return clean(y-1) == rule && clean(y+1) == rule && clean(y) != ""
	}
	return false
}
