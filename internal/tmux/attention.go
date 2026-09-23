package tmux

import (
	"context"
	"crypto/rand"
	"errors"
	"strconv"
	"strings"
	"time"
)

// InputSnapshot pins a pane and cursor. Screen capture is only an input safety
// gate, never the authority for conversation identity or lifecycle state.
type InputSnapshot struct {
	PaneID           string
	PanePID          int
	Command          string
	CursorX, CursorY int
	Width            int
	InMode           bool
	Lines            []string
}

func (m *Manager) inputSnapshot(ctx context.Context, name string) (InputSnapshot, error) {
	var s InputSnapshot
	out, err := m.run(ctx, "display-message", "-p", "-t", name+":", "#{pane_id}|#{pane_pid}|#{pane_current_command}|#{cursor_x}|#{cursor_y}|#{pane_width}|#{pane_in_mode}")
	if err != nil {
		return s, err
	}
	fields := strings.Split(strings.TrimSpace(out), "|")
	if len(fields) != 7 {
		return s, errors.New("unreadable pane metadata")
	}
	s.PaneID = fields[0]
	s.Command = fields[2]
	for _, v := range []struct {
		p *int
		s string
	}{{&s.PanePID, fields[1]}, {&s.CursorX, fields[3]}, {&s.CursorY, fields[4]}, {&s.Width, fields[5]}} {
		n, e := strconv.Atoi(v.s)
		if e != nil {
			return s, e
		}
		*v.p = n
	}
	s.InMode = fields[6] != "0"
	out, err = m.run(ctx, "capture-pane", "-p", "-e", "-t", s.PaneID)
	s.Lines = strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	return s, err
}

// PasteOnly never presses Enter and uses a unique buffer across recipients.
func (m *Manager) pasteOnly(ctx context.Context, paneID, text string) error {
	if !strings.HasPrefix(paneID, "%") {
		return errors.New("exact pane required")
	}
	buffer := "picode-message-" + rand.Text()
	if _, err := m.runStdin(ctx, text, "load-buffer", "-b", buffer, "-"); err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		_, _ = m.run(cleanup, "delete-buffer", "-b", buffer)
	}()
	_, err := m.run(ctx, "paste-buffer", "-p", "-b", buffer, "-t", paneID)
	return err
}

// sendPaneKey presses one named key (Enter, Tab, M-Enter) on an exact pane:
// the attach door's mid-turn sequences (ADR-0206) — never a key list, so a
// caller cannot turn it into a control protocol.
func (m *Manager) sendPaneKey(ctx context.Context, paneID, key string) error {
	if !strings.HasPrefix(paneID, "%") {
		return errors.New("exact pane required")
	}
	_, err := m.run(ctx, "send-keys", "-t", paneID, key)
	return err
}

func (m *Manager) submitPane(ctx context.Context, paneID string) error {
	if !strings.HasPrefix(paneID, "%") {
		return errors.New("exact pane required")
	}
	_, err := m.run(ctx, "send-keys", "-t", paneID, "Enter")
	return err
}
