//go:build embedui

package server

import "testing"

// The UI is inside the binary in this build, so there is nothing to arrange.
func withUI(t *testing.T) string {
	t.Helper()
	return ""
}
