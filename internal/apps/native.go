package apps

import (
	"context"
	"fmt"
)

// Native surfaces (ADR-0109). An app whose Manifest.Surface is
// SurfaceNative has no primitives body: its body is a component compiled
// into a shell that registers the app id (the desktop today). What every
// such app answers on the primitives routes lives here, so a client that
// cannot host the surface — the phone, an older bundle, curl — still gets
// one honest line instead of a 500.

// nativeView is the one screen a native app has on the primitives route.
func nativeView(name string) View {
	return View{
		APIVersion: APIVersion,
		Title:      name,
		Blocks:     []Block{{Type: "detail", Markdown: name + " opens on the desktop."}},
	}
}

// nativeAction refuses: nothing on the primitives route acts for a
// native app. The handler maps the error to 400.
func nativeAction(name string, req ActionRequest) (ActionResult, error) {
	return ActionResult{}, fmt.Errorf("%s: no action %q here — this app opens on the desktop", name, req.Action)
}

// nativeDemoApp is the hidden QA app for the native surface
// (PICODE_DEMO_APP=1, beside demoApp). It gives the surface kind a real
// consumer before the Canvas (docs/plans/matrix-app.md) lands: the
// desktop registers "demo-native" and renders a terminal through it, the
// phone lists it as Desktop only. Never in BuiltIns(false).
type nativeDemoApp struct{}

func (nativeDemoApp) Manifest() Manifest {
	return Manifest{ID: "demo-native", Name: "Native demo", Icon: "grid", APIVersion: APIVersion, Surface: SurfaceNative}
}

func (nativeDemoApp) Badge(context.Context, Host) (Badge, error) { return Badge{}, nil }

func (a nativeDemoApp) View(_ context.Context, _ Host, _ string) (View, error) {
	return nativeView(a.Manifest().Name), nil
}

func (a nativeDemoApp) Action(_ context.Context, _ Host, req ActionRequest) (ActionResult, error) {
	return nativeAction(a.Manifest().Name, req)
}
