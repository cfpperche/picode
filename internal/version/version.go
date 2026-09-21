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
var Version = "0.4.0"

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
	if rev := Revision(); rev != "" {
		return build(Version, rev)
	}
	return Version
}

// Revision is the full VCS revision embedded at build time (40 or 64 hex), or
// "" when the binary carries no VCS stamp. D2's publication observation maps a
// running artifact to Git by this exact value; Build() shortens it for display
// and must never be compared against a Git object.
func Revision() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" && isRevision(s.Value) {
			return s.Value
		}
	}
	return ""
}

func isRevision(v string) bool {
	if len(v) != 40 && len(v) != 64 {
		return false
	}
	for _, c := range v {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
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
