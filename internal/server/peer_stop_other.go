//go:build !linux

package server

import (
	"context"
	"errors"
)

func stopPeerPane(ctx context.Context, deps Deps, name, id string, pane int, rt TermRuntime) error {
	return errors.New("Automatic resume is not supported on this platform. Reopen the conversation.")
}
