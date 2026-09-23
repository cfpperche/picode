//go:build windows

package server

import "time"

// reapPaneGroup is a no-op on Windows: the tmux fixtures skip there.
func reapPaneGroup(pid int, wait time.Duration) {}
