package store

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ADR-0048: a mutation is a store method that appends its event in the
// same transaction, so the UI can subscribe to the feed instead of
// polling. TestEveryMutationAppendsAnEvent proves that for the cases
// somebody remembered to write down — it is a checklist of 118 hand-made
// rows, and a new mutator that forgets its event simply is not on it.
//
// This is the invariant itself: every exported method that writes to a
// table either announces the change, or is named below with the reason it
// does not. Adding a mutator now means adding its event or adding a line
// here, and the second one is a decision somebody has to type out.

// silentMutators are the exported writes that deliberately announce
// nothing. Each needs a reason, not just a name: an entry without one
// fails, so "I could not think of the event" cannot pass as a decision.
var silentMutators = map[string]string{
	// Bookkeeping on the event log itself.
	"PruneEvents": "prunes the event log; an event about it would refill what it just emptied",

	// Auth housekeeping. Devices are read through /api/auth/sessions on
	// demand, so a feed event here would be one row per authenticated
	// request.
	"LookupSession": "stamps last_seen_at on the authenticating session, at most once a minute",
	"PruneSessions": "drops expired rows on a timer; a row that cannot authenticate is no longer a device",

	// Web Push delivery marks (ADR-0047). No surface watches an endpoint's
	// failure count; the notifier owns the retry.
	"MarkPushOK":      "records one delivery result; the notifier owns the retry and nothing renders it",
	"MarkPushFailure": "records one delivery result; the notifier owns the retry and nothing renders it",

	// Extension actuation batches (ADR-0053/0054). The extension panel
	// drives execution by polling for its own batch, so a feed event would
	// reach every subscriber except the one that acts on it.
	"CreateActBatch":          "the extension panel polls for its batch; the feed has no subscriber for one",
	"ClaimActBatch":           "the extension panel polls for its batch; the feed has no subscriber for one",
	"FinishActBatch":          "the extension panel polls for its batch; the feed has no subscriber for one",
	"ExpirePendingActBatches": "sweeps batches nobody came for; there is nothing on screen to update",

	// Agent session history (ADR-0039). These resolve which JSONL an agent
	// owns, around spawn and listing. What a viewer sees is the agent row,
	// and that one does announce itself.
	"NewPendingAgentSession":   "reserves a session id before the file exists; agent.updated carries what is visible",
	"ResolveAgentSessionID":    "binds a reserved id to the file that appeared; agent.updated carries what is visible",
	"RecordAgentSessionPath":   "historizes a session file the agent now owns; agent.updated carries what is visible",
	"SealPendingAgentSessions": "closes reservations when an agent stops; agent.updated carries what is visible",
}

var (
	sqlWrite  = regexp.MustCompile(`(?i)\b(INSERT\s+INTO|INSERT\s+OR\s+\w+\s+INTO|UPDATE\s+[a-z_]+\s+SET|DELETE\s+FROM|REPLACE\s+INTO)\b`)
	announces = regexp.MustCompile(`AppendEvent`)
)

// storeMethods returns every method on *Store in this package, exported
// or not, mapped to the source text of its body.
func storeMethods(t *testing.T) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read store package: %v", err)
	}
	fset := token.NewFileSet()
	out := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		f, err := parser.ParseFile(fset, name, b, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		base := fset.File(f.Pos()).Base()
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil || len(fn.Recv.List) != 1 {
				continue
			}
			star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			id, ok := star.X.(*ast.Ident)
			if !ok || id.Name != "Store" {
				continue
			}
			out[fn.Name.Name] = string(b[int(fn.Body.Pos())-base : int(fn.Body.End())-base])
		}
	}
	if len(out) == 0 {
		t.Fatal("parsed no *Store methods — the analysis would pass vacuously")
	}
	return out
}

// emits reports whether a method appends an event, directly or through a
// helper on the same receiver.
func emits(bodies map[string]string, name string, seen map[string]bool) bool {
	if seen[name] {
		return false
	}
	seen[name] = true
	body, ok := bodies[name]
	if !ok {
		return false
	}
	if announces.MatchString(body) {
		return true
	}
	for _, m := range regexp.MustCompile(`\bs\.([A-Za-z]\w*)\(`).FindAllStringSubmatch(body, -1) {
		if emits(bodies, m[1], seen) {
			return true
		}
	}
	return false
}

func TestEveryExportedMutationAnnouncesOrIsListed(t *testing.T) {
	bodies := storeMethods(t)

	var silent []string
	for name, body := range bodies {
		if !ast.IsExported(name) || !sqlWrite.MatchString(body) {
			continue
		}
		if emits(bodies, name, map[string]bool{}) {
			if _, listed := silentMutators[name]; listed {
				t.Errorf("%s announces its change but is still listed in silentMutators — remove the line", name)
			}
			continue
		}
		if reason, listed := silentMutators[name]; listed {
			if strings.TrimSpace(reason) == "" {
				t.Errorf("silentMutators[%q] has no reason — say why the feed does not need it", name)
			}
			continue
		}
		silent = append(silent, name)
	}
	sort.Strings(silent)
	for _, name := range silent {
		t.Errorf("Store.%s writes to a table and appends no event (ADR-0048).\n"+
			"    Append the event in the same transaction, or add %q to silentMutators with the reason.", name, name)
	}
}

// A name that is no longer a method, or no longer a mutator, keeps the
// list honest only if something says so.
func TestSilentMutatorListHasNoStaleEntries(t *testing.T) {
	bodies := storeMethods(t)
	for name := range silentMutators {
		body, ok := bodies[name]
		if !ok {
			t.Errorf("silentMutators lists %q, which is not a method on *Store any more", name)
			continue
		}
		if !sqlWrite.MatchString(body) {
			t.Errorf("silentMutators lists %q, which no longer writes to a table — remove the line", name)
		}
	}
}
