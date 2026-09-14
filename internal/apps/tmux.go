package apps

import "context"

// tmuxApp is the tmux app (docs/plans/tmux-app.md): the tmux Server Inspector
// — every session on the tmux server this daemon talks to, PiCode's own
// sessions, the leftovers no store row claims, and the user's own sessions
// beside them. Its body is the component the desktop registers as "tmux"
// (web/browser/src/lib/nativeApps.js); its data lives under /api/tmux
// (internal/server/tmux.go).
//
// No badge, deliberately (the Canvas's reasoning, plan §4.10): a badge would
// have to run a tmux subprocess on every grid render to answer a question
// nobody asked at that moment, and this app's whole job is to be opened when
// something is wrong. Present in every build.
type tmuxApp struct{}

func (tmuxApp) Manifest() Manifest {
	return Manifest{ID: "tmux", Name: "tmux", Icon: "tmux", APIVersion: APIVersion, Surface: SurfaceNative}
}

func (tmuxApp) Badge(context.Context, Host) (Badge, error) { return Badge{}, nil }

func (a tmuxApp) View(_ context.Context, _ Host, _ string) (View, error) {
	return nativeView(a.Manifest().Name), nil
}

func (a tmuxApp) Action(_ context.Context, _ Host, req ActionRequest) (ActionResult, error) {
	return nativeAction(a.Manifest().Name, req)
}
