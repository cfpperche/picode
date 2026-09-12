package apps

import "context"

// canvasApp is the Canvas (docs/plans/matrix-app.md; persistence ADR-0108,
// surface ADR-0109, renamed by ADR-0118): a named plane of agent and
// terminal panels rendered live by the desktop. The app has no primitives
// body — its data lives under /api/canvases (internal/server/canvas.go) and
// its body is the component the desktop registers as "canvas"
// (web/desktop/src/lib/nativeApps.js). Every other client gets nativeView's
// one honest line. Present in every build; no badge (plan §4.10).
type canvasApp struct{}

func (canvasApp) Manifest() Manifest {
	return Manifest{ID: "canvas", Name: "Canvas", Icon: "canvas", APIVersion: APIVersion, Surface: SurfaceNative}
}

func (canvasApp) Badge(context.Context, Host) (Badge, error) { return Badge{}, nil }

func (a canvasApp) View(_ context.Context, _ Host, _ string) (View, error) {
	return nativeView(a.Manifest().Name), nil
}

func (a canvasApp) Action(_ context.Context, _ Host, req ActionRequest) (ActionResult, error) {
	return nativeAction(a.Manifest().Name, req)
}
