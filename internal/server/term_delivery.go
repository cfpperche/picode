package server

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// Attach delivery modes (ADR-0206). The door's third answer to a working
// CLI: instead of refusing, hand the message to the CLI's own mid-turn
// queue — "steer" (reaches the model inside the running turn) or
// "follow_up" (held until the turn ends). Each CLI does this its own way,
// measured live in docs/benchmarks/2026-09-23-attach-delivery-modes.md;
// the table below is that measurement, and a mode a CLI does not have is
// refused, never approximated.

const (
	deliveryPrompt    = "prompt"
	deliverySteer     = "steer"
	deliveryFollowUp  = "follow_up"
	deliveryInterrupt = "interrupt"
)

// deliverySeq is how one mode reaches one CLI: the bracketed paste of the
// payload (behind a slash command when `command` is set), then exactly one
// key. There is never a second key — in Grok an Enter on the emptied row
// means "cancel the turn and send now".
type deliverySeq struct {
	command string // "/steer " — the payload is flattened to one line behind it
	key     string // tmux key name pressed once after the paste
}

type deliveryAdapter struct {
	steer, followUp *deliverySeq
	// interrupt is the key sequence that stops the running turn (the
	// "interrupt" mode, ADR-0206 amendment): pressed, then the door waits for
	// the CLI's own "interrupted" line and sends a verified prompt.
	interrupt []string
}

var (
	seqEnter    = &deliverySeq{key: "Enter"}
	seqAltEnter = &deliverySeq{key: "M-Enter"}
	seqTab      = &deliverySeq{key: "Tab"}
)

// deliveryAdapters: the measured table. Hermes's plain Enter is absent on
// purpose (with busy_input_mode "interrupt" it cancels the turn); Grok
// steer needs ui.follow_up_behavior, which PiCode does not read yet.
//
// Interrupt keys, measured 2026-09-24: Esc everywhere except OpenCode (a
// second Esc within 5 s confirms) and Grok / Hermes (Ctrl+C; Esc only
// toasts in Grok). Hermes gets exactly one Ctrl+C — a second within 2 s
// force-exits the CLI.
var (
	keyEsc      = []string{"Escape"}
	keyEscTwice = []string{"Escape", "Escape"}
	keyCtrlC    = []string{"C-c"}
)

var deliveryAdapters = map[string]deliveryAdapter{
	"pi":          {steer: seqEnter, followUp: seqAltEnter, interrupt: keyEsc},
	"omp":         {steer: seqEnter, followUp: &deliverySeq{command: "/queue ", key: "Enter"}, interrupt: keyEsc},
	"hermes":      {steer: &deliverySeq{command: "/steer ", key: "Enter"}, followUp: &deliverySeq{command: "/queue ", key: "Enter"}, interrupt: keyCtrlC},
	"muse":        {steer: seqEnter, followUp: seqAltEnter, interrupt: keyEsc},
	"codex":       {steer: seqEnter, followUp: seqTab, interrupt: keyEsc},
	"claude-code": {steer: seqEnter, interrupt: keyEsc},
	"opencode":    {steer: seqEnter, interrupt: keyEscTwice},
	"agy":         {followUp: seqEnter, interrupt: keyEsc},
	"grok":        {followUp: seqEnter, interrupt: keyCtrlC},
}

// deliveryModesFor lists the modes the door offers for a CLI, prompt first.
func deliveryModesFor(cli string) []string {
	modes := []string{deliveryPrompt}
	a := deliveryAdapters[cli]
	if a.steer != nil {
		modes = append(modes, deliverySteer)
	}
	if a.followUp != nil {
		modes = append(modes, deliveryFollowUp)
	}
	if len(a.interrupt) > 0 {
		modes = append(modes, deliveryInterrupt)
	}
	return modes
}

func deliverySeqFor(cli, mode string) *deliverySeq {
	a := deliveryAdapters[cli]
	switch mode {
	case deliverySteer:
		return a.steer
	case deliveryFollowUp:
		return a.followUp
	}
	return nil
}

func validDelivery(mode string) bool {
	return mode == "" || mode == deliveryPrompt || mode == deliverySteer || mode == deliveryFollowUp || mode == deliveryInterrupt
}

// deliveryModeLabel is the word the composer shows for a mode.
func deliveryModeLabel(mode string) string {
	switch mode {
	case deliverySteer:
		return "Steer"
	case deliveryFollowUp:
		return "Follow-up"
	case deliveryInterrupt:
		return "Stop and send"
	}
	return "Prompt"
}

// workingRefusal is today's 409 working, now naming the modes that would
// have gone through.
func workingRefusal(cli, state string) map[string]any {
	body := map[string]any{"error": "The CLI is " + state + ". Try again when it is your turn.", "reason": state}
	if state != TermWorking {
		return body
	}
	var alt []string
	for _, m := range deliveryModesFor(cli) {
		if m != deliveryPrompt {
			alt = append(alt, deliveryModeLabel(m))
		}
	}
	if len(alt) > 0 {
		body["error"] = "The CLI is working. Choose " + strings.Join(alt, " or ") + " to send now."
		body["modes"] = deliveryModesFor(cli)
	}
	return body
}

// doorDeliverAs is the attended door with a mode: prompt keeps today's
// path; steer / follow_up go through the adapter when the CLI is working
// and fall back to the prompt path when it is not (every measured CLI
// sends at once when idle).
func doorDeliverAs(deps Deps, ctx context.Context, t store.Terminal, payload, mode string) (int, map[string]any) {
	if mode == "" || mode == deliveryPrompt {
		return doorDeliver(deps, ctx, t, payload)
	}
	cli := terminalLaunchCLI(deps, t.ID)
	if mode == deliveryInterrupt {
		return doorDeliverInterrupt(deps, ctx, t, cli, payload)
	}
	seq := deliverySeqFor(cli, mode)
	if seq == nil {
		return http.StatusConflict, map[string]any{
			"error":  deliveryModeLabel(mode) + " is not available for this CLI.",
			"reason": "unsupported-mode", "modes": deliveryModesFor(cli),
		}
	}
	st, ok := deps.TermStates.Get(t.ID)
	if !ok || st.State != TermWorking {
		if ok && (st.State == TermNeedsYou || st.State == TermCompacting) {
			return http.StatusConflict, workingRefusal(cli, st.State)
		}
		return doorDeliver(deps, ctx, t, payload)
	}
	return doorDeliverMidTurn(deps, ctx, t, cli, payload, seq)
}

// doorDeliverMidTurn sends into a working CLI. Same guards as the prompt
// door (tmux, session, one in flight, a recognized draft refused), no
// retry Enter, and a receipt read from the CLI's own queue render.
func doorDeliverMidTurn(deps Deps, ctx context.Context, t store.Terminal, cli, payload string, seq *deliverySeq) (int, map[string]any) {
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return http.StatusServiceUnavailable, map[string]any{"error": "Need tmux to send to a terminal."}
	}
	session := tmux.ShellSessionName(t.ID)
	cctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	has, err := deps.Tmux.HasSession(cctx, session)
	if err != nil {
		return http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again."}
	}
	if !has {
		return http.StatusConflict, map[string]any{"error": "Start the terminal first.", "reason": "closed"}
	}
	if !tryLockPrompt(t.ID) {
		return http.StatusConflict, map[string]any{"error": "Already sending to this terminal.", "reason": "busy"}
	}
	defer unlockPrompt(t.ID)
	// Re-read under the lock: an approval dialog that opened since the
	// check treats keys as choices.
	if st, ok := deps.TermStates.Get(t.ID); ok && st.State == TermNeedsYou {
		return http.StatusConflict, workingRefusal(cli, st.State)
	}
	before, err := deps.Tmux.InputSnapshot(cctx, session)
	if err != nil {
		return http.StatusConflict, map[string]any{"error": "PiCode could not read this terminal, so nothing was sent.", "reason": "unobservable"}
	}
	if doorReaderCLI[cli] {
		if state, known := peerComposerState(cli, before); known && state == "occupied" {
			return http.StatusConflict, map[string]any{"error": "Finish or clear the draft in the terminal first.", "reason": "occupied"}
		}
	}
	text := midTurnText(seq, payload)
	if err := deps.Tmux.PasteOnly(ctx, before.PaneID, text); err != nil {
		return http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again.", "reason": "closed"}
	}
	time.Sleep(doorComposerLag)
	if err := deps.Tmux.SendPaneKey(ctx, before.PaneID, seq.key); err != nil {
		return http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again.", "reason": "closed"}
	}
	head := deliveryHead(payload)
	deadline := time.Now().Add(doorSettleWindow)
	for {
		time.Sleep(doorSettleInterval)
		after, e := deps.Tmux.InputSnapshot(cctx, session)
		if e == nil && after.PaneID == before.PaneID && midTurnQueued(before, after, head) {
			announceDoorPrompt(deps, t.ID, "queued")
			return http.StatusOK, map[string]any{"ok": true, "typed": true, "delivery": "queued"}
		}
		if e != nil || after.PaneID != before.PaneID || time.Now().After(deadline) {
			announceDoorPrompt(deps, t.ID, "unconfirmed")
			return http.StatusOK, map[string]any{"ok": true, "typed": true, "delivery": "unconfirmed", "reason": "no-queue-render"}
		}
	}
}

// midTurnText is what gets pasted: the payload itself, or the payload
// flattened onto one line behind a slash command (a newline would submit
// the command early).
func midTurnText(seq *deliverySeq, payload string) string {
	payload = strings.TrimRight(payload, "\n")
	if seq.command == "" {
		return payload
	}
	return seq.command + strings.Join(strings.Fields(payload), " ")
}

// deliveryHead is the part of a payload every measured queue render shows:
// the first words of its first line (Pi truncates each queued row to one
// line, Codex to three).
func deliveryHead(payload string) string {
	for _, line := range strings.Split(payload, "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			continue
		}
		r := []rune(line)
		if len(r) > 24 {
			r = r[:24]
		}
		return string(r)
	}
	return ""
}

// midTurnQueued: the head shows on more rows outside the input row than it
// did before the send. Every measured CLI renders the queued text on its
// own row (a queue list, a steering line, a transcript row with a badge),
// and the input row itself was just emptied — so a count that grew is the
// CLI saying it took the message.
func midTurnQueued(before, after tmux.InputSnapshot, head string) bool {
	if head == "" {
		return false
	}
	return headRows(after, head) > headRows(before, head)
}

func headRows(s tmux.InputSnapshot, head string) int {
	n := 0
	for i, raw := range s.Lines {
		if i == s.CursorY {
			continue
		}
		line := strings.Join(strings.Fields(strings.ReplaceAll(terminalSGR.ReplaceAllString(raw, ""), " ", " ")), " ")
		if strings.Contains(line, head) {
			n++
		}
	}
	return n
}

// GET /api/terminals/{id}/prompt — what the attach composer may offer:
// the CLI's modes and its state right now. A terminal bound to an
// interactive Pi agent answers for that agent (routeBoundPi).
func handleTerminalPromptModes(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if routeBoundPi(deps, w, r, handleAgentPromptModes(deps)) {
			return
		}
		t, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if !termHoldsCLI(deps, t.ID) {
			writeErr(w, http.StatusConflict, "Attach is for Agent CLI terminals.")
			return
		}
		cli := terminalLaunchCLI(deps, t.ID)
		state := ""
		if st, ok := deps.TermStates.Get(t.ID); ok {
			state = st.State
		}
		writeJSON(w, http.StatusOK, map[string]any{"cli": cli, "modes": deliveryModesFor(cli), "state": state, "termId": t.ID})
	}
}

// GET /api/agents/{id}/prompt — an interactive Pi agent takes every mode
// through its receiver; a managed agent's composer has its own selector.
func handleAgentPromptModes(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent, err := deps.Store.GetAgent(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		state, termID := "", ""
		if agent.TerminalID != nil {
			termID = *agent.TerminalID
			if st, ok := deps.TermStates.Get(termID); ok {
				state = st.State
			}
		}
		modes := []string{deliveryPrompt}
		if deps.runMode(r, agent.ID) == modeInteractive {
			modes = append(modes, deliverySteer, deliveryFollowUp, deliveryInterrupt)
		}
		writeJSON(w, http.StatusOK, map[string]any{"cli": "pi", "modes": modes, "state": state, "termId": termID})
	}
}

// interruptMarker matches the line each measured CLI prints when a turn
// stops: "Interrupted" (Claude Code, Muse, Antigravity, OpenCode),
// "Conversation interrupted" (Codex), "Operation aborted" (Pi), "Command
// aborted" (Omp), "Turn cancelled" (Grok), "Operation interrupted" (Hermes).
var interruptMarker = regexp.MustCompile(`(?i)\b(interrupted|aborted|cancelled)\b`)

var (
	interruptKeyGap    = 150 * time.Millisecond
	interruptSettle    = 3 * time.Second
	interruptPollEvery = 150 * time.Millisecond
)

// doorDeliverInterrupt stops the running turn and sends the message as a
// new prompt. Idle CLIs skip the stop (every CLI sends at once when idle);
// needs-you and a recognized draft are refused; the stop must be seen —
// the CLI's own interrupted line, or its state leaving working — before
// anything is pasted, and a composer the stop refilled (Muse puts a prompt
// it retracted back in the field) is refused rather than appended to.
func doorDeliverInterrupt(deps Deps, ctx context.Context, t store.Terminal, cli, payload string) (int, map[string]any) {
	keys := deliveryAdapters[cli].interrupt
	if len(keys) == 0 {
		return http.StatusConflict, map[string]any{
			"error":  "Stop and send is not available for this CLI.",
			"reason": "unsupported-mode", "modes": deliveryModesFor(cli),
		}
	}
	st, ok := deps.TermStates.Get(t.ID)
	if ok && st.State == TermNeedsYou {
		return http.StatusConflict, workingRefusal(cli, st.State)
	}
	if !ok || st.State != TermWorking {
		return doorDeliver(deps, ctx, t, payload)
	}
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return http.StatusServiceUnavailable, map[string]any{"error": "Need tmux to send to a terminal."}
	}
	session := tmux.ShellSessionName(t.ID)
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	has, err := deps.Tmux.HasSession(cctx, session)
	if err != nil {
		return http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again."}
	}
	if !has {
		return http.StatusConflict, map[string]any{"error": "Start the terminal first.", "reason": "closed"}
	}
	if !tryLockPrompt(t.ID) {
		return http.StatusConflict, map[string]any{"error": "Already sending to this terminal.", "reason": "busy"}
	}
	defer unlockPrompt(t.ID)
	if st, ok := deps.TermStates.Get(t.ID); ok && st.State == TermNeedsYou {
		return http.StatusConflict, workingRefusal(cli, st.State)
	}
	before, err := deps.Tmux.InputSnapshot(cctx, session)
	if err != nil {
		return http.StatusConflict, map[string]any{"error": "PiCode could not read this terminal, so nothing was sent.", "reason": "unobservable"}
	}
	if doorReaderCLI[cli] {
		if state, known := peerComposerState(cli, before); known && state == "occupied" {
			return http.StatusConflict, map[string]any{"error": "Finish or clear the draft in the terminal first.", "reason": "occupied"}
		}
	}
	for i, k := range keys {
		if i > 0 {
			time.Sleep(interruptKeyGap)
		}
		if err := deps.Tmux.SendPaneKey(ctx, before.PaneID, k); err != nil {
			return http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again.", "reason": "closed"}
		}
	}
	after, stopped := awaitInterrupt(deps, cctx, t.ID, session, before)
	if !stopped {
		announceDoorPrompt(deps, t.ID, "unconfirmed")
		return http.StatusConflict, map[string]any{
			"error":  "PiCode asked the CLI to stop but could not see it stop, so the message was not sent. Check the terminal.",
			"reason": "not-stopped",
		}
	}
	if composerRefilled(before, after) {
		return http.StatusConflict, map[string]any{
			"error":  "Stopped. The CLI put the interrupted message back in its field — clear it in the terminal, then send again.",
			"reason": "restored",
		}
	}
	return doorPasteVerified(deps, ctx, cctx, t, cli, session, payload, false)
}

// awaitInterrupt polls until the pane shows a new interrupted line or the
// terminal's state leaves working, within interruptSettle.
func awaitInterrupt(deps Deps, ctx context.Context, termID, session string, before tmux.InputSnapshot) (tmux.InputSnapshot, bool) {
	deadline := time.Now().Add(interruptSettle)
	last := before
	for time.Now().Before(deadline) {
		time.Sleep(interruptPollEvery)
		snap, err := deps.Tmux.InputSnapshot(ctx, session)
		if err != nil || snap.PaneID != before.PaneID {
			return last, false
		}
		last = snap
		if markerRows(snap) > markerRows(before) {
			return snap, true
		}
		if st, ok := deps.TermStates.Get(termID); ok && st.State != TermWorking {
			return snap, true
		}
	}
	return last, false
}

func markerRows(s tmux.InputSnapshot) int {
	n := 0
	for _, raw := range s.Lines {
		if interruptMarker.MatchString(terminalSGR.ReplaceAllString(raw, "")) {
			n++
		}
	}
	return n
}

// composerRefilled: the input row now holds text it did not hold before the
// stop. Prompt glyphs, frame characters and dim or italic placeholders do
// not count; anything else is the CLI giving back a message.
func composerRefilled(before, after tmux.InputSnapshot) bool {
	a := composerText(after)
	return a != "" && a != composerText(before)
}

// placeholderSGR spans dim (2) or italic (3) text up to the next reset —
// the suggestion rows CLIs paint into an empty field (Hermes: italic, then
// a colour, then the words).
var placeholderSGR = regexp.MustCompile(`\x1b\[(?:[0-9]+;)*[23](?:;[0-9]+)*m.*?(?:\x1b\[(?:0|22|23)?m|$)`)

func composerText(s tmux.InputSnapshot) string {
	if s.CursorY < 0 || s.CursorY >= len(s.Lines) {
		return ""
	}
	raw := placeholderSGR.ReplaceAllString(s.Lines[s.CursorY], "")
	line := terminalSGR.ReplaceAllString(raw, "")
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(line), "❯›>│┃|$ \u00a0"))
}
