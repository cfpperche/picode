package transcript

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// BriefMaxBytes bounds a brief so it fits comfortably in any target's
// first turn; sections are trimmed from the middle before the cap bites.
const BriefMaxBytes = 16 * 1024

// Brief renders a deterministic markdown summary of the timeline for the
// fallback handoff mode: the target reads it as its first task. No model
// is involved — the same timeline always yields the same bytes, so the
// brief is testable and free.
//
// summary is the compaction summary the recent window dropped behind, ""
// when the whole conversation is described. Sections are always present;
// an empty one says "none" so the reader never wonders whether a section
// was cut.
func Brief(t Timeline, summary string, now time.Time) string {
	t = t.Repair()
	var b strings.Builder
	fmt.Fprintf(&b, "# Handoff from %s · %s · %s\n\n", t.Header.DisplayName(), now.UTC().Format("2006-01-02"), t.Header.Cwd)
	c := t.Counts()
	fmt.Fprintf(&b, "Source session %s (%d messages", ShortID(t.Header.SourceID), c.Messages)
	if t.Header.Model != "" {
		fmt.Fprintf(&b, ", model %s", t.Header.Model)
	}
	b.WriteString("). Translated by PiCode; prior tool results are stale — re-check anything that matters before relying on it.\n\n")

	section := func(title, body string) {
		b.WriteString("## " + title + "\n")
		if strings.TrimSpace(body) == "" {
			b.WriteString("none\n\n")
			return
		}
		b.WriteString(strings.TrimSpace(body) + "\n\n")
	}
	if s := strings.TrimSpace(summary); s != "" {
		section("Earlier part of the conversation, as summarized by the previous agent", truncate(s, 6000))
	}
	section("Last request", truncate(t.LastUserText(), 1500))
	section("Where it stopped", truncate(t.LastAssistantText(), 1500))

	var turns []string
	for i := len(t.Events) - 1; i >= 0 && len(turns) < 6; i-- {
		e := t.Events[i]
		if e.Kind != KindMessage || strings.TrimSpace(e.Text) == "" {
			continue
		}
		turns = append(turns, "- **"+e.Role+":** "+truncate(oneLine(e.Text), 400))
	}
	for i, j := 0, len(turns)-1; i < j; i, j = i+1, j-1 {
		turns[i], turns[j] = turns[j], turns[i]
	}
	section("Recent turns", strings.Join(turns, "\n"))

	files, commands := touched(t)
	section("Files touched", bullets(files))
	section("Commands run", bullets(commands))

	out := b.String()
	if len(out) > BriefMaxBytes {
		// Cut the middle of the recent-turns section first; the head and
		// the tail (files, commands) are what the next agent acts on.
		out = out[:BriefMaxBytes-len(briefCutMark)] + briefCutMark
	}
	return out
}

const briefCutMark = "\n\n[brief cut at 16 KB]\n"

var (
	writeToolName = regexp.MustCompile(`(?i)write|edit|create|patch|apply|replace|multi_?edit`)
	pathKeys      = []string{"file_path", "path", "target_file", "filePath", "file", "notebook_path"}
	commandKeys   = []string{"command", "cmd", "script"}
)

// touched extracts the files a writing tool named and the shell commands
// the agent ran, in first-seen order and de-duplicated.
func touched(t Timeline) (files, commands []string) {
	seenF := map[string]bool{}
	seenC := map[string]bool{}
	for _, e := range t.Events {
		if e.Kind != KindToolCall || e.Call == nil {
			continue
		}
		in := e.Call.InputObject()
		if in == nil {
			continue
		}
		for _, k := range commandKeys {
			if v, ok := in[k].(string); ok && strings.TrimSpace(v) != "" {
				v = truncate(oneLine(v), 200)
				if !seenC[v] {
					seenC[v] = true
					commands = append(commands, v)
				}
				break
			}
		}
		if !writeToolName.MatchString(e.Call.Name) {
			continue
		}
		for _, k := range pathKeys {
			if v, ok := in[k].(string); ok && strings.TrimSpace(v) != "" {
				if !seenF[v] {
					seenF[v] = true
					files = append(files, v)
				}
				break
			}
		}
	}
	return files, commands
}

func bullets(items []string) string {
	if len(items) == 0 {
		return ""
	}
	const max = 40
	if len(items) > max {
		rest := len(items) - max
		items = append(append([]string{}, items[:max]...), fmt.Sprintf("… and %d more", rest))
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, "- "+it)
	}
	return strings.Join(out, "\n")
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	cut := s[:n]
	if i := strings.LastIndexByte(cut, ' '); i > n/2 {
		cut = cut[:i]
	}
	return cut + "…"
}

// SortedKeys is a small helper for deterministic output in writers.
func SortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
