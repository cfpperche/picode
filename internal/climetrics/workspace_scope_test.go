package climetrics

import (
	"testing"
	"time"
)

func TestWorkspaceScopeFiltersCurrentPriorAndSeries(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	req := Request{From: now.Add(-24 * time.Hour), To: now.Add(time.Hour), PriorFrom: now.Add(-48 * time.Hour), Loc: time.UTC,
		KeepCwd: func(cwd string) bool { return cwd == "/wanted" }}
	acc := newCliAcc(req, "codex")
	for _, e := range []cliEntry{
		{at: now, key: "a", cwd: "/wanted", role: "user", cost: 2},
		{at: now, key: "b", cwd: "/other", role: "user", cost: 99},
		{at: now.Add(-36 * time.Hour), key: "c", cwd: "/wanted", role: "user", cost: 3},
		{at: now.Add(-36 * time.Hour), key: "d", cwd: "/other", role: "user", cost: 77},
	} {
		acc.add(e)
	}
	st := acc.result()
	if st.Current.Cost != 2 || st.Current.Messages != 1 || st.Prior == nil || st.Prior.Cost != 3 || len(st.Series) != 1 || st.Series[0].Cost != 2 {
		t.Fatalf("scope leaked: %+v", st)
	}
}
