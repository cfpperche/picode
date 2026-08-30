// Package web supplies the frontend the server serves.
//
// ADR-0023: the built UI is not committed. Two builds exist. The default one
// reads `internal/web/public` from disk, so a fresh clone compiles and tests
// without Node. `-tags embedui` embeds the same directory, and that is what
// ships (ADR-0001: one binary, UI inside).
package web

import "io/fs"

// Built reports whether the UI is actually there. In an embedded build it is,
// by construction. On disk it is only there after `make web`, and saying so is
// better than serving 404s to someone who just cloned the repo.
func Built() bool {
	_, err := fs.Stat(UI(), "index.html")
	return err == nil
}
