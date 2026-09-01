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
// time ("0.1.0+0550fa2"). A binary with no VCS info falls back to plain
// Version.
//
// vcs.modified is deliberately ignored: Go computes it against the
// repository's primary checkout, not the (linked) worktree being built,
// and this repo's primary checkout is routinely dirty with other agents'
// work — the flag would be noise, while the revision is exact.
func Build() string {
	if Stamped != "" {
		return Version
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return Version
	}
	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" {
			return build(Version, s.Value)
		}
	}
	return Version
}

func build(v, rev string) string {
	if rev == "" {
		return v
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	return v + "+" + rev
}
