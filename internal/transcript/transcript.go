// Package transcript is the portable, CLI-neutral shape of a coding-agent
// conversation (ADR-0088). Each Agent CLI's reader projects its native
// session into an ordered Timeline; each writer emits a native session from
// one. The model is deliberately small — it carries what a second agent
// needs to continue the work (turns, tool calls and their results, the
// summaries left by compaction), never an agent's runtime: no signed
// reasoning, no hooks, no permissions, no provider payloads. Everything a
// reader cannot carry is counted in the Manifest so the user sees what the
// handoff left behind before anything is written.
package transcript

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Kind is the event kind of one Timeline beat.
type Kind string

const (
	KindMessage    Kind = "message"     // a user or assistant text turn
	KindToolCall   Kind = "tool_call"   // an assistant tool invocation
	KindToolResult Kind = "tool_result" // the result that answered a call
	KindThinking   Kind = "thinking"    // visible reasoning text; never re-emitted
	KindCompaction Kind = "compaction"  // a summary that replaced older history
	KindContext    Kind = "context"     // injected instructions; never re-emitted
)

// ToolCall is one tool invocation. Input is the call's arguments as JSON:
// an object when the source had one, otherwise a JSON string.
type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// ToolResult answers the ToolCall whose ID equals CallID. Name is the tool
// name when the source repeats it (pi, Hermes); otherwise readers copy it
// from the call. Text is the result flattened to text — images, structured
// details and provider blocks do not travel.
type ToolResult struct {
	CallID  string
	Name    string
	Text    string
	IsError bool
}

// Event is one beat of the conversation, in source order. Group is the
// ordinal of the source record (message) the beat came from: writers merge
// consecutive assistant beats of one group back into one native message so
// "text then tool call" stays one turn where the target expects it.
type Event struct {
	Kind      Kind
	Role      string // user | assistant | system — message and context only
	Text      string
	Call      *ToolCall
	Result    *ToolResult
	Model     string // the model that produced an assistant beat, when known
	Timestamp time.Time
	Group     int
}

// Header is what a reader knows about the session as a whole.
type Header struct {
	SourceCLI     string // catalog id: claude-code, codex, grok, pi, hermes
	SourceName    string // display name, filled by the server from the catalog
	SourceID      string
	SourcePath    string
	Cwd           string
	Title         string
	Provider      string
	Model         string
	FormatVersion string // the source CLI's own version marker, when it records one
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Manifest counts what did not travel. Dropped keys are short dotted names
// ("thinking", "claude.file-history-snapshot", "tool_result.orphan");
// Warnings are one-line, user-facing sentences.
type Manifest struct {
	Dropped  map[string]int `json:"dropped"`
	Warnings []string       `json:"warnings"`
}

// MarshalJSON renders empty maps and lists as {} and [] so API clients
// never see null where a collection is promised.
func (m Manifest) MarshalJSON() ([]byte, error) {
	type wire struct {
		Dropped  map[string]int `json:"dropped"`
		Warnings []string       `json:"warnings"`
	}
	w := wire{Dropped: m.Dropped, Warnings: m.Warnings}
	if w.Dropped == nil {
		w.Dropped = map[string]int{}
	}
	if w.Warnings == nil {
		w.Warnings = []string{}
	}
	return json.Marshal(w)
}

// Drop counts one omission of kind.
func (m *Manifest) Drop(kind string) {
	if m.Dropped == nil {
		m.Dropped = map[string]int{}
	}
	m.Dropped[kind]++
}

// Warn appends one user-facing warning.
func (m *Manifest) Warn(format string, a ...any) {
	m.Warnings = append(m.Warnings, fmt.Sprintf(format, a...))
}

// clone copies the manifest so a derived timeline never mutates its
// source's counters through the shared map.
func (m Manifest) clone() Manifest {
	out := Manifest{Warnings: append([]string(nil), m.Warnings...)}
	if m.Dropped != nil {
		out.Dropped = make(map[string]int, len(m.Dropped))
		for k, v := range m.Dropped {
			out.Dropped[k] = v
		}
	}
	return out
}

// Total is the number of dropped items across every kind.
func (m Manifest) Total() int {
	n := 0
	for _, c := range m.Dropped {
		n += c
	}
	return n
}

// Kinds lists the dropped kinds, most frequent first.
func (m Manifest) Kinds() []string {
	out := make([]string, 0, len(m.Dropped))
	for k := range m.Dropped {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if m.Dropped[out[i]] != m.Dropped[out[j]] {
			return m.Dropped[out[i]] > m.Dropped[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}

// Counts summarizes a Timeline by kind.
type Counts struct {
	Messages    int `json:"messages"`
	ToolCalls   int `json:"toolCalls"`
	ToolResults int `json:"toolResults"`
	Thinking    int `json:"thinking"`
	Compactions int `json:"compactions"`
	Context     int `json:"context"`
}

// Timeline is one session in portable form.
type Timeline struct {
	Header   Header
	Events   []Event
	Manifest Manifest
}

// Counts tallies the events by kind.
func (t Timeline) Counts() Counts {
	var c Counts
	for _, e := range t.Events {
		switch e.Kind {
		case KindMessage:
			c.Messages++
		case KindToolCall:
			c.ToolCalls++
		case KindToolResult:
			c.ToolResults++
		case KindThinking:
			c.Thinking++
		case KindCompaction:
			c.Compactions++
		case KindContext:
			c.Context++
		}
	}
	return c
}

// Window cuts the timeline the way the source agent replays it: recent
// keeps only what follows the LAST compaction (the older turns live inside
// that compaction's summary, which is returned) — the rule
// session.TranscriptWindow applies to pi's own chat. recent=false keeps
// everything and returns no summary. Compaction events themselves are
// removed in both cases: writers fold the summary into text.
func (t Timeline) Window(recent bool) (Timeline, string) {
	out := Timeline{Header: t.Header, Manifest: t.Manifest.clone()}
	last := -1
	for i, e := range t.Events {
		if e.Kind == KindCompaction {
			last = i
		}
	}
	summary := ""
	start := 0
	if recent && last >= 0 {
		summary = strings.TrimSpace(t.Events[last].Text)
		start = last + 1
	}
	out.Events = make([]Event, 0, len(t.Events)-start)
	for _, e := range t.Events[start:] {
		if e.Kind == KindCompaction {
			continue
		}
		out.Events = append(out.Events, e)
	}
	return out, summary
}

// HasCompaction reports whether the timeline carries at least one
// compaction summary — when it does, the user can choose the recent window.
func (t Timeline) HasCompaction() bool {
	for _, e := range t.Events {
		if e.Kind == KindCompaction {
			return true
		}
	}
	return false
}

// Repair makes the tool history well-formed for a strict target: every
// call is answered, in order, by a result with the same ID, placed before
// the next message. A result the source recorded later than the next beat
// is hoisted up next to its call; a result with no call is dropped and
// counted; a call the source never answered gets a synthesized error
// result, so a target that validates pairing accepts the file. Idempotent.
func (t Timeline) Repair() Timeline {
	out := Timeline{Header: t.Header, Manifest: t.Manifest.clone()}
	calls := map[string]bool{}
	for _, e := range t.Events {
		if e.Kind == KindToolCall && e.Call != nil {
			calls[e.Call.ID] = true
		}
	}
	// First real result per call id, so a late one can be hoisted.
	results := map[string]*ToolResult{}
	for _, e := range t.Events {
		if e.Kind == KindToolResult && e.Result != nil && calls[e.Result.CallID] && results[e.Result.CallID] == nil {
			results[e.Result.CallID] = e.Result
		}
	}
	delivered := map[string]bool{}
	var pending []*ToolCall // emitted, not yet answered, in call order
	deliver := func(c *ToolCall) {
		if r := results[c.ID]; r != nil {
			out.Events = append(out.Events, Event{Kind: KindToolResult, Result: r, Group: -1})
		} else {
			out.Events = append(out.Events, Event{Kind: KindToolResult, Result: &ToolResult{CallID: c.ID, Name: c.Name, Text: "[result not carried over]", IsError: true}, Group: -1})
			out.Manifest.Drop("tool_result.synthesized")
		}
		delivered[c.ID] = true
	}
	flush := func() {
		for _, c := range pending {
			deliver(c)
		}
		pending = nil
	}
	for _, e := range t.Events {
		switch e.Kind {
		case KindToolCall:
			if e.Call == nil {
				continue
			}
			out.Events = append(out.Events, e)
			pending = append(pending, e.Call)
		case KindToolResult:
			if e.Result == nil || !calls[e.Result.CallID] {
				out.Manifest.Drop("tool_result.orphan")
				continue
			}
			if delivered[e.Result.CallID] {
				continue // hoisted next to its call already, or a duplicate answer
			}
			// Answer every call issued before this one first, then this
			// one, so each call keeps its result adjacent and in order. A
			// result recorded before its call is placed when the call comes.
			var rest []*ToolCall
			found := false
			for _, c := range pending {
				if found {
					rest = append(rest, c)
					continue
				}
				deliver(c)
				found = c.ID == e.Result.CallID
			}
			pending = rest
		case KindThinking, KindContext:
			out.Events = append(out.Events, e)
		default:
			flush()
			out.Events = append(out.Events, e)
		}
	}
	flush()
	return out
}

// Prepare is what every writer starts from: Repair, then strip the beats
// no target receives (thinking, injected context), counting each. A
// compaction that still sits in the timeline (the caller chose the whole
// conversation, or skipped Window) becomes a user message carrying its
// summary — a summary is never silently lost.
func (t Timeline) Prepare() Timeline {
	r := t.Repair()
	out := Timeline{Header: r.Header, Manifest: r.Manifest.clone()}
	for _, e := range r.Events {
		switch e.Kind {
		case KindThinking:
			out.Manifest.Drop("thinking")
		case KindContext:
			out.Manifest.Drop("context")
		case KindCompaction:
			out.Events = append(out.Events, Event{Kind: KindMessage, Role: "user", Text: "Earlier part of the conversation, as summarized by the previous agent:\n" + strings.TrimSpace(e.Text), Timestamp: e.Timestamp, Group: e.Group})
		default:
			out.Events = append(out.Events, e)
		}
	}
	return out
}

// LastUserText is the text of the newest user message, or "".
func (t Timeline) LastUserText() string {
	for i := len(t.Events) - 1; i >= 0; i-- {
		e := t.Events[i]
		if e.Kind == KindMessage && e.Role == "user" && strings.TrimSpace(e.Text) != "" {
			return strings.TrimSpace(e.Text)
		}
	}
	return ""
}

// LastAssistantText is the text of the newest assistant message, or "".
func (t Timeline) LastAssistantText() string {
	for i := len(t.Events) - 1; i >= 0; i-- {
		e := t.Events[i]
		if e.Kind == KindMessage && e.Role == "assistant" && strings.TrimSpace(e.Text) != "" {
			return strings.TrimSpace(e.Text)
		}
	}
	return ""
}

// HandoffNote is the first user message of every native handoff: it tells
// the receiving agent where the history came from and that the world may
// have moved since. summary is the compaction summary the window dropped
// behind (empty when the whole conversation travels).
func HandoffNote(h Header, summary string, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Handoff from %s, session %s, %s, folder %s.\n", h.DisplayName(), ShortID(h.SourceID), now.UTC().Format("2006-01-02 15:04 MST"), h.Cwd)
	b.WriteString("PiCode translated this conversation from another coding agent. Everything above the last user message is history: files may have changed since, and the tools named in it are that agent's, not yours. Continue from the user's last request; re-read files before editing.")
	if s := strings.TrimSpace(summary); s != "" {
		b.WriteString("\n\nEarlier part of the conversation, as summarized by the previous agent:\n")
		b.WriteString(s)
	}
	return b.String()
}

// DisplayName is the source's display name, falling back to its id.
func (h Header) DisplayName() string {
	if h.SourceName != "" {
		return h.SourceName
	}
	if h.SourceCLI != "" {
		return h.SourceCLI
	}
	return "another agent"
}

// NewID is an RFC 4122 version-4 UUID from crypto/rand, the id shape every
// supported CLI accepts for a session.
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("transcript: crypto/rand unavailable: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ShortID is the first eight characters of an id — enough for a human to
// match it against a listing.
func ShortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// RandomHex returns n random bytes as lowercase hex.
func RandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("transcript: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// RandomAlnum returns n random characters from [A-Za-z0-9], the alphabet
// vendor tool-call ids use.
func RandomAlnum(n int) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("transcript: crypto/rand unavailable: " + err.Error())
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

// InputObject returns the call's Input as an object when it is one, or nil
// when the source recorded a non-object (a raw string, a number).
func (c ToolCall) InputObject() map[string]any {
	if len(c.Input) == 0 {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(c.Input, &m) != nil {
		return nil
	}
	return m
}

// InputString renders the call's Input for a target that wants a string:
// an object marshals compactly, a JSON string unquotes, anything else is
// the raw JSON.
func (c ToolCall) InputString() string {
	if len(c.Input) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(c.Input, &s) == nil {
		return s
	}
	return string(c.Input)
}

// ObjectInput wraps a non-object Input so targets that require an object
// ({"input": <raw>}) still receive the original value.
func (c ToolCall) ObjectInput() json.RawMessage {
	if len(c.Input) == 0 {
		return json.RawMessage(`{}`)
	}
	if c.InputObject() != nil {
		return c.Input
	}
	wrapped, err := json.Marshal(map[string]json.RawMessage{"input": c.Input})
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return wrapped
}
