package clisession

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/transcript"
)

// Muse Code brief handoff (Fatia 1 of docs/plans/launch-muse-agy.md): a
// positional prompt starts the session with it. Verified against a real
// install (Muse Code 1.3.0, `muse --help` 2026-09-15): "pass a prompt to
// start a session". A session id is never pre-assigned — `resume` only
// reopens an existing uuid — so it is ignored, the way Codex's is.
func (MuseSource) PromptArgs(prompt, _ string) []string {
	return []string{prompt}
}

// MuseBin overrides the muse executable lookup (tests only). Production
// resolves it from PATH: the configured launch executable lives in the
// server store, which this package cannot see.
var MuseBin string

func museBinary() string {
	if MuseBin != "" {
		return MuseBin
	}
	p, err := exec.LookPath("muse")
	if err != nil {
		return ""
	}
	return p
}

// MuseSessionsRoot is the local session store the export resolves ids in.
// MuseSessionsRoot is the local session store the export resolves ids in,
// exported so a metric reads the same logs this package projects.
func MuseSessionsRoot() string { return museSessionsRoot() }

func museSessionsRoot() string { return filepath.Join(filepath.Dir(museDBPath()), "sessions") }

// Read projects a Muse session through the CLI's own exporter
// (`muse export --session <id|path>`, export_schema_version 1): user turns
// arrive as runtime.user_intent.accepted, assistant text as run events of
// kind assistant_message_committed, tool traffic as the commit batches, the
// model as model_completed. Reasoning is encrypted and never travels, so
// reasoning_committed is counted and dropped like thinking elsewhere. The
// export is one JSON document, so an over-cap session gets ErrTooLarge even
// with Tail — the caller falls back to the brief, which exists now that
// MuseSource is a Prompter.
func (MuseSource) Read(ctx context.Context, ref Ref) (transcript.Timeline, error) {
	arg := strings.TrimSpace(ref.Path)
	if arg == "" {
		arg = strings.TrimSpace(ref.ID)
	}
	if arg == "" {
		return transcript.Timeline{}, fmt.Errorf("a Muse session id or path is required")
	}
	if ref.Path != "" && !underRoot(museSessionsRoot(), arg) {
		return transcript.Timeline{}, ErrNotUnderRoot
	}
	bin := museBinary()
	if bin == "" {
		return transcript.Timeline{}, fmt.Errorf("the muse executable is not on PATH")
	}
	tmp, err := os.CreateTemp("", "muse-export-*.json")
	if err != nil {
		return transcript.Timeline{}, err
	}
	name := tmp.Name()
	tmp.Close()
	defer os.Remove(name)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "export", "--session", arg, "--out", name)
	cmd.Env = museExportEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		return transcript.Timeline{}, fmt.Errorf("muse export: %v: %s", err, clip(string(out), 300))
	}
	if st, err := os.Stat(name); err != nil {
		return transcript.Timeline{}, err
	} else if st.Size() > MaxReadBytes {
		return transcript.Timeline{}, ErrTooLarge
	}
	raw, err := os.ReadFile(name)
	if err != nil {
		return transcript.Timeline{}, err
	}
	return parseMuseExport(raw, ref)
}

// museExportEnv runs the exporter the way the setup check does: the
// launcher must not background-update while it reads.
func museExportEnv() []string {
	env := []string{"MUSE_NO_AUTO_UPDATE=1"}
	for _, kv := range os.Environ() {
		if k, _, ok := strings.Cut(kv, "="); ok && k != "MUSE_NO_AUTO_UPDATE" {
			env = append(env, kv)
		}
	}
	return env
}

type museEvent struct {
	Kind     string       `json:"kind"`
	Envelope museEnvelope `json:"envelope"`
}

type museEnvelope struct {
	RecordedAt  int64           `json:"recorded_at"`
	PayloadType string          `json:"payload_type"`
	Payload     json.RawMessage `json:"payload"`
	Children    []struct {
		RecordJSON string `json:"record_json"`
	} `json:"children"`
}

func parseMuseExport(raw []byte, ref Ref) (transcript.Timeline, error) {
	var doc struct {
		Schema int         `json:"export_schema_version"`
		Events []museEvent `json:"events"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return transcript.Timeline{}, fmt.Errorf("muse export: %v", err)
	}
	if doc.Schema != 1 {
		return transcript.Timeline{}, fmt.Errorf("muse export schema %d is not supported", doc.Schema)
	}
	return parseMuseExportEvents(doc.Events, ref)
}

func parseMuseExportEvents(events []museEvent, ref Ref) (transcript.Timeline, error) {
	t := transcript.Timeline{Header: transcript.Header{SourceCLI: "muse", SourceID: ref.ID, SourcePath: ref.Path, Cwd: ref.Cwd, FormatVersion: "export_schema_version 1"}}
	group := 0
	emit := func(ev transcript.Event, at int64) {
		if at > 0 {
			ev.Timestamp = time.Unix(0, at*1000).UTC()
		}
		ev.Group = group
		t.Events = append(t.Events, ev)
	}
	feed := func(env museEvent) {
		e := env.Envelope
		switch e.PayloadType {
		case "runtime.user_intent.accepted":
			var p struct {
				Messages []struct {
					Content []struct {
						Kind string `json:"kind"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"model_messages"`
			}
			if json.Unmarshal(e.Payload, &p) != nil {
				t.Manifest.Drop("muse.malformed")
				return
			}
			var b strings.Builder
			for _, m := range p.Messages {
				for _, c := range m.Content {
					if c.Kind == "text" && strings.TrimSpace(c.Text) != "" {
						if b.Len() > 0 {
							b.WriteString("\n")
						}
						b.WriteString(c.Text)
					}
				}
			}
			text := strings.TrimSpace(b.String())
			if text == "" {
				return
			}
			group++
			if t.Header.Title == "" {
				t.Header.Title = clip(text, 120)
			}
			emit(transcript.Event{Kind: transcript.KindMessage, Role: "user", Text: text}, e.RecordedAt)
		case "runtime.session":
			var p struct {
				Kind  string `json:"kind"`
				Event *struct {
					Kind   string `json:"kind"`
					Text   string `json:"text"`
					Prompt string `json:"prompt"`
					// Model is a string on model_completed and an object on
					// records that only assess one (automated_review_*), so it
					// decodes late, never at struct shape.
					Model json.RawMessage `json:"model"`
					Calls []struct {
						CallID string `json:"call_id"`
						Name   string `json:"name"`
						Args   string `json:"args"`
					} `json:"tool_calls"`
					Results []struct {
						CallID string `json:"tool_call_id"`
						Text   string `json:"text"`
					} `json:"results"`
				} `json:"event"`
				Record *struct {
					Workspace string `json:"workspace_root"`
					ModelID   string `json:"model_id"`
				} `json:"record"`
			}
			if json.Unmarshal(e.Payload, &p) != nil {
				t.Manifest.Drop("muse.malformed")
				return
			}
			switch p.Kind {
			case "metadata":
				if p.Record != nil {
					if t.Header.Cwd == "" {
						t.Header.Cwd = p.Record.Workspace
					}
					if t.Header.Model == "" {
						t.Header.Model = p.Record.ModelID
					}
				}
			case "run":
				if p.Event == nil {
					return
				}
				switch p.Event.Kind {
				case "assistant_message_committed":
					if strings.TrimSpace(p.Event.Text) == "" {
						return
					}
					emit(transcript.Event{Kind: transcript.KindMessage, Role: "assistant", Text: p.Event.Text, Model: t.Header.Model}, e.RecordedAt)
				case "assistant_tool_calls_committed":
					for _, c := range p.Event.Calls {
						emit(transcript.Event{Kind: transcript.KindToolCall, Call: &transcript.ToolCall{ID: c.CallID, Name: c.Name, Input: json.RawMessage(c.Args)}}, e.RecordedAt)
					}
				case "tool_result_batch_committed":
					for _, r := range p.Event.Results {
						emit(transcript.Event{Kind: transcript.KindToolResult, Result: &transcript.ToolResult{CallID: r.CallID, Text: r.Text}}, e.RecordedAt)
					}
				case "reasoning_committed":
					t.Manifest.Drop("thinking")
				case "model_completed":
					if t.Header.Model == "" {
						var model string
						if json.Unmarshal(p.Event.Model, &model) == nil {
							t.Header.Model = model
						}
					}
				}
			}
		}
	}
	for _, ev := range events {
		switch ev.Kind {
		case "record":
			feed(ev)
		case "gap", "retained_frame":
			t.Manifest.Drop("muse." + ev.Kind)
		default:
			t.Manifest.Drop("muse.unknown-kind")
		}
		for _, ch := range ev.Envelope.Children {
			var inner museEvent
			inner.Kind = "record"
			if json.Unmarshal([]byte(ch.RecordJSON), &inner.Envelope) != nil {
				t.Manifest.Drop("muse.malformed")
				continue
			}
			feed(inner)
		}
	}
	if t.Header.Cwd == "" {
		t.Header.Cwd = ref.Cwd
	}
	return t, nil
}
