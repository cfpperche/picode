package push

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// Presence is what the notifier needs from internal/presence: is a browser
// on the host machine alive right now? If so, the user is at the desk and
// the phone stays quiet (Claude Code Remote Control's own rule).
type Presence interface {
	AnyHostOnline() bool
}

// Notifier decides and sends. Decision table (each row: condition → action):
//
//	host device online                         → skip everything
//	no subscriptions                            → skip
//	inbox item, kind result, prefs.finished     → push "finished", tag result:<agent>
//	inbox item, blocking, prefs.actions         → push "needs you", tag inbox:<id>
//	inbox item, non-blocking fyi/question       → skip (the badge is enough)
//	live dialog (agent waiting), prefs.actions  → push "needs you", tag ask:<agent>
type Notifier struct {
	Store    *store.Store
	Sender   *Sender
	Presence Presence
	Log      *log.Logger
	// Clock/Send hooks for tests.
	send func(ctx context.Context, t Target, payload []byte, ttl int, urgency, topic string) error
}

func (n *Notifier) logf(format string, args ...any) {
	if n.Log != nil {
		n.Log.Printf(format, args...)
	}
}

func (n *Notifier) doSend(ctx context.Context, t Target, payload []byte, ttl int, urgency, topic string) error {
	if n.send != nil {
		return n.send(ctx, t, payload, ttl, urgency, topic)
	}
	return n.Sender.Send(ctx, t, payload, ttl, urgency, topic)
}

// OnInbox is store.Store.OnInboxCreated.
func (n *Notifier) OnInbox(it store.InboxItem) {
	if n == nil {
		return
	}
	var msg Message
	var want string
	switch {
	case it.Kind == store.InboxResult:
		want = "finished"
		msg = Message{Title: it.Title, Body: clip(it.Body, 140), Hash: "#/inbox/" + it.ID, Tag: "result:" + it.SourceID, Urgency: "normal"}
		if it.SourceID == "" {
			msg.Tag = "result:" + it.ID
		}
	case it.Blocking:
		want = "actions"
		msg = Message{Title: it.Title, Body: clip(firstNonEmpty(it.Reason, it.Body), 140), Hash: "#/inbox/" + it.ID, Tag: "inbox:" + it.ID, Urgency: "high"}
	default:
		return
	}
	go n.deliver(want, msg)
}

// OnWaiting is rpc.Runtime.OnWaiting: a managed agent raised a dialog and
// nobody has its socket open.
func (n *Notifier) OnWaiting(agentID, agentName, title, message string) {
	if n == nil {
		return
	}
	name := firstNonEmpty(agentName, agentID)
	msg := Message{
		Title:   name + " needs you",
		Body:    clip(firstNonEmpty(title, message, "The agent is waiting on a decision."), 140),
		Hash:    "#/agent/" + agentID,
		Tag:     "ask:" + agentID,
		Urgency: "high",
	}
	go n.deliver("actions", msg)
}

// SendTest pushes a sample to one endpoint, presence or not.
func (n *Notifier) SendTest(ctx context.Context, endpoint string) error {
	if n == nil || n.Store == nil {
		return errors.New("push: not configured")
	}
	subs, err := n.Store.ListPushSubscriptions()
	if err != nil {
		return err
	}
	for _, s := range subs {
		if s.Endpoint != endpoint {
			continue
		}
		payload, _ := json.Marshal(Message{Title: "PiCode", Body: "Push works on this device.", Hash: "#/more/notifications", Tag: "test"})
		err := n.doSend(ctx, Target{Endpoint: s.Endpoint, P256dh: s.P256dh, Auth: s.Auth}, payload, 60, "high", "test")
		n.afterSend(s, err)
		return err
	}
	return errors.New("push: no such subscription")
}

// deliver fans one message out to every subscription whose prefs want it.
func (n *Notifier) deliver(want string, msg Message) {
	if n.Store == nil {
		return
	}
	if n.Presence != nil && n.Presence.AnyHostOnline() {
		return
	}
	subs, err := n.Store.ListPushSubscriptions()
	if err != nil || len(subs) == 0 {
		return
	}
	payload, _ := json.Marshal(msg)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	for _, s := range subs {
		if !s.Prefs.Wants(want) {
			continue
		}
		err := n.doSend(ctx, Target{Endpoint: s.Endpoint, P256dh: s.P256dh, Auth: s.Auth}, payload, 6*3600, msg.Urgency, msg.Tag)
		n.afterSend(s, err)
	}
}

func (n *Notifier) afterSend(s store.PushSubscription, err error) {
	switch {
	case err == nil:
		_ = n.Store.MarkPushOK(s.Endpoint)
	case errors.Is(err, ErrGone):
		n.logf("push: %s gone, dropping subscription", shortEndpoint(s.Endpoint))
		_ = n.Store.DeletePushSubscription(s.Endpoint)
	default:
		n.logf("push: %s: %v", shortEndpoint(s.Endpoint), err)
		_ = n.Store.MarkPushFailure(s.Endpoint)
	}
}

func clip(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func shortEndpoint(e string) string {
	if len(e) > 48 {
		return e[:48] + "…"
	}
	return e
}
