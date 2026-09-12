import { supportedApp, SUPPORTED_API } from "@picode/shared/contracts/appPrimitives.js";

// Native app surfaces (ADR-0109) — the desktop's registry: manifest id →
// the component compiled into this shell. Explicit assembly, no
// self-registration: App.jsx builds the one instance with the components
// it imports (this module holds no JSX, so `node --test` can load it).
// `ids` is the Set the contract's gate (supportedApp) reads; the grid and
// the tab mount both go through it, so a tile is never enabled for an app
// this shell could not open.
export function nativeApps(entries) {
  const map = new Map(Object.entries(entries || {}));
  return Object.freeze({
    get: (id) => map.get(id) || null,
    has: (id) => map.has(id),
    ids: new Set(map.keys()),
  });
}

// appTile is how the grid draws one manifest — ADR-0109's decision table:
// enabled, or unsupported with the reason the tile's title says.
export function appTile(manifest, registry) {
  const ids = registry ? registry.ids : new Set();
  const name = manifest.name;
  if (supportedApp(manifest, ids)) return { ok: true, title: name };
  if (manifest.apiVersion !== SUPPORTED_API) return { ok: false, title: name + " needs a newer PiCode (app speaks v" + manifest.apiVersion + ")" };
  if (manifest.surface === "native") return { ok: false, title: name + " needs a newer PiCode (this build has no " + name + " surface)" };
  return { ok: false, title: name + " needs a newer PiCode (unknown surface \u201c" + manifest.surface + "\u201d)" };
}

// nativeSurfaceFor is the tab mount's question: the component that renders
// this manifest, or null — a primitives app, or a native one this shell
// did not compile in (AppSurface then shows the honest line).
export function nativeSurfaceFor(manifest, registry) {
  if (!manifest || manifest.surface !== "native" || !registry) return null;
  return registry.get(manifest.id);
}
