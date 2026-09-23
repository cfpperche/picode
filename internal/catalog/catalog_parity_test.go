package catalog

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/climodels"
)

// The table parser moved to climodels (ADR-0009 amendment, 2026-09-23). This
// is the parser as it stood before the move, kept to prove the catalog's rows
// did not change: same providers, models, labels and flags, row for row.
func legacyParseListModels(text string) []parsedRow {
	var rows []parsedRow
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		if fields[0] == "provider" {
			continue
		}
		// provider | model… | context | max-out | thinking | images
		images := fields[len(fields)-1]
		thinking := fields[len(fields)-2]
		// The CLI also prints prose when no models are available. A table
		// row always ends in two yes/no capability columns; prose is not
		// a provider or a model, even when it contains six or more words.
		if (images != "yes" && images != "no") || (thinking != "yes" && thinking != "no") {
			continue
		}
		maxOut := fields[len(fields)-3]
		context := fields[len(fields)-4]
		model := strings.Join(fields[1:len(fields)-4], " ")
		rows = append(rows, parsedRow{
			provider: fields[0],
			model:    model,
			context:  context,
			maxOut:   maxOut,
			thinking: thinking == "yes",
			images:   images == "yes",
		})
	}
	return rows
}

// Every size label measured on 2026-09-23 (pi 0.87.1), a model id with
// spaces, both capability columns in all four combinations, and prose.
const parityTable = `provider          model                                               context  max-out  thinking  images
anthropic         claude-fable-5                                      1M       128K     yes       yes
anthropic         claude-haiku-4-5                                    200K     64K      yes       no
openrouter        qwen/qwen3 coder free                               1.0M     32.8K    no        yes
openrouter        meta/llama 3.1 8b                                   131.1K   4.1K     no        no
xai               grok-4.6                                            2M       1.1M     yes       yes
zai               glm-5                                               202.8K   3.7K     yes       no
No models are available for this provider right now, sign in first
`

func TestParserParityWithTheCatalogsOwn(t *testing.T) {
	for _, table := range []string{parityTable, sample, "", "not a table\n\n"} {
		got, want := ParseListModels(table), legacyParseListModels(table)
		if len(got) == 0 && len(want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("rows differ for table %q:\n got %+v\nwant %+v", strings.SplitN(table, "\n", 2)[0], got, want)
		}
	}
}

// Filling Pi's thinking levels copies: the rows handed in may be a cached
// report other reads share.
func TestFillPiThinkingLeavesItsInputAlone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	in := climodels.ParsePiTable(parityTable)
	out := FillPiThinking(in)
	if len(out) != len(in) || len(out[0].Thinking) < 2 {
		t.Fatalf("out = %+v", out[0])
	}
	for _, m := range in {
		if m.Thinking != nil {
			t.Fatalf("the input was changed: %+v", m)
		}
	}
}
