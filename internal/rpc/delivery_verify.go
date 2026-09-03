package rpc

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/session"
)

// awaitReplyInSession waits for the delivered reply to materialize as a
// user message in the agent's private session directory — the only
// honest proof pi took it. The RPC ack means "command received", while
// pi holds early prompts in an in-memory follow-up queue that does not
// survive its init/resume window (the silent loss of 2026-09-03, where
// a task read "delivered" and the session file never changed).
func (ma *ManagedAgent) awaitReplyInSession(needle string, grace time.Duration) bool {
	dir := session.AgentDir(ma.AgentID)
	deadline := time.Now().Add(grace)
	for {
		if dirHasUserText(dir, needle) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// replyNeedle reduces the delivered payload to the distinctive fragment
// a session file must contain: the human's own words, whitespace-
// normalized, tail-trimmed to a comparable length (the payload opens
// with a boilerplate prefix that adds nothing to the match).
func replyNeedle(payload string) string {
	norm := strings.Join(strings.Fields(payload), " ")
	r := []rune(norm)
	if len(r) > 64 {
		r = r[len(r)-64:]
	}
	return string(r)
}

// dirHasUserText scans .jsonl session files (tail of each) for a user
// message whose normalized text contains the needle.
func dirHasUserText(dir, needle string) bool {
	needle = normalizeSpace(needle)
	if needle == "" {
		return false
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		st, err := os.Stat(p)
		if err != nil || st.Size() == 0 {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		const tail = 256 * 1024
		buf := make([]byte, tail)
		if st.Size() > int64(tail) {
			_, _ = f.Seek(-int64(tail), io.SeekEnd)
		}
		n, _ := f.Read(buf)
		_ = f.Close()
		if chunkHasUserText(string(buf[:n]), needle) {
			return true
		}
	}
	return false
}

// chunkHasUserText parses each JSONL line and matches a user message
// whose text contains the needle (unescaped by the JSON decode — the
// raw line would hide quotes inside escaped strings).
func chunkHasUserText(chunk, needle string) bool {
	for _, ln := range strings.Split(chunk, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		var d struct {
			Type    string `json:"type"`
			Message struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal([]byte(ln), &d) != nil {
			continue
		}
		if d.Message.Role != "user" {
			continue
		}
		text := ""
		for _, c := range d.Message.Content {
			text += " " + c.Text
		}
		if strings.Contains(normalizeSpace(text), needle) {
			return true
		}
	}
	return false
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
