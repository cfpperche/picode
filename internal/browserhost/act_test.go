package browserhost

import (
	"strings"
	"testing"
)

func TestParseActBlock(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantOK  bool
		wantErr string
		acts    int
	}{
		{
			name:   "no block",
			text:   "Here is the answer.",
			wantOK: false,
		},
		{
			name:   "valid",
			text:   "Working on it.\n```picode-act\n{\"actions\":[{\"act\":\"click\",\"selector\":\"#go\"},{\"act\":\"fill\",\"selector\":\".q\",\"value\":\"hi\"}]}\n```\ndone",
			wantOK: true, acts: 2,
		},
		{
			name:    "unclosed",
			text:    "```picode-act\n{\"actions\":[]}",
			wantErr: "not closed",
		},
		{
			name:    "bad json",
			text:    "```picode-act\nnope\n```",
			wantErr: "not valid JSON",
		},
		{
			name:    "empty actions",
			text:    "```picode-act\n{\"actions\":[]}\n```",
			wantErr: "no actions",
		},
		{
			name:    "unknown act",
			text:    "```picode-act\n{\"actions\":[{\"act\":\"reload\"}]}\n```",
			wantErr: "unknown act",
		},
		{
			name:    "click without selector",
			text:    "```picode-act\n{\"actions\":[{\"act\":\"click\"}]}\n```",
			wantErr: "needs a selector",
		},
		{
			name:    "scroll without to",
			text:    "```picode-act\n{\"actions\":[{\"act\":\"scroll\"}]}\n```",
			wantErr: "to=top|bottom",
		},
		{
			name:   "last block wins",
			text:   "```picode-act\n{\"actions\":[{\"act\":\"read\",\"selector\":\"a\"}]}\n```\ntext\n```picode-act\n{\"actions\":[{\"act\":\"read\",\"selector\":\"b\"}]}\n```",
			wantOK: true, acts: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := ParseActBlock(tt.text)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if ok != tt.wantOK {
				t.Fatalf("ok %v", ok)
			}
			if ok && len(got.Actions) != tt.acts {
				t.Fatalf("acts %d want %d", len(got.Actions), tt.acts)
			}
		})
	}
}

func TestParseActBlockTooMany(t *testing.T) {
	var b strings.Builder
	b.WriteString("```picode-act\n{\"actions\":[")
	for i := 0; i < ActMaxActions+1; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"act":"read","selector":"x"}`)
	}
	b.WriteString("]}\n```")
	_, _, err := ParseActBlock(b.String())
	if err == nil || !strings.Contains(err.Error(), "max 12") {
		t.Fatalf("err %v", err)
	}
}

func TestComposeActResult(t *testing.T) {
	got := ComposeActResult(2, []ActOutcome{
		{Act: "click", Selector: "#go", OK: true},
		{Act: "fill", Selector: ".q", Error: "no match"},
		{Act: "read", Selector: "main", OK: true, Text: strings.Repeat("x", 1600)},
	}, false)
	for _, want := range []string{
		"round 2 of 3", "click #go: ok", "fill .q: error: no match", "text:\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	if !strings.Contains(got, "…") {
		t.Error("read text was not clipped")
	}
	if !strings.Contains(ComposeActResult(1, nil, true), "stopped the loop") {
		t.Error("stopped line missing")
	}
}
