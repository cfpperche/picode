package server

import (
	"sort"
	"testing"
)

func TestDiffWorking(t *testing.T) {
	prev := map[string]bool{"a": false, "b": true, "gone": true}
	cur := map[string]bool{"a": true, "b": true, "new": false}
	got := diffWorking(prev, cur)
	sort.Slice(got, func(i, j int) bool { return got[i].id < got[j].id })
	want := []tuiChange{{"a", true, true}, {"gone", false, false}, {"new", false, true}}
	if len(got) != len(want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %+v want %+v", got, want)
		}
	}
	if len(diffWorking(cur, cur)) != 0 {
		t.Fatal("no change must diff to nothing")
	}
}
