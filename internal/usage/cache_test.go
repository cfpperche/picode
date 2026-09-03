package usage

import (
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
)

func TestSummaryReadsCacheAndNeverGuesses(t *testing.T) {
	t.Cleanup(func() { ForgetProvider("anthropic"); ForgetProvider("openrouter") })
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)

	providers := []catalog.Provider{{
		ID: "anthropic", SignedIn: true, AuthType: "oauth", QuotaKind: "oauth",
		Accounts: []catalog.Account{
			{ID: "a1", Label: "Work", Type: "oauth", Active: true, QuotaKind: "oauth"},
			{ID: "a2", Label: "Personal", Type: "oauth", QuotaKind: "oauth"},
		},
	}, {
		// Signed in, but no plan endpoint: it must not appear at all rather
		// than appear as a zero.
		ID: "google", SignedIn: true, AuthType: "api_key",
		Accounts: []catalog.Account{{ID: "g1", Type: "api_key", Active: true}},
	}}

	pct := 42.0
	Remember("anthropic", "a1", Report{
		Status: StatusOK, Plan: "Max", Windows: []Window{{ID: "5h", Label: "5 hours", UsedPercent: &pct}},
	}, now.Add(-90*time.Second))

	got := Summary(providers, now)
	if len(got) != 2 {
		t.Fatalf("want the two meterable rows, got %d: %+v", len(got), got)
	}
	if got[0].AccountID != "a1" || got[0].Status != StatusOK || got[0].Plan != "Max" {
		t.Fatalf("cached row wrong: %+v", got[0])
	}
	if got[0].AgeSec != 90 {
		t.Fatalf("age %d, want 90 — the roster must say how old the number is", got[0].AgeSec)
	}
	if got[1].AccountID != "a2" || got[1].Status != StatusUnknown {
		t.Fatalf("never-fetched row must be unknown, got %+v", got[1])
	}
	if len(got[1].Windows) != 0 {
		t.Fatalf("an unknown row must carry no windows, got %+v", got[1].Windows)
	}
}

func TestSummaryKeepsFailuresVisible(t *testing.T) {
	t.Cleanup(func() { ForgetProvider("kimi-coding") })
	now := time.Now()
	Remember("kimi-coding", "k1", Report{Status: StatusAuthRequired, Error: "Sign in again."}, now)
	got := Summary([]catalog.Provider{{
		ID: "kimi-coding", SignedIn: true, AuthType: "oauth", QuotaKind: "oauth",
		Accounts: []catalog.Account{{ID: "k1", Type: "oauth", Active: true, QuotaKind: "oauth"}},
	}}, now)
	if len(got) != 1 || got[0].Status != StatusAuthRequired || got[0].Error == "" {
		t.Fatalf("a failed fetch is state to show, got %+v", got)
	}
}

func TestActiveTargetsSkipsPausedAndInactiveRows(t *testing.T) {
	targets := ActiveTargets([]catalog.Provider{{
		ID: "anthropic", SignedIn: true, AuthType: "oauth", QuotaKind: "oauth",
		Accounts: []catalog.Account{
			{ID: "a1", Type: "oauth", Active: true, QuotaKind: "oauth"},
			{ID: "a2", Type: "oauth", QuotaKind: "oauth"},
		},
	}, {
		ID: "xai", SignedIn: true, AuthType: "oauth", QuotaKind: "oauth",
		Accounts: []catalog.Account{{ID: "x1", Type: "oauth", Active: true, Paused: true, QuotaKind: "oauth"}},
	}, {
		ID: "opencode", SignedIn: false,
	}})
	if len(targets) != 1 || targets[0].Provider != "anthropic" || targets[0].AccountID != "a1" {
		t.Fatalf("only the live active slot refreshes on a timer, got %+v", targets)
	}
}

func TestForgetDropsOneRow(t *testing.T) {
	now := time.Now()
	Remember("zai", "z1", Report{Status: StatusOK}, now)
	Remember("zai", "z2", Report{Status: StatusOK}, now)
	Forget("zai", "z1")
	if _, _, ok := Lookup("zai", "z1"); ok {
		t.Fatal("signed-out row still cached")
	}
	if _, _, ok := Lookup("zai", "z2"); !ok {
		t.Fatal("Forget removed the wrong row")
	}
	ForgetProvider("zai")
	if _, _, ok := Lookup("zai", "z2"); ok {
		t.Fatal("ForgetProvider left a row behind")
	}
}
