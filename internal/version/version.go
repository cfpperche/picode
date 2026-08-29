// Package version holds build identity for picode.
package version

// Version is the semantic version of the running build. It is a var, not a
// const, so a release build can stamp the real tag into it:
//
//	go build -ldflags "-X github.com/cfpperche/picode/internal/version.Version=1.2.3"
//
// Kept in sync with CHANGELOG.md releases; a source build keeps this value.
var Version = "0.1.0"
