package apps

import "context"

// matrixApp is the Matrix (docs/plans/matrix-app.md; persistence ADR-0108,
// surface ADR-0109): a named 12-column grid of agent and terminal panels
// rendered live by the desktop. The app has no primitives body — its data
// lives under /api/matrices (internal/server/matrix.go) and its body is the
// component the desktop registers as "matrix" (web/desktop/src/lib/
// nativeApps.js). Every other client gets nativeView's one honest line.
// Present in every build; no badge (plan §4.10).
type matrixApp struct{}

func (matrixApp) Manifest() Manifest {
	return Manifest{ID: "matrix", Name: "Matrix", Icon: "matrix", APIVersion: APIVersion, Surface: SurfaceNative}
}

func (matrixApp) Badge(context.Context, Host) (Badge, error) { return Badge{}, nil }

func (a matrixApp) View(_ context.Context, _ Host, _ string) (View, error) {
	return nativeView(a.Manifest().Name), nil
}

func (a matrixApp) Action(_ context.Context, _ Host, req ActionRequest) (ActionResult, error) {
	return nativeAction(a.Manifest().Name, req)
}
