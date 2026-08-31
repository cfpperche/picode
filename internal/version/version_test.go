package version

import "testing"

// The display identity: source builds carry the revision (and a dirty
// marker); missing VCS info degrades to the plain version.
func TestBuild(t *testing.T) {
	rows := []struct {
		v, rev   string
		modified bool
		want     string
	}{
		{"0.1.0", "0550fa29aabbcc", false, "0.1.0+0550fa2"},
		{"0.1.0", "0550fa29aabbcc", true, "0.1.0+0550fa2*"},
		{"0.1.0", "", false, "0.1.0"},
		{"2.3.4", "abc", false, "2.3.4+abc"},
	}
	for _, r := range rows {
		if got := build(r.v, r.rev, r.modified); got != r.want {
			t.Errorf("build(%q,%q,%v) = %q, want %q", r.v, r.rev, r.modified, got, r.want)
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
