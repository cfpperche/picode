// Package version holds build identity for picode.
package version

import "runtime/debug"

// Version is the semantic version of the running build. It is a var, not a
// const, so a release build can stamp the real tag into it:
//
//	go build -ldflags "-X github.com/cfpperche/picode/internal/version.Version=1.2.3 \
//	                   -X github.com/cfpperche/picode/internal/version.Stamped=release"
//
// Kept in sync with CHANGELOG.md releases; a source build keeps this value.
// Update comparisons (install.Newer) use Version alone — never Build().
var Version = "0.1.0"

// Stamped marks a release build (set by the release workflow alongside
// Version). Empty on source builds, which is what makes Build() append
// the git revision.
var Stamped = ""

// Build is the display identity: the stamped release version as-is, or —
// for a source build — Version plus the VCS revision Go embedded at build
// time ("0.1.0+0550fa2", with a trailing * when the tree was dirty).
// A binary with no VCS info falls back to plain Version.
func Build() string {
	if Stamped != "" {
		return Version
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return Version
	}
	rev, modified := "", false
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	return build(Version, rev, modified)
}

func build(v, rev string, modified bool) string {
	if rev == "" {
		return v
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	s := v + "+" + rev
	if modified {
		s += "*"
	}
	return s
}
