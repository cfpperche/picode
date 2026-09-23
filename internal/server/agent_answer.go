package server

// A human's answer to an agent's question reaches the agent through the door
// that agent is actually listening on (2026-09-22). ADR-0184 made every CLI
// launch an agent, so questions from Claude Code, Codex and the rest arrive
// as agent-sourced items — and the agent path assumed Pi: it demanded the Pi
// session file the question was filed from, which an `ask_human` from any
// other CLI never carries, and refused every reply ("could not be identified
// safely"). The rule, first match wins:
//
//	asker polling the item      → record it; ask_human / `inbox ask --wait` read it
//	Pi, or Omp with a session   → ADR-0060: receiver or verified paste, JSONL proof
//	Pi not in a terminal        → the durable follow-up queue (unchanged)
//	other CLI, terminal running → record it and type it into the TUI
//	other CLI, not running      → record it; the note says nobody was told
//
// Recording first for a polling asker is what keeps a waiting agent from
// hearing the answer twice — once as the tool result, once as a typed turn.

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// askWaiterTTL is how long one poll of an item counts as "an asker is waiting
// on it". Pollers read every 2 seconds; the margin absorbs a slow daemon round
// trip without calling a departed asker present for long.
var askWaiterTTL = 30 * time.Second

// askPolled records that a waiting asker read this item just now.
func (t *TuiReplies) askPolled(itemID string) {
	if t == nil || strings.TrimSpace(itemID) == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.waiting == nil {
		t.waiting = map[string]time.Time{}
	}
	now := time.Now()
	t.waiting[itemID] = now
	for id, at := range t.waiting {
		if now.Sub(at) > askWaiterTTL {
			delete(t.waiting, id)
		}
	}
}

// askWaiting reports whether an asker polled this item recently.
func (t *TuiReplies) askWaiting(itemID string) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	at, ok := t.waiting[itemID]
	return ok && time.Since(at) <= askWaiterTTL
}

// agentAnswer is the door an answer took; the Inbox turns it into a toast.
type agentAnswer string

const (
	answerRead      agentAnswer = "read"      // a waiting asker reads the recorded answer
	answerDelivered agentAnswer = "delivered" // ADR-0060: receiver or paste, JSONL proof
	answerForwarded agentAnswer = "forwarded" // Pi's durable follow-up queue
	answerTyped     agentAnswer = "typed"     // typed into a non-Pi TUI, submit verified
	answerUntold    agentAnswer = "untold"    // recorded; no running terminal was told
)

// Toast is the one line the human sees after answering.
func (a agentAnswer) Toast() string {
	switch a {
	case answerRead:
		return "Answer sent — the agent was waiting for it."
	case answerDelivered:
		return "Reply sent to the terminal."
	case answerForwarded:
		return "Reply sent — the agent will pick it up."
	case answerTyped:
		return "Reply typed into the agent's terminal."
	default:
		return "Answer saved on the item — the agent was not told. Tell it in its terminal."
	}
}

// Notes appended to the item when an answer is recorded but no terminal took
// it, so the item itself tells the truth later.
const (
	answerNotRunningNote = "Answer recorded on the item. The agent was not running and nothing was waiting for it, so it was not told — tell it when it starts."
	answerNotTypedNote   = "Answer recorded on the item, but PiCode could not type it into the agent's terminal (%s). Tell the agent there."
)

// errAnswerInvalid marks an answer the item itself refuses (a done item, a
// verb it does not allow) — the caller's mistake, not a delivery failure.
var errAnswerInvalid = errors.New("invalid answer")

// receiverCLI reports whether this agent's TUI loads PiCode's reply receiver
// (ADR-0060): Pi, and Omp since it shares Pi's extension API.
func receiverCLI(a store.Agent) bool {
	return a.IsPi() || strings.EqualFold(strings.TrimSpace(a.CLI), "omp")
}

// AnswerAgentQuestion answers an agent-sourced question or approval by the
// rule at the top of this file. Ignore never comes here: it sends nothing.
func (deps Deps) AnswerAgentQuestion(ctx context.Context, itemID, verb, text string) (agentAnswer, error) {
	it, err := deps.Store.GetInboxItem(itemID)
	if err != nil {
		return "", err
	}
	if it.SourceKind != store.InboxFromAgent || strings.TrimSpace(it.SourceID) == "" {
		return "", fmt.Errorf("this item has no agent")
	}
	if it.State == store.InboxDone {
		return "", fmt.Errorf("item is already done: %w", errAnswerInvalid)
	}
	if !slices.Contains(it.Allowed, verb) {
		return "", fmt.Errorf("response %q is not allowed on this item: %w", verb, errAnswerInvalid)
	}
	agent, err := deps.Store.GetAgent(it.SourceID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			_ = deps.Store.AnnotateInboxItem(itemID, "Reply could not be delivered: agent no longer exists.")
			return "", fmt.Errorf("agent no longer exists: %w", store.ErrNotFound)
		}
		return "", err
	}
	if deps.Replies.askWaiting(itemID) {
		if _, err := deps.Store.RespondInboxItem(itemID, verb, text); err != nil {
			return "", err
		}
		return answerRead, nil
	}
	interactive := deps.agentInteractive(ctx, agent.ID)
	if agent.IsPi() || (receiverCLI(agent) && strings.TrimSpace(it.SessionPath) != "" && interactive) {
		if interactive {
			if _, err := deps.DeliverReply(ctx, itemID, verb, text); err != nil {
				return "", err
			}
			return answerDelivered, nil
		}
		deliverable := func(id string) bool { return !deps.agentInteractive(ctx, id) }
		if _, err := deps.Store.RespondAndForward(itemID, verb, text, deliverable); err != nil {
			return "", err
		}
		return answerForwarded, nil
	}
	if !interactive {
		if _, err := deps.Store.RespondInboxItem(itemID, verb, text); err != nil {
			return "", err
		}
		_ = deps.Store.AnnotateInboxItem(itemID, answerNotRunningNote)
		return answerUntold, nil
	}
	return deps.typeAnswer(ctx, agent, it, verb, text)
}

// typeAnswer records the answer, then types it into a running non-Pi TUI
// with a bracketed paste and a verified Enter. The record comes first so a
// second click finds the item done instead of typing the answer twice; a
// paste that does not land leaves the answer on the item with a note.
func (deps Deps) typeAnswer(ctx context.Context, agent store.Agent, it store.InboxItem, verb, text string) (agentAnswer, error) {
	if err := deps.Replies.Controls.check(agent.ID); err != nil {
		return "", err
	}
	deps.Replies.mu.Lock()
	if deps.Replies.active[agent.ID] {
		deps.Replies.mu.Unlock()
		return "", errors.New("This agent already has a reply on its way to the terminal. Try again in a moment.")
	}
	deps.Replies.active[agent.ID] = true
	deps.Replies.mu.Unlock()
	defer func() {
		deps.Replies.mu.Lock()
		delete(deps.Replies.active, agent.ID)
		deps.Replies.mu.Unlock()
	}()

	if _, err := deps.Store.RespondInboxItem(it.ID, verb, text); err != nil {
		return "", err
	}
	session := deps.agentSession(agent.ID)
	reason := ""
	if err := deps.Tmux.PasteText(ctx, session, store.InboxForwardPayload(it, verb, text)); err != nil {
		reason = err.Error()
	} else if !deps.verifyTUISubmit(ctx, agent.ID, session) {
		reason = "the text stayed in the composer"
	}
	if reason != "" {
		_ = deps.Store.AnnotateInboxItem(it.ID, fmt.Sprintf(answerNotTypedNote, reason))
		return answerUntold, nil
	}
	return answerTyped, nil
}
