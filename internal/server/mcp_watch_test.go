package server

import (
	"sort"
	"testing"

	"github.com/cfpperche/picode/internal/mcp"
)

func TestNormalizeLive(t *testing.T) {
	got := normalizeLive(map[string]string{
		"github":  "connected",
		"slack":   "failed",
		"drive":   "needs-auth",
		"context": "connecting", // raw value the report has no badge for
	})
	want := map[string]string{
		"github":  mcp.LiveOn,
		"slack":   mcp.LiveFailed,
		"drive":   mcp.LiveAuth,
		"context": mcp.LiveIdle,
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("%s: got %q want %q", k, got[k], v)
		}
	}
}

func TestDiffLive(t *testing.T) {
	on := map[string]map[string]string{"a": {"github": mcp.LiveOn}}
	failed := map[string]map[string]string{"a": {"github": mcp.LiveFailed}}
	sameOn := map[string]map[string]string{"a": {"github": mcp.LiveOn}} // equal content, new map

	tests := []struct {
		name  string
		prev  map[string]map[string]string
		cur   map[string]map[string]string
		title string
		want  []string
	}{
		{"no change", on, sameOn, "no change must diff to nothing", nil},
		{"status flip", on, failed, "flip must publish once", []string{"a"}},
		{"snapshot appears", nil, on, "new snapshot must publish", []string{"a"}},
		{"snapshot disappears", on, nil, "gone snapshot must publish", []string{"a"}},
		{
			"server added",
			map[string]map[string]string{"a": {"github": mcp.LiveOn}},
			map[string]map[string]string{"a": {"github": mcp.LiveOn, "slack": mcp.LiveAuth}},
			"added server must publish",
			[]string{"a"},
		},
		{
			"other agent only",
			map[string]map[string]string{"a": {"github": mcp.LiveOn}, "b": {"slack": mcp.LiveOn}},
			map[string]map[string]string{"a": {"github": mcp.LiveOn}, "b": {"slack": mcp.LiveFailed}},
			"only the changed agent is listed",
			[]string{"b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffLive(tt.prev, tt.cur)
			sort.Strings(got)
			sort.Strings(tt.want)
			if len(got) != len(tt.want) {
				t.Fatalf("%s: got %v want %v", tt.title, got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("%s: got %v want %v", tt.title, got, tt.want)
				}
			}
		})
	}
}
