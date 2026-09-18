// Package browser is the daemon's half of the work-browser command channel
// (ADR-0132): the desktop shell opens a stream, the daemon pushes one command
// down it, and the shell's answer comes back over an authenticated POST.
//
// The hub carries no policy. The daemon resolves an agent's {tier, domains}
// before it calls Dispatch; the shell re-checks both (ADR-0128). Delivery
// cannot grant what the catalog refuses.
//
// Commands are not replayed and not persisted: a request lives only as long as
// the tool call waits for it (Timeout). A shell that reconnects rejoins by
// opening a new stream, which becomes the active one — the newest window wins,
// so a reconnect never leaves commands going to a stream the shell abandoned.
package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrNoShell: no desktop shell is on the line. The tool call fails fast
	// instead of hanging while a window is closed.
	ErrNoShell = errors.New("the desktop app is not connected")
	// ErrUnknownResult: a result whose command is gone (timed out, or never
	// existed). The shell gets a 404 for it.
	ErrUnknownResult = errors.New("no such command")
	// ErrBusy: the active stream is not keeping up (a shell wedged mid-command).
	ErrBusy = errors.New("the desktop app is not keeping up")
)

// DefaultTimeout bounds how long a tool call waits for the shell's answer.
const DefaultTimeout = 30 * time.Second

// backlog is how many commands may sit unread on one stream before the hub
// calls the shell busy.
const backlog = 8

// Command is what the shell executes. Tier and Domains travel with it so the
// shell can re-check the policy the daemon resolved: the tier gates the
// method catalog, the domains gate navigation.
type Command struct {
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	Tier    string          `json:"tier"`
	Domains []string        `json:"domains,omitempty"`
	// Raw marks a method the catalog did not name (ADR-0144): the shell lets
	// it past its own table only when its copy of the developer-mode setting
	// is on and the tier above is full.
	Raw bool `json:"raw,omitempty"`
	// Kind names the command family (ADR-0148). Empty is the work browser:
	// Method is a CDP method for the tab on screen. "computer" is a desktop
	// action the shell runs itself, with Method naming the action. The page
	// dispatches on it; a page that predates the field pushes every command
	// through the browser bridge, whose deny-by-default catalog refuses an
	// unknown method at once instead of hanging the call.
	Kind string `json:"kind,omitempty"`
	// Principal is the grant key (ADR-0143) the daemon resolved for the
	// caller, carried so the shell can keep per-principal state (the last
	// image it returned) without a second identity of its own.
	Principal string `json:"principal,omitempty"`
	// Timeout overrides the hub's wait for this one command; zero means the
	// hub's own. It never travels: the shell has its own clocks.
	Timeout time.Duration `json:"-"`
}

// Result is the shell's answer to one command. Error is the shell's refusal
// or failure in words (a tier refusal, a closed tab, a CDP error); the tool
// call surfaces it verbatim.
type Result struct {
	ID     string          `json:"id"`
	Output json.RawMessage `json:"output,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type conn struct {
	events chan Command
}

// Hub owns the connected shell streams and the in-flight commands.
type Hub struct {
	// Timeout is how long Dispatch waits for the shell. Zero means
	// DefaultTimeout (tests set it short).
	Timeout time.Duration

	mu      sync.Mutex
	nextID  int64
	active  *conn
	pending map[string]chan Result
}

func New() *Hub {
	return &Hub{pending: make(map[string]chan Result)}
}

// Attach registers a newly opened shell stream and makes it the active target.
// The returned channel carries commands; cancel releases the stream and, when
// it was the active one, leaves the line empty until the next Attach.
func (h *Hub) Attach() (<-chan Command, func()) {
	c := &conn{events: make(chan Command, backlog)}
	h.mu.Lock()
	h.active = c
	h.mu.Unlock()

	var once sync.Once
	return c.events, func() { once.Do(func() { h.detach(c) }) }
}

func (h *Hub) detach(c *conn) {
	h.mu.Lock()
	if h.active == c {
		h.active = nil
	}
	h.mu.Unlock()
}

// Connected reports whether a shell is on the line.
func (h *Hub) Connected() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.active != nil
}

// Dispatch sends one command to the active shell and waits for its result.
// It returns the shell's output JSON, the shell's error, or a transport error
// (no shell, busy, timeout, canceled context).
func (h *Hub) Dispatch(ctx context.Context, cmd Command) (json.RawMessage, error) {
	h.mu.Lock()
	c := h.active
	if c == nil {
		h.mu.Unlock()
		return nil, ErrNoShell
	}
	h.nextID++
	cmd.ID = fmt.Sprintf("%d-%d", time.Now().UnixNano(), h.nextID)
	answer := make(chan Result, 1)
	h.pending[cmd.ID] = answer
	h.mu.Unlock()

	select {
	case c.events <- cmd:
	default:
		h.forget(cmd.ID)
		return nil, ErrBusy
	}

	timeout := cmd.Timeout
	if timeout <= 0 {
		timeout = h.Timeout
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case res := <-answer:
		if res.Error != "" {
			return nil, errors.New(res.Error)
		}
		return res.Output, nil
	case <-ctx.Done():
		h.forget(cmd.ID)
		return nil, ctx.Err()
	case <-timer.C:
		h.forget(cmd.ID)
		return nil, fmt.Errorf("the desktop app did not answer in %s", timeout)
	}
}

func (h *Hub) forget(id string) {
	h.mu.Lock()
	delete(h.pending, id)
	h.mu.Unlock()
}

// Complete hands one result to the waiting command. A result for a command
// that already finished (a duplicate, or one that timed out) returns
// ErrUnknownResult so the shell can say so instead of it being swallowed.
func (h *Hub) Complete(res Result) error {
	h.mu.Lock()
	answer, ok := h.pending[res.ID]
	if ok {
		delete(h.pending, res.ID)
	}
	h.mu.Unlock()
	if !ok {
		return ErrUnknownResult
	}
	answer <- res
	return nil
}
