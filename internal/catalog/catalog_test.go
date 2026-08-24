package catalog

import "testing"

const sample = `provider      model                                               context  max-out  thinking  images
anthropic     claude-sonnet-4-5                                   1M       64K      yes       yes
openai-codex  gpt-5.4                                             272K     128K     yes       yes
opencode      gemini-3-flash                                      1.0M     65.5K    yes       yes
`

func TestParseListModels(t *testing.T) {
	rows := ParseListModels(sample)
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	if rows[0].provider != "anthropic" || rows[0].model != "claude-sonnet-4-5" || !rows[0].thinking {
		t.Fatalf("row0 = %+v", rows[0])
	}
	if rows[1].provider != "openai-codex" || rows[1].model != "gpt-5.4" {
		t.Fatalf("row1 = %+v", rows[1])
	}
}

func TestParseListModelsSkipsJunk(t *testing.T) {
	if n := len(ParseListModels("not a table\n\n")); n != 0 {
		t.Fatalf("got %d", n)
	}
}
