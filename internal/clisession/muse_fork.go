package clisession

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Muse Code forks through its own session protocol, not a launch flag:
// `muse serve` speaks MSP (JSON-RPC 2.0, one JSON object per line on
// stdio), and `session/fork` copies a session's completed turns into a new
// one whose `forkedFrom` names the source. Measured against Muse Code 1.3.0
// (2026-09-24): initialize → the `initialized` notification → session/fork
// answers in under a second, also while the source is open in a TUI; the
// copy opens with `muse resume <id>` (1.3.0 refuses the older `--resume`)
// and carries the source's history. `muse resume` takes no prompt, so the
// task travels after launch (TaskAfterLaunch).
func (MuseSource) ForkSession(ctx context.Context, src Ref, start func(ctx context.Context, args ...string) *exec.Cmd) (Fork, error) {
	if strings.TrimSpace(src.ID) == "" {
		return Fork{}, errors.New("Muse Code needs the session id to fork it.")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	c, _, done, err := openMSP(ctx, start)
	if err != nil {
		return Fork{}, err
	}
	defer done()
	res, err := c.call(ctx, "session/fork", map[string]any{"commandId": uuidV7(), "sessionId": src.ID, "excludeItems": true})
	if err != nil {
		return Fork{}, err
	}
	var out struct {
		Session struct {
			SessionID  string `json:"sessionId"`
			ForkedFrom *struct {
				SessionID string `json:"sessionId"`
			} `json:"forkedFrom"`
		} `json:"session"`
	}
	if err := json.Unmarshal(res, &out); err != nil || out.Session.SessionID == "" {
		return Fork{}, errors.New("Muse Code answered the fork without a new session.")
	}
	if out.Session.ForkedFrom == nil || out.Session.ForkedFrom.SessionID != src.ID {
		return Fork{}, errors.New("Muse Code's new session does not name this conversation as its source.")
	}
	id := out.Session.SessionID
	return Fork{Args: []string{"resume", id}, ID: id, ResumeArgs: []string{"resume", id}, TaskAfterLaunch: true}, nil
}

// LiveSession finds the conversation a running Muse TUI in cwd is writing.
// Muse indexes a session only when its TUI exits, so neither the index nor
// MSP session/list sees it while it runs (measured, 1.3.0). Its log
// directory exists from the first turn, though — <museHome>/sessions/
// YYYY/MM/DD/<session id> — so the directories touched since the TUI
// started are the candidates, and MSP session/read (no attach, no lease)
// says whose folder each one is. The newest one in cwd that is not taken
// wins; "" is none.
func (MuseSource) LiveSession(ctx context.Context, cwd string, since time.Time, taken func(id string) bool, start func(ctx context.Context, args ...string) *exec.Cmd) (string, []string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	c, home, done, err := openMSP(ctx, start)
	if err != nil {
		return "", nil, err
	}
	defer done()
	if home == "" {
		return "", nil, errors.New("Muse Code did not say where it keeps its sessions.")
	}
	type candidate struct {
		id string
		at time.Time
	}
	var found []candidate
	days, _ := filepath.Glob(filepath.Join(home, "sessions", "[0-9][0-9][0-9][0-9]", "[0-9][0-9]", "[0-9][0-9]", "*"))
	for _, dir := range days {
		st, err := os.Stat(dir)
		// Strictly after the TUI started: a session another run in this
		// folder flushed on its way out is older than this launch.
		if err != nil || !st.IsDir() || !st.ModTime().After(since) {
			continue
		}
		if taken != nil && taken(filepath.Base(dir)) {
			continue
		}
		found = append(found, candidate{filepath.Base(dir), st.ModTime()})
	}
	sort.Slice(found, func(i, j int) bool { return found[i].at.After(found[j].at) })
	want := filepath.Clean(cwd)
	for i, cand := range found {
		if i == 8 {
			break
		}
		res, err := c.call(ctx, "session/read", map[string]any{"sessionId": cand.id})
		if err != nil {
			continue
		}
		var out struct {
			Session struct {
				SessionID     string  `json:"sessionId"`
				WorkspaceRoot *string `json:"workspaceRoot"`
			} `json:"session"`
		}
		if json.Unmarshal(res, &out) != nil || out.Session.WorkspaceRoot == nil {
			continue
		}
		if filepath.Clean(*out.Session.WorkspaceRoot) == want {
			return out.Session.SessionID, []string{"resume", out.Session.SessionID}, nil
		}
	}
	return "", nil, nil
}

// openMSP starts `muse serve` and completes the MSP handshake: initialize
// (whose result names the muse home) and the `initialized` notification.
// done closes stdin — `muse serve` then flushes and exits — and waits.
func openMSP(ctx context.Context, start func(ctx context.Context, args ...string) *exec.Cmd) (*mspConn, string, func(), error) {
	cmd := start(ctx, "serve")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, "", nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, "", nil, fmt.Errorf("Muse Code could not start its session server: %w", err)
	}
	done := func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	}
	c := &mspConn{w: stdin, r: bufio.NewReaderSize(stdout, 64<<10)}
	res, err := c.call(ctx, "initialize", map[string]any{"clientInfo": map[string]string{"name": "picode", "version": "1"}})
	if err != nil {
		done()
		return nil, "", nil, err
	}
	var info struct {
		MuseHome string `json:"museHome"`
	}
	_ = json.Unmarshal(res, &info)
	if err := c.notify("initialized"); err != nil {
		done()
		return nil, "", nil, err
	}
	return c, info.MuseHome, done, nil
}

// mspConn is the few lines of MSP a fork needs: numbered requests, one
// notification, and skipping the server's own notifications while a
// response is awaited.
type mspConn struct {
	w    io.Writer
	r    *bufio.Reader
	next int
}

func (c *mspConn) notify(method string) error {
	return c.write(map[string]any{"jsonrpc": "2.0", "method": method})
}

func (c *mspConn) write(v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = c.w.Write(append(raw, '\n'))
	return err
}

func (c *mspConn) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	c.next++
	id := c.next
	if err := c.write(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return nil, fmt.Errorf("Muse Code's session server closed: %w", err)
	}
	for {
		if ctx.Err() != nil {
			return nil, errors.New("Muse Code's session server did not answer in time.")
		}
		line, err := c.r.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("Muse Code's session server closed before answering %s.", method)
		}
		var m struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(line, &m) != nil || m.ID == nil || *m.ID != id {
			continue // a notification, or a line that is not ours
		}
		if m.Error != nil {
			return nil, fmt.Errorf("Muse Code refused %s: %s", method, m.Error.Message)
		}
		return m.Result, nil
	}
}

// uuidV7 is the idempotency handle MSP commands carry (RFC 9562 v7: a
// millisecond timestamp, then random bits).
func uuidV7() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(time.Now().UnixMilli()))
	copy(b[0:6], ts[2:8])
	b[6] = (b[6] & 0x0f) | 0x70
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
