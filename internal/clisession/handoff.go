package clisession

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/transcript"
)

// Optional capabilities a Source may implement for cross-CLI session
// handoff (ADR-0088). They are discovered by type assertion on the
// registered Source — never by a switch on the CLI id — so a new CLI joins
// the handoff surface by implementing the interfaces in its own file, and
// a CLI that lacks one simply is not offered that mode.

// Ref addresses one session of one CLI: what a Sessions row carries.
type Ref struct {
	ID   string
	Path string
	Cwd  string
	// Tail, when the file is over MaxReadBytes, loads only the last
	// MaxReadBytes (a brief of the recent turns). Native reads leave it
	// false and get ErrTooLarge.
	Tail bool
	// Roots are extra session directories the caller vouches for, read
	// beside the CLI's own root. PiCode's per-agent Omp directories
	// (OmpAgentSessionsRoot) are the one use: a workspace agent's live
	// conversation lives there, not under ~/.omp.
	Roots []string
}

// Reader projects a native session into the portable timeline. It never
// writes, and it opens only the file the Ref names (after checking it sits
// under the CLI's own session root) — it does not re-list to find it.
type Reader interface {
	Read(ctx context.Context, ref Ref) (transcript.Timeline, error)
}

// Writer creates a NEW native session from a timeline so the target CLI
// lists and resumes it. Create-only: a writer never modifies an existing
// session, settings or credentials, and it re-reads its own output before
// returning (round-trip), so a file the CLI would reject is never left
// behind.
type Writer interface {
	Write(ctx context.Context, t transcript.Timeline, req WriteRequest) (Summary, error)
}

// Prompter composes the launch arguments that start the CLI with an
// initial prompt (the brief handoff mode). sessionID is a pre-assigned id
// for the new conversation; CLIs that cannot pre-assign one ignore it.
type Prompter interface {
	PromptArgs(prompt, sessionID string) []string
}

// Forker composes the launch arguments that open a copy of an existing
// session of the same CLI with the vendor's own fork (Fork agent…): the
// source stays as it is, the copy carries the whole conversation and
// starts on prompt when one is given (none leaves it waiting). newID
// pre-assigns the copy's id where the CLI allows it. Each form is the
// CLI's own, verified on its installed version: a fork never goes through
// a translated session.
type Forker interface {
	ForkArgs(src Ref, prompt, newID string) Fork
}

// SessionForker forks through the CLI's own program instead of launch
// flags, for a CLI whose fork is a protocol call (Muse Code's `muse serve`
// session/fork). start builds the CLI's command with its configured
// executable, environment and the session's folder (ADR-0094's runner);
// the returned Fork opens the copy that already exists.
type SessionForker interface {
	ForkSession(ctx context.Context, src Ref, start func(ctx context.Context, args ...string) *exec.Cmd) (Fork, error)
}

// AgentForker forks into the new agent's own session folder, for a CLI
// whose agent owns its conversation file (Pi: `--session` is the agent's,
// sessions live in its private folder, ADR-0040). The caller creates the
// agent, then asks for the copy in dir under newID before the first start,
// and pins the returned file as the agent's session: a restart reopens the
// copy instead of forking again. start builds the CLI's command in the
// copy's working folder.
type AgentForker interface {
	ForkIntoDir(ctx context.Context, src Ref, dir, newID string, start func(ctx context.Context, args ...string) *exec.Cmd) (string, error)
}

// LiveSessionFinder names the conversation a running TUI is writing when
// the CLI records it only at exit (Muse Code), so a fork can start from a
// terminal that has no pinned session yet: its id and the arguments that
// resume it. An empty id means none was found. taken reports sessions the
// caller knows belong elsewhere (pinned by another terminal, or a copy a
// fork made) — in a shared folder they are newer candidates, not this one.
type LiveSessionFinder interface {
	LiveSession(ctx context.Context, cwd string, since time.Time, taken func(id string) bool, start func(ctx context.Context, args ...string) *exec.Cmd) (id string, resumeArgs []string, err error)
}

// Fork is one composed fork launch. ID and ResumeArgs are set only when
// the copy's id is known before it starts (pre-assigned, or made by a
// SessionForker), so the new terminal can pin it at once. Otherwise both
// are empty and the terminal's pinned session resolves the copy on its
// first turn. TaskAfterLaunch marks a launch that cannot carry the task:
// the caller delivers it through the prompt door once the TUI is ready.
type Fork struct {
	Args            []string
	ID              string
	ResumeArgs      []string
	TaskAfterLaunch bool
}

// WriteRequest parameterizes one native write.
type WriteRequest struct {
	Cwd string
	// Dir is the directory to create the session in, for CLIs whose
	// session path is free (pi, launched with --session <path>). CLIs with
	// a fixed root ignore it.
	Dir string
	// FormatVersion is the installed target's version marker. "" lets the
	// writer probe the newest local artifact; when nothing is known the
	// write refuses with ErrUnknownFormat rather than guessing.
	FormatVersion string
	// SessionID pre-assigns the new session's id; "" allocates one.
	SessionID string
	// Tools is "native" (tool calls keep the target's own shape) or "text"
	// (calls and results become plain text turns).
	Tools string
	Now   time.Time
	// Run executes the target CLI itself, in the session's folder, with the
	// CLI's configured executable and environment (ADR-0094). Writers that
	// publish through the vendor's own import command need it; nil means
	// PiCode cannot run the CLI here, and such a writer refuses with
	// ErrNoRunner rather than writing the store behind the CLI's back.
	Run func(ctx context.Context, args ...string) ([]byte, error)
}

// Capabilities is what GET /api/clis advertises per CLI so the web derives
// the handoff targets from the server instead of a hardcoded list. Agent
// marks a CLI that is also the platform's managed agent, so a handoff can
// land as a stopped agent in the app instead of a terminal.
type Capabilities struct {
	List   bool `json:"list"`
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Prompt bool `json:"prompt"`
	Fork   bool `json:"fork"`
	Agent  bool `json:"agent"`
}

// CapabilitiesOf reports the session capabilities of one catalog CLI.
func CapabilitiesOf(cli string) Capabilities {
	src, ok := Get(cli)
	if !ok {
		return Capabilities{}
	}
	_, read := src.(Reader)
	_, write := src.(Writer)
	_, prompt := src.(Prompter)
	_, forker := src.(Forker)
	_, sessionForker := src.(SessionForker)
	_, agentForker := src.(AgentForker)
	fork := forker || sessionForker || agentForker
	return Capabilities{List: true, Read: read, Write: write, Prompt: prompt, Fork: fork}
}

// ReaderFor, WriterFor and PrompterFor are the typed lookups the server uses.
func ReaderFor(cli string) (Reader, bool) {
	src, ok := Get(cli)
	if !ok {
		return nil, false
	}
	r, ok := src.(Reader)
	return r, ok
}

func WriterFor(cli string) (Writer, bool) {
	src, ok := Get(cli)
	if !ok {
		return nil, false
	}
	w, ok := src.(Writer)
	return w, ok
}

func PrompterFor(cli string) (Prompter, bool) {
	src, ok := Get(cli)
	if !ok {
		return nil, false
	}
	p, ok := src.(Prompter)
	return p, ok
}

func AgentForkerFor(cli string) (AgentForker, bool) {
	src, ok := Get(cli)
	if !ok {
		return nil, false
	}
	f, ok := src.(AgentForker)
	return f, ok
}

func SessionForkerFor(cli string) (SessionForker, bool) {
	src, ok := Get(cli)
	if !ok {
		return nil, false
	}
	f, ok := src.(SessionForker)
	return f, ok
}

func ForkerFor(cli string) (Forker, bool) {
	src, ok := Get(cli)
	if !ok {
		return nil, false
	}
	f, ok := src.(Forker)
	return f, ok
}

// idFork is the shape Claude Code and Grok share: resume the source with
// the fork flag, name the copy with --session-id, and resume it later with
// --resume <id> like any of their sessions.
func idFork(args []string, prompt, newID string) Fork {
	if newID == "" {
		return Fork{Args: withPrompt(args, prompt)}
	}
	args = append(args, "--session-id", newID)
	return Fork{Args: withPrompt(args, prompt), ID: newID, ResumeArgs: []string{"--resume", newID}}
}

// withPrompt appends prompt as the last argument when there is one.
func withPrompt(args []string, prompt string) []string {
	if prompt == "" {
		return args
	}
	return append(args, prompt)
}

var (
	// ErrUnknownFormat: the target's session format version could not be
	// established, so a native write would be a guess.
	ErrUnknownFormat = errors.New("The installed CLI's session format could not be verified; use a brief instead.")
	// ErrNotUnderRoot: the Ref points outside the CLI's session root.
	ErrNotUnderRoot = errors.New("That session is not on this machine.")
	// ErrTooLarge: the source exceeds MaxReadBytes.
	ErrTooLarge = errors.New("That session is too large to translate; use a brief instead.")
	// ErrNoDir: a writer that needs a caller-chosen directory got none.
	ErrNoDir = errors.New("A directory for the new session is required.")
	// ErrNoRunner: a writer that publishes through the target's own import
	// command was given no way to run it.
	ErrNoRunner = errors.New("This CLI receives a session through its own import command, which is not available here.")
)

// MaxReadBytes caps what a reader loads: a native transcript beyond this
// is offered the brief mode only.
const MaxReadBytes = 64 << 20

// ToolsNative and ToolsText are the two WriteRequest.Tools values.
const (
	ToolsNative = "native"
	ToolsText   = "text"
)

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// underRoot reports whether path is inside root (both made absolute). It
// does not require a suffix; readers decide what kind of path they accept.
func underRoot(root, path string) bool {
	if root == "" || path == "" {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != "."
}

// scanLines feeds every non-empty line of a JSONL file to fn, refusing
// files over MaxReadBytes before reading them.
func scanLines(path string, fn func(line []byte)) error {
	return scanSessionLines(path, false, fn)
}

// scanSession is scanLines, or the last MaxReadBytes when ref.Tail is set
// and the file is over the cap.
func scanSession(path string, ref Ref, fn func(line []byte)) error {
	return scanSessionLines(path, ref.Tail, fn)
}

func scanSessionLines(path string, tail bool, fn func(line []byte)) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if st.Size() > MaxReadBytes {
		if !tail {
			return ErrTooLarge
		}
		return scanLineTail(path, MaxReadBytes, fn)
	}
	return scanLineFile(path, fn)
}

func scanLineFile(path string, fn func(line []byte)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return scanReader(f, fn)
}

func scanLineTail(path string, budget int64, fn func(line []byte)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	start := st.Size() - budget
	if start < 0 {
		start = 0
	}
	buf := make([]byte, st.Size()-start)
	n, err := f.ReadAt(buf, start)
	if err != nil && err != io.EOF {
		return err
	}
	buf = buf[:n]
	// Sparse oversize fixtures (and any NUL padding) would otherwise glue
	// the last real record onto a 64 MB "line" and drop it.
	if bytes.IndexByte(buf, 0) >= 0 {
		buf = bytes.ReplaceAll(buf, []byte{0}, []byte{'\n'})
	}
	const maxLine = 16 * 1024 * 1024
	for len(buf) > 0 {
		i := bytes.IndexByte(buf, '\n')
		var line []byte
		if i < 0 {
			line, buf = buf, nil
		} else {
			line, buf = buf[:i], buf[i+1:]
		}
		if len(line) == 0 || len(line) > maxLine {
			continue
		}
		fn(line)
	}
	return nil
}

func scanReader(r io.Reader, fn func(line []byte)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		fn(line)
	}
	return sc.Err()
}

// parseTime accepts RFC 3339 (with or without fraction) and the zone-less
// form Grok writes ("2026-09-06T21:49:50.724389980"), read as UTC.
func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.999999999", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// rfc3339 renders a time for Summary fields, "" for the zero time.
func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// textBlocks joins the text of a content value that is either a string or
// an array of typed blocks; other block types are counted on m under key.
func textBlocks(raw json.RawMessage, m *transcript.Manifest, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case "text", "input_text", "output_text", "summary_text":
			if t := strings.TrimSpace(b.Text); t != "" {
				parts = append(parts, t)
			}
		default:
			if m != nil && key != "" {
				m.Drop(key + "." + b.Type)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// argumentsJSON normalizes a tool's arguments: a JSON object stays as is;
// a string that parses as an object becomes that object; anything else is
// kept as a JSON string so no information is invented or lost.
func argumentsJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) == nil {
		return raw
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if json.Unmarshal([]byte(s), &obj) == nil && strings.HasPrefix(strings.TrimSpace(s), "{") {
			return json.RawMessage(s)
		}
		return raw
	}
	b, err := json.Marshal(string(raw))
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

// writeNewFile creates path atomically and never overwrites: the body goes
// to a sibling temp file, is synced, then renamed into place only if the
// name is still free. This is the create-only invariant of ADR-0088 in one
// place.
func writeNewFile(path string, body []byte) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("%s already exists", filepath.Base(path))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	cleanup := func() { _ = os.Remove(name) }
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		cleanup()
		return err
	}
	// Link then remove: link fails if the target exists, which rename
	// would silently replace.
	if err := os.Link(name, path); err != nil {
		cleanup()
		return err
	}
	cleanup()
	return nil
}

// roundTrip re-reads a freshly written session through the same package's
// reader and compares what came back with what was meant to be written.
// Thinking never travels, so it is excluded from the comparison.
func roundTrip(ctx context.Context, r Reader, ref Ref, want transcript.Counts) error {
	got, err := r.Read(ctx, ref)
	if err != nil {
		return fmt.Errorf("re-reading the written session failed: %w", err)
	}
	c := got.Counts()
	c.Thinking, want.Thinking = 0, 0
	c.Context, want.Context = 0, 0
	if c != want {
		return fmt.Errorf("the written session does not read back as written (messages %d/%d, tool calls %d/%d, results %d/%d)", c.Messages, want.Messages, c.ToolCalls, want.ToolCalls, c.ToolResults, want.ToolResults)
	}
	return nil
}

// toolCallText and toolResultText render tool activity as plain turns for
// the "text" tools mode and for targets without a native shape.
func toolCallText(c *transcript.ToolCall) string {
	if c == nil {
		return ""
	}
	args := c.InputString()
	if len(args) > 4000 {
		args = args[:4000] + "…"
	}
	return "Tool call `" + c.Name + "`:\n" + args
}

func toolResultText(r *transcript.ToolResult) string {
	if r == nil {
		return ""
	}
	text := r.Text
	if len(text) > 4000 {
		text = text[:4000] + "…"
	}
	head := "Result of `" + r.Name + "`"
	if r.Name == "" {
		head = "Tool result"
	}
	if r.IsError {
		head += " (error)"
	}
	return head + ":\n" + text
}

// title picks a session name for the target listing: the source title,
// else the first user line clipped.
func title(t transcript.Timeline) string {
	if s := strings.TrimSpace(t.Header.Title); s != "" {
		return clip(s, 120)
	}
	if s := preview(t); s != "" {
		return s
	}
	// Nothing of the source's own to name it after: the handoff note is
	// not a title, and a listing showing its first 120 characters is how
	// this was found.
	if name := strings.TrimSpace(t.Header.DisplayName()); name != "" && t.Header.SourceCLI != "" {
		return "From " + name
	}
	return ""
}

// preview is the first user line the source itself wrote, skipping the
// handoff note PiCode prepends. Used for Summary.Preview and, when the
// source recorded no title, to name the written session.
func preview(t transcript.Timeline) string {
	for _, e := range t.Events {
		if e.Kind == transcript.KindMessage && e.Role == "user" && strings.TrimSpace(e.Text) != "" && !strings.HasPrefix(e.Text, "Handoff from ") {
			return clip(e.Text, 120)
		}
	}
	return ""
}

// assistantTurn is one assistant message as a target writes it: the text
// and tool-call beats of one source Group, in order.
type assistantTurn struct {
	Model     string
	Timestamp time.Time
	Blocks    []transcript.Event // KindMessage (assistant) or KindToolCall
}

// walkTurns hands a prepared timeline to a writer turn by turn: user
// messages, assistant turns (consecutive assistant beats of one Group)
// and tool results, in source order. Repair() already placed each result
// right after the turn that issued its call.
func walkTurns(events []transcript.Event, user func(transcript.Event), assistant func(assistantTurn), result func(transcript.Event)) {
	for i := 0; i < len(events); {
		e := events[i]
		switch {
		case e.Kind == transcript.KindMessage && e.Role != "assistant":
			user(e)
			i++
		case e.Kind == transcript.KindToolResult:
			result(e)
			i++
		case e.Kind == transcript.KindMessage || e.Kind == transcript.KindToolCall:
			turn := assistantTurn{Model: e.Model, Timestamp: e.Timestamp}
			j := i
			for ; j < len(events); j++ {
				n := events[j]
				if n.Group != e.Group || !(n.Kind == transcript.KindToolCall || (n.Kind == transcript.KindMessage && n.Role == "assistant")) {
					break
				}
				if turn.Model == "" {
					turn.Model = n.Model
				}
				turn.Blocks = append(turn.Blocks, n)
			}
			assistant(turn)
			i = j
		default:
			i++ // thinking/context/compaction never reach a writer after Prepare
		}
	}
}

// stamp returns the event's own time or the fallback, monotonic per
// writer so files sort the way they were written.
func stamp(t time.Time, fallback time.Time) time.Time {
	if !t.IsZero() {
		return t
	}
	return fallback
}

// newestFile returns the most recently modified path among candidates.
func newestFile(paths []string) string {
	var best string
	var bestAt time.Time
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if best == "" || st.ModTime().After(bestAt) {
			best, bestAt = p, st.ModTime()
		}
	}
	return best
}

// marshalNative encodes a record the way the CLIs write their own files:
// compact, and without Go's HTML escaping, so a `<user_query>` or a
// `<system-reminder>` in the text reads as itself rather than as
// \u003c escapes. Both are valid JSON; matching the vendor's shape keeps
// a written session indistinguishable from one the CLI produced.
func marshalNative(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// jsonLine marshals one record and appends a newline.
func jsonLine(b *strings.Builder, v any) error {
	raw, err := marshalNative(v)
	if err != nil {
		return err
	}
	b.Write(raw)
	b.WriteByte('\n')
	return nil
}

// resolvedNow is req.Now or the wall clock.
func (r WriteRequest) resolvedNow() time.Time {
	if r.Now.IsZero() {
		return time.Now().UTC()
	}
	return r.Now.UTC()
}

// textTools reports the "text" tools mode.
func (r WriteRequest) textTools() bool { return r.Tools == ToolsText }
