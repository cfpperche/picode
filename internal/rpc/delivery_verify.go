package rpc

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DeliveryBaseline records the last byte that existed before a delivery.
// Matching only bytes appended after this cursor prevents a repeated short
// reply from falsely proving a new turn (the live failure behind ADR-0059).
type deliveryCursor struct {
	offset int64
	info   os.FileInfo
}

type DeliveryBaseline struct {
	files map[string]deliveryCursor
	dirs  []string
}

// CaptureDeliveryBaseline snapshots JSONL files or directories. A new file
// starts at byte zero; a replaced/truncated file is also read from zero.
func CaptureDeliveryBaseline(paths ...string) DeliveryBaseline {
	out := DeliveryBaseline{files: map[string]deliveryCursor{}}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		st, err := os.Stat(p)
		if err == nil && !st.IsDir() {
			if strings.HasSuffix(p, ".jsonl") {
				out.files[p] = deliveryCursor{offset: st.Size(), info: st}
			}
			continue
		}
		if err != nil {
			if strings.HasSuffix(p, ".jsonl") {
				out.files[p] = deliveryCursor{}
			}
			continue
		}
		out.dirs = append(out.dirs, p)
		for path, info := range jsonlFiles(p) {
			out.files[path] = deliveryCursor{offset: info.Size(), info: info}
		}
	}
	return out
}

// AwaitUserMessageAfter waits for the full normalized payload to appear as a
// newly appended user message. RPC success only means the command was accepted;
// this is the durable delivery truth.
func AwaitUserMessageAfter(baseline DeliveryBaseline, payload string, grace time.Duration) bool {
	return AwaitUserMessageAfterContext(context.Background(), baseline, payload, grace)
}

// AwaitUserMessageAfterContext is cancellation-aware for user-controlled
// bursts; the ordinary delivery loop uses the background wrapper above.
func AwaitUserMessageAfterContext(ctx context.Context, baseline DeliveryBaseline, payload string, grace time.Duration) bool {
	if normalizeSpace(payload) == "" {
		return false
	}
	deadline := time.NewTimer(grace)
	defer deadline.Stop()
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		if baselineHasUserText(baseline, payload) {
			return true
		}
		select {
		case <-ctx.Done():
			// Close the scan/cancel race: the writer may have materialized the
			// row between the probe above and cancellation becoming visible.
			return UserMessageAfter(baseline, payload)
		case <-deadline.C:
			return UserMessageAfter(baseline, payload)
		case <-tick.C:
		}
	}
}

// UserMessageAfter performs one non-blocking durable-delivery probe. Callers
// use it after stopping a writer to settle cancellation races without ever
// acknowledging an in-memory RPC response as delivery.
func UserMessageAfter(baseline DeliveryBaseline, payload string) bool {
	return normalizeSpace(payload) != "" && baselineHasUserText(baseline, payload)
}

// awaitReplyInSession is the ordinary managed queue's private-dir wrapper.
func (ma *ManagedAgent) awaitReplyInSession(baseline DeliveryBaseline, payload string, grace time.Duration) bool {
	return AwaitUserMessageAfter(baseline, payload, grace)
}

func baselineHasUserText(baseline DeliveryBaseline, payload string) bool {
	needle := normalizeSpace(payload)
	files := make(map[string]deliveryCursor, len(baseline.files))
	for path, cursor := range baseline.files {
		files[path] = cursor
	}
	for _, dir := range baseline.dirs {
		for path := range jsonlFiles(dir) {
			if _, existed := files[path]; !existed {
				files[path] = deliveryCursor{}
			}
		}
	}
	for path, cursor := range files {
		st, err := os.Stat(path)
		if err != nil || st.IsDir() {
			continue
		}
		offset := cursor.offset
		if offset < 0 || st.Size() < offset || (cursor.info != nil && !os.SameFile(cursor.info, st)) {
			offset = 0
		}
		if st.Size() <= offset {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			_ = f.Close()
			continue
		}
		// The delivered user row is the first new session entry. Cover the
		// payload's worst-case JSON escaping plus framing while retaining a
		// useful floor for metadata; this stays bounded by the already-held
		// input rather than truncating a valid long reply into a false retry.
		appendedCap := int64(len(payload))*6 + 64*1024
		if appendedCap < 512*1024 {
			appendedCap = 512 * 1024
		}
		buf, _ := io.ReadAll(io.LimitReader(f, appendedCap))
		_ = f.Close()
		if chunkHasUserText(string(buf), needle) {
			return true
		}
	}
	return false
}

func jsonlFiles(dir string) map[string]os.FileInfo {
	files := map[string]os.FileInfo{}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return files
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if st, err := os.Stat(path); err == nil {
			files[path] = st
		}
	}
	return files
}

// chunkHasUserText parses each JSONL line and matches a user message whose
// normalized text is the full payload (unescaped by JSON decoding).
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
		if json.Unmarshal([]byte(ln), &d) != nil || d.Type != "message" || d.Message.Role != "user" {
			continue
		}
		text := ""
		for _, c := range d.Message.Content {
			text += " " + c.Text
		}
		if normalizeSpace(text) == needle {
			return true
		}
	}
	return false
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
