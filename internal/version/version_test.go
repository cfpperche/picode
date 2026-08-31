package version

import "testing"

// The display identity: source builds carry the revision; missing VCS
// info degrades to the plain version. (vcs.modified is ignored — see
// Build's comment.)
func TestBuild(t *testing.T) {
	rows := []struct {
		v, rev string
		want   string
	}{
		{"0.1.0", "0550fa29aabbcc", "0.1.0+0550fa2"},
		{"0.1.0", "", "0.1.0"},
		{"2.3.4", "abc", "2.3.4+abc"},
	}
	for _, r := range rows {
		if got := build(r.v, r.rev); got != r.want {
			t.Errorf("build(%q,%q) = %q, want %q", r.v, r.rev, got, r.want)
		}
	}
}

func TestBuildStampedReleaseIsClean(t *testing.T) {
	old, oldV := Stamped, Version
	t.Cleanup(func() { Stamped, Version = old, oldV })
	Stamped, Version = "release", "1.2.3"
	if got := Build(); got != "1.2.3" {
		t.Errorf("stamped Build() = %q, want 1.2.3", got)
	}
}
