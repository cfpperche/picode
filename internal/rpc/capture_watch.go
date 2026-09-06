package rpc

// Live browser-capture bridge (ADR-0080): while an `agent_browser` tool call
// runs, watch the sidecar's capture directory beside the pi session file and
// forward the newest bounded frame to the UI hub as a `capture_frame` event.
// The daemon owns no pixels: frames originate from the pi-browser-capture
// sidecar extension, are already bounded (JPEG ≤ 200 KiB, ≤ 1600 px per
// side), and are re-validated here before web fan-out.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/toolpreview"
)

const (
	capturePollInterval = 150 * time.Millisecond
	captureEndGrace     = 3 * time.Second
	captureMaxWatch     = 15 * time.Minute
)

type captureWatch struct {
	toolCallID string
	dir        string
	lastSeq    int
	finalSent  bool
	startedAt  time.Time
	endAt      time.Time
	stopped    bool
}

// captureState guards the watch table and the cached session file path.
type captureState struct {
	mu          sync.Mutex
	watches     map[string]*captureWatch
	sessionFile string
}

func newCaptureState() *captureState {
	return &captureState{watches: map[string]*captureWatch{}}
}

// startCaptureWatch arms a watcher for one agent_browser tool call. The
// session file path is resolved lazily (one get_state per process, cached);
// without it there is no capture directory to watch.
func (ma *ManagedAgent) startCaptureWatch(toolCallID string) {
	if ma.capture == nil || toolCallID == "" {
		return
	}
	cs := ma.capture
	cs.mu.Lock()
	if w, ok := cs.watches[toolCallID]; ok && !w.stopped {
		cs.mu.Unlock()
		return
	}
	sessionFile := cs.sessionFile
	cs.mu.Unlock()

	if sessionFile == "" {
		sessionFile = ma.resolveCaptureSessionFile()
		if sessionFile == "" {
			return
		}
		cs.mu.Lock()
		cs.sessionFile = sessionFile
		cs.mu.Unlock()
	}

	cs.mu.Lock()
	defer cs.mu.Unlock()
	if w, ok := cs.watches[toolCallID]; ok && !w.stopped {
		return
	}
	cs.watches[toolCallID] = &captureWatch{
		toolCallID: toolCallID,
		dir:        sessionFile + ".capture",
		startedAt:  time.Now(),
	}
	go ma.pumpCaptureFrames(cs, cs.watches[toolCallID])
}

// refreshCaptureSessionFile caches the live session file while the agent is
// idle: pi queues RPC commands behind a running turn, so asking during a turn
// times out. Called at spawn and after every settled turn.
func (ma *ManagedAgent) refreshCaptureSessionFile() {
	if ma.capture == nil {
		return
	}
	go func() {
		file := ma.resolveCaptureSessionFile()
		if file == "" {
			return
		}
		cs := ma.capture
		cs.mu.Lock()
		cs.sessionFile = file
		cs.mu.Unlock()
	}()
}

// resolveCaptureSessionFile asks the live pi process for its current session
// file. Failures stay silent: capture is optional decoration.
func (ma *ManagedAgent) resolveCaptureSessionFile() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := ma.GetState(ctx)
	if err != nil || !resp.Success {
		return ""
	}
	var data struct {
		SessionFile string `json:"sessionFile"`
	}
	if json.Unmarshal(resp.Data, &data) != nil || data.SessionFile == "" {
		return ""
	}
	return data.SessionFile
}

// markCaptureEnd notes a finished tool call so its watcher can retire after
// the grace window (the sidecar writes the final marker shortly after end).
func (ma *ManagedAgent) markCaptureEnd(toolCallID string) {
	if ma.capture == nil || toolCallID == "" {
		return
	}
	cs := ma.capture
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if w, ok := cs.watches[toolCallID]; ok {
		w.endAt = time.Now()
	}
}

// pumpCaptureFrames polls the capture directory until the final marker is
// forwarded, the grace window after tool end lapses, the watch ages out, or
// the agent process exits.
func (ma *ManagedAgent) pumpCaptureFrames(cs *captureState, w *captureWatch) {
	ticker := time.NewTicker(capturePollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ma.done:
			return
		case <-ticker.C:
		}
		cs.mu.Lock()
		retired := w.stopped
		if !retired {
			if time.Since(w.startedAt) > captureMaxWatch || (!w.endAt.IsZero() && time.Since(w.endAt) > captureEndGrace) {
				w.stopped = true
				retired = true
			}
		}
		stopNow := retired
		if !retired && w.finalSent {
			// The final frame went out; one more poll cycle is unnecessary.
			w.stopped = true
			delete(cs.watches, w.toolCallID)
			stopNow = true
		}
		cs.mu.Unlock()
		if stopNow {
			return
		}
		ma.forwardCaptureFrame(cs, w)
	}
}

// forwardCaptureFrame reads the sidecar's latest-wins files and broadcasts a
// frame when the sequence advanced. The final marker retires the watch.
func (ma *ManagedAgent) forwardCaptureFrame(cs *captureState, w *captureWatch) {
	raw, err := os.ReadFile(filepath.Join(w.dir, "current.json"))
	if err != nil {
		return
	}
	var meta struct {
		Seq    int    `json:"seq"`
		Final  bool   `json:"final"`
		TS     int64  `json:"ts"`
		URL    string `json:"url"`
		Title  string `json:"title"`
		Source string `json:"source"`
	}
	if json.Unmarshal(raw, &meta) != nil || meta.Seq <= 0 {
		return
	}

	cs.mu.Lock()
	newSeq := meta.Seq > w.lastSeq
	isFinal := meta.Final && !w.finalSent
	if !newSeq && !isFinal {
		cs.mu.Unlock()
		return
	}
	cs.mu.Unlock()

	image, err := os.ReadFile(filepath.Join(w.dir, "current.jpg"))
	if err != nil || len(image) == 0 {
		return
	}
	// Re-validate the frame before web fan-out (the sidecar bounds it too).
	dataURI := toolpreview.EncodeJPEG(image)
	if dataURI == "" {
		return
	}

	cs.mu.Lock()
	if meta.Seq > w.lastSeq {
		w.lastSeq = meta.Seq
	}
	if isFinal {
		w.finalSent = true
	}
	cs.mu.Unlock()
	frame, err := json.Marshal(map[string]any{
		"type":       "capture_frame",
		"toolCallId": w.toolCallID,
		"seq":        meta.Seq,
		"ts":         meta.TS,
		"final":      isFinal,
		"image":      dataURI,
		"url":        meta.URL,
		"title":      meta.Title,
		"source":     meta.Source,
	})
	if err != nil {
		return
	}
	// Same envelope shape as pumpEvents: agentId beside the event.
	ma.hub.Broadcast([]byte(`{"agentId":` + strconv.Quote(ma.AgentID) + `,"event":` + string(frame) + `}`))
}
