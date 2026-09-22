package climetrics

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/session"
)

// ClaudeCodeMeter reports Claude Code sessions from the transcripts under
// ~/.claude/projects, the same tree clisession.ClaudeCodeSource lists.
//
// Cost needs care. Claude Code writes no price on a message: the only money
// on disk is the "cost-state" record, which is a *cumulative* snapshot of
// the whole session (totalCostUSD plus per-model costUSD and token counts).
// Two facts follow, both measured on this machine on 2026-09-07:
//
//   - Taking the last snapshot as the window's cost would charge a
//     month-old session's whole bill to the day it was last touched — the
//     fabricated-spike failure ADR-0041 refused for exactly this reason.
//   - Only 46 of 340 transcripts carry a cost-state at all, and none of the
//     six most recently written ones did. A live session has no snapshot.
//
// So this adapter takes the per-model cost from the session's own final
// snapshot — the vendor's own arithmetic, never a price list PiCode would
// have to maintain — and spreads it across that model's messages in
// proportion to their tokens. A window covering the whole session reports
// exactly what Claude Code said it cost; a shorter window reports its share.
// Sessions with no snapshot contribute tokens and activity but no cost, and
// say so: coverage reports cost as partial with both counts, never as a zero
// that would read as "Claude Code was free".
type ClaudeCodeMeter struct{}

func (ClaudeCodeMeter) CLI() string   { return "claude-code" }
func (ClaudeCodeMeter) Label() string { return "Claude Code" }

// Fingerprint sweeps the same files Meter reads — subagent transcripts
// included, since a running subagent appends to its own file while the
// parent's stays still, and a sweep one level deep would serve the stale
// window from cache until the parent next wrote.
func (ClaudeCodeMeter) Fingerprint() string {
	root := clisession.ClaudeProjectsRoot()
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return "0:0:0"
	} else if err != nil {
		return ""
	}
	var n int
	var size, newest int64
	for _, f := range ccTranscripts(root) {
		n++
		size += f.info.Size()
		if m := f.info.ModTime().UnixNano(); m > newest {
			newest = m
		}
	}
	return itoa(n) + ":" + strconv.FormatInt(size, 10) + ":" + strconv.FormatInt(newest, 10)
}

func (m ClaudeCodeMeter) Meter(req Request) (Window, error) {
	root := clisession.ClaudeProjectsRoot()
	acc := newCliAcc(req, m.CLI())
	// A session is priced if any of its files carried a snapshot. Keyed by
	// session rather than file because a subagent transcript (agent-*.jsonl)
	// is a second file of the same session and never has one of its own —
	// counted per file, 75 of 120 files in one week were unpriced "sessions"
	// that were really this machine's Explore agents.
	pricedBy := map[string]bool{}
	underBy := map[string]bool{} // the snapshot itself admits a model it could not price

	// Parse first, replay second: whether a session was priced is known only
	// once all of its files are read (the snapshot lives in the parent, never
	// in a subagent's), and only an unpriced session's turns are estimated —
	// estimating a subagent whose parent's snapshot already covers it would
	// charge it twice.
	var parses []*parsed
	snapshotBy := map[string]bool{}
	for _, f := range ccTranscripts(root) {
		// A file untouched since before the widened window cannot hold
		// an in-window message; the same cheap pre-filter pi's scan uses.
		if !req.PriorFrom.IsZero() && f.info.ModTime().Before(req.PriorFrom) {
			continue
		}
		p := cachedParse(f.path, ccParse)
		parses = append(parses, p)
		snapshotBy[p.key] = snapshotBy[p.key] || p.priced
	}
	for _, p := range parses {
		acc.estimate = !snapshotBy[p.key]
		if !replay(p, acc, req) {
			continue
		}
		pricedBy[p.key] = pricedBy[p.key] || p.priced
		underBy[p.key] = underBy[p.key] || p.underpriced
	}
	acc.estimate = false
	priced, unpriced, under := 0, 0, 0
	for key := range acc.files {
		switch {
		case !pricedBy[key]:
			unpriced++
		case underBy[key]:
			under++
			priced++
		default:
			priced++
		}
	}

	billing := req.BillingFor(m.CLI())
	w := Window{
		CLI:      m.CLI(),
		Stats:    acc.result(),
		Coverage: ccCoverage(m, billing, priced, unpriced, under, acc),
	}
	if acc.impact != (Impact{}) {
		i := acc.impact
		w.Impact = &i
	}
	if acc.timing != (Timing{}) {
		t := acc.timing
		w.Timing = &t
	}
	return w, nil
}

// ccCan is what Claude Code is capable of recording. Whether it *did* in a
// given window is the accumulator's evidence, not this table's claim.
var ccCan = map[Signal]bool{
	SigCost: true, SigTokens: true, SigModel: true, SigMessages: true,
	SigTurns: true, SigTools: true, SigErrors: true,
	SigImpact: true, SigTiming: true,
}

func ccCoverage(m ClaudeCodeMeter, b Billing, priced, unpriced, under int, acc *cliAcc) CoverageRow {
	sig := acc.evidence(ccCan, map[Signal]bool{
		SigImpact: acc.impact != (Impact{}),
		SigTiming: acc.timing != (Timing{}),
	})
	note := ""
	switch {
	case priced == 0 && unpriced == 0:
		// nothing in window; nothing to qualify
	case priced == 0 && acc.estimated > 0:
		// Nothing Claude Code priced itself; everything shown is PiCode's
		// list-price estimate (ADR-0185), whole or partial.
		sig[SigCost] = StateEstimated
		if acc.unpriced > 0 {
			sig[SigCost] = StatePartial
		}
		note = "No session in this window recorded a cost snapshot. " + acc.estimateNote()
	case priced == 0:
		sig[SigCost] = StateNotReported
		note = "No session in this window recorded a cost snapshot; tokens and activity are complete."
	case unpriced > 0 && acc.estimated > 0:
		// Snapshot sessions carry Claude Code's own figure; the rest are
		// estimated, and the note names how much of the total that is.
		if acc.unpriced > 0 {
			sig[SigCost] = StatePartial
		}
		note = "Priced from " + itoa(priced) + " of " + itoa(priced+unpriced) +
			" sessions' own snapshots; a live session has none, and the rest are estimated. " + acc.estimateNote()
	case unpriced > 0:
		sig[SigCost] = StatePartial
		note = "Priced from " + itoa(priced) + " of " + itoa(priced+unpriced) +
			" sessions — cost is written only on a session snapshot, and a live session has none."
	case under > 0:
		// Every session has a snapshot, but Claude Code itself flagged one
		// (hasUnknownModelCost) as carrying a model it could not price.
		// The total is a floor by the vendor's own admission.
		sig[SigCost] = StatePartial
		note = itoa(under) + " of " + itoa(priced) + " sessions carry a model Claude Code could not price, so the total is a floor."
	}
	return CoverageRow{CLI: m.CLI(), Label: m.Label(), Billing: b, Signals: sig, Note: note}
}

// ccFile is one transcript Meter reads.
type ccFile struct {
	path string
	info os.FileInfo
}

// ccTranscripts lists every transcript under root: each project's
// <session>.jsonl, and the subagent transcripts Claude Code files beside
// them under <session>/subagents/. Reading only the top level
// dropped 254 subagent files — 1.48B tokens across 34 sessions — from a
// 30-day window on this machine (2026-09-22). A subagent names its
// parent's sessionId, so it folds into that session rather than counting
// as one of its own.
func ccTranscripts(root string) []ccFile {
	projects, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []ccFile
	add := func(dir string, descend bool) {
		ents, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range ents {
			if e.IsDir() {
				if descend {
					// Workflow agents nest one step further, under
					// subagents/workflows/wf_*/ (31 files, 9M tokens in
					// the same window), so the subagents tree is walked.
					_ = filepath.WalkDir(filepath.Join(dir, e.Name(), "subagents"), func(p string, d os.DirEntry, err error) error {
						if err != nil || d.IsDir() || filepath.Ext(p) != ".jsonl" {
							return nil
						}
						if info, err := d.Info(); err == nil {
							out = append(out, ccFile{p, info})
						}
						return nil
					})
				}
				continue
			}
			if filepath.Ext(e.Name()) != ".jsonl" {
				continue
			}
			if info, err := e.Info(); err == nil {
				out = append(out, ccFile{filepath.Join(dir, e.Name()), info})
			}
		}
	}
	for _, p := range projects {
		if p.IsDir() {
			add(filepath.Join(root, p.Name()), true)
		}
	}
	return out
}

// ccMsg is one transcript entry before pricing.
type ccMsg struct {
	at      time.Time
	role    string
	model   string
	units   int64
	tokens  session.TokenTotals
	stop    string
	results int // tool_result blocks inspected — they arrive on the user turn
	errs    int
	tools   []string
}

type ccState struct {
	modelCost map[string]float64 // model -> USD for the whole session
	unknownPx bool               // Claude Code flagged a model it could not price
	linesAdd  int64
	linesDel  int64
	apiMs     int64
	toolMs    int64
	wallMs    int64
	hasCost   bool
}

// ccParse reads one transcript into a window-independent parse: every
// message priced at its session's own rate, plus the lifetime figures that
// replay will prorate. Nothing here depends on the range, which is what
// lets one parse answer every window and survive in the cache.
func ccParse(path string) *parsed {
	f, err := os.Open(path)
	if err != nil {
		return &parsed{}
	}
	defer f.Close()

	var msgs []ccMsg
	var st ccState
	seen := map[string]int{} // dedupe key -> index in msgs
	cwd, name, sid := "", "", ""

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw map[string]any
		if json.Unmarshal(line, &raw) != nil {
			continue
		}
		if c, _ := raw["cwd"].(string); c != "" && cwd == "" {
			cwd = c
		}
		if id, _ := raw["sessionId"].(string); id != "" && sid == "" {
			sid = id
		}
		switch raw["type"] {
		case "summary":
			if s, _ := raw["summary"].(string); s != "" && name == "" {
				name = s
			}
		case "cost-state":
			readCostState(raw, &st)
		case "user", "assistant":
			m, ok := ccMessage(raw)
			if !ok {
				continue
			}
			// Claude Code writes one record per content block of an API
			// response, and every one of them repeats the response's full
			// usage. Summed, they counted 12.64B tokens where 6.53B were
			// billed (30 days, 2026-09-22). One record per response keeps
			// the usage; the others add only their blocks' tools and results.
			// The key is ccusage's: message id plus request id.
			if k := ccDedupeKey(raw); k != "" {
				if i, dup := seen[k]; dup {
					msgs[i].absorb(m)
					continue
				}
				seen[k] = len(msgs)
			}
			msgs = append(msgs, m)
		}
	}
	if cwd == "" {
		// No entry named a folder: fall back to Claude Code's encoded
		// directory name, which is not a real path and so matches no
		// claimed workspace — the right verdict for a folder we cannot place.
		cwd = filepath.Base(filepath.Dir(path))
	}
	// The session id is the identity, not the file: a subagent's transcript
	// carries its parent's sessionId and folds into that session. Only a
	// file that never names one falls back to its own path.
	key := sid
	if key == "" {
		key = path
	}
	return ccPrice(key, cwd, name, msgs, st)
}

// ccPrice turns the raw messages into priced entries.
func ccPrice(key, cwd, name string, msgs []ccMsg, st ccState) *parsed {
	out := &parsed{key: key, priced: st.hasCost, underpriced: st.unknownPx}
	observed := map[string]int64{}
	for _, m := range msgs {
		out.units += m.units
		observed[ccModelKey(m.model)] += m.units
	}
	rates, flat := ccRates(st.modelCost, observed, out.units)

	out.ents = make([]cliEntry, 0, len(msgs))
	for _, m := range msgs {
		cost := flat * float64(m.units)
		if r, ok := rates[ccModelKey(m.model)]; ok {
			cost += r * float64(m.units)
		}
		out.ents = append(out.ents, cliEntry{
			at: m.at, key: key, cwd: cwd, name: name,
			role: m.role, model: m.model, prov: ccProvider(m.model),
			cost: cost, toks: m.tokens, tools: m.tools,
			results: m.results, errs: m.errs,
			// Claude Code writes no "aborted" stop; an interrupt is not a
			// stop_reason at all. Counting stop_sequence here reported 45
			// aborts against a real zero. A refusal is its own beat.
			refusal: m.stop == "refusal",
		})
	}
	out.impact = Impact{LinesAdded: st.linesAdd, LinesRemoved: st.linesDel}
	out.timing = Timing{APIMs: st.apiMs, ToolMs: st.toolMs, SessionMs: st.wallMs}
	return out
}

// ccRates turns a session's per-model snapshot costs into per-token rates.
//
// Two mismatches between the snapshot and the transcript make the naive
// division wrong, both measured on real files (2026-09-07):
//
//   - The snapshot names models by their billing id ("claude-opus-5[1m]")
//     while messages name them plainly ("claude-opus-5"). Matching on the
//     raw string silently dropped $540 of a $1,001 session — the largest
//     model in it — so the key is normalised on both sides.
//   - A model can be billed with no message of its own left on disk (a
//     subagent's haiku turns live in another transcript). Its cost cannot be
//     placed on any one model, so rather than lose it, it is spread flat
//     across the session's remaining tokens: the model breakdown stays
//     approximate for that slice, the session total stays exact.
//
// Dividing by observed rather than snapshot tokens also absorbs the 1.6x
// gap between them, whatever its cause, because each rate is that model's
// own cost over its own surviving tokens.
func ccRates(modelCost map[string]float64, observed map[string]int64, total int64) (rates map[string]float64, flat float64) {
	rates = make(map[string]float64, len(modelCost))
	var unplaced float64
	for model, cost := range modelCost {
		key := ccModelKey(model)
		if u := observed[key]; u > 0 {
			rates[key] += cost / float64(u)
			continue
		}
		unplaced += cost
	}
	if unplaced > 0 && total > 0 {
		flat = unplaced / float64(total)
	}
	return rates, flat
}

// ccModelKey strips the bracketed billing variant a snapshot appends
// ("claude-opus-5[1m]" -> "claude-opus-5") so both sides of the match speak
// the same name.
func ccModelKey(model string) string {
	if i := strings.IndexByte(model, '['); i > 0 {
		return model[:i]
	}
	return model
}

// ccProvider is the vendor behind a Claude Code model id. Claude Code only
// ever talks to Anthropic, so this is a constant rather than a lookup — but
// it is written as a function so a future Bedrock/Vertex id has one place
// to land.
func ccProvider(model string) string {
	if strings.HasPrefix(model, "<") { // "<synthetic>" — Claude Code's own placeholder
		return "unknown"
	}
	return "anthropic"
}

// readCostState turns one cumulative snapshot into per-model costs.
func readCostState(raw map[string]any, st *ccState) {
	st.hasCost = true
	st.unknownPx, _ = raw["hasUnknownModelCost"].(bool)
	st.linesAdd = int64(num(raw["totalLinesAdded"]))
	st.linesDel = int64(num(raw["totalLinesRemoved"]))
	st.apiMs = int64(num(raw["totalAPIDuration"]))
	st.toolMs = int64(num(raw["totalToolDuration"]))
	st.wallMs = int64(num(raw["totalDuration"]))

	mu, _ := raw["modelUsage"].(map[string]any)
	if mu == nil {
		return
	}
	costs := make(map[string]float64, len(mu))
	for model, v := range mu {
		u, _ := v.(map[string]any)
		if u == nil {
			continue
		}
		if c := num(u["costUSD"]); c > 0 {
			costs[model] = c
		}
	}
	if len(costs) > 0 {
		st.modelCost = costs
	}
}

// ccMessage projects one transcript entry. Content is read for tool names
// and error flags only — never text, which never leaves this package.
func ccMessage(raw map[string]any) (ccMsg, bool) {
	m, _ := raw["message"].(map[string]any)
	if m == nil {
		return ccMsg{}, false
	}
	ts, _ := raw["timestamp"].(string)
	at, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ccMsg{}, false
	}
	out := ccMsg{at: at}
	out.role, _ = m["role"].(string)
	out.model, _ = m["model"].(string)
	out.stop, _ = m["stop_reason"].(string)

	if u, _ := m["usage"].(map[string]any); u != nil {
		out.tokens = session.TokenTotals{
			Input:      int64(num(u["input_tokens"])),
			Output:     int64(num(u["output_tokens"])),
			CacheRead:  int64(num(u["cache_read_input_tokens"])),
			CacheWrite: int64(num(u["cache_creation_input_tokens"])),
		}
		if d, _ := u["output_tokens_details"].(map[string]any); d != nil {
			out.tokens.Reasoning = int64(num(d["thinking_tokens"]))
		}
		out.units = out.tokens.Input + out.tokens.Output + out.tokens.CacheRead + out.tokens.CacheWrite
	}
	if blocks, _ := m["content"].([]any); blocks != nil {
		for _, b := range blocks {
			bm, _ := b.(map[string]any)
			if bm == nil {
				continue
			}
			switch bm["type"] {
			case "tool_use":
				if n, _ := bm["name"].(string); n != "" {
					out.tools = append(out.tools, n)
				}
			case "tool_result":
				out.results++
				if e, _ := bm["is_error"].(bool); e {
					out.errs++
				}
			}
		}
	}
	return out, true
}

// ccDedupeKey identifies the API response an assistant record belongs to,
// or "" when the record names neither half and cannot be matched.
func ccDedupeKey(raw map[string]any) string {
	if raw["type"] != "assistant" {
		return ""
	}
	m, _ := raw["message"].(map[string]any)
	id, _ := m["id"].(string)
	req, _ := raw["requestId"].(string)
	if id == "" && req == "" {
		return ""
	}
	return id + ":" + req
}

// absorb folds a repeated record of the same response into m: its content
// blocks are new, its usage is not.
//
// Usage is not always identical across the records: the output count can
// grow on a later one (2,262 of the groups in 30 days; keeping the first
// lost 12% of output tokens). The largest output is the response's final
// usage, so the record carrying it wins.
func (m *ccMsg) absorb(o ccMsg) {
	if o.tokens.Output > m.tokens.Output {
		m.tokens, m.units = o.tokens, o.units
	}
	m.tools = append(m.tools, o.tools...)
	m.results += o.results
	m.errs += o.errs
	if o.stop != "" {
		m.stop = o.stop
	}
}

func num(v any) float64 {
	f, _ := v.(float64)
	return f
}
