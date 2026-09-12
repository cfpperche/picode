import { it } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, relative, resolve } from "node:path";
import { importSpecifiers, sourceFiles } from "./boundaries.mjs";

// ADR-0109, amendment 2026-09-11: an app does not leak into PiCode's own
// interface. The ways an app reaches the host are a closed list the host
// declares — the tile and its badge, the main tab, the app's own body, the
// manifest icon key the host's icon map draws, the `host` object, and the
// hash route the host owns. None of those is an import: the host names an
// app by *id*, never by reaching into its modules.
//
// So the mechanical half of the rule is this direction check. The Canvas is
// the first native app and the case the amendment was written for, so its
// module tree is what this asserts; a second native app adds a row.
const ROOT = fileURLToPath(new URL("../browser/src/", import.meta.url));

// What belongs to the app: its component tree, the per-viewer preference it
// owns (the plane's ground — Preferences carries nothing about a canvas), and
// the plane's own geometry (where a link touches a panel), which lives in
// shared/domain because it is pure and testable there and nowhere else.
const APP_TREE = "components/canvas/";
const APP_OWNED = new Set(["@picode/shared/domain/canvasPattern.js", "@picode/shared/domain/canvasAnchors.js"]);

// The one door that is an import: the host's native-surface mount, which
// lazy-imports the registered component (ADR-0109). Everything else in the
// shell reaches the Canvas through a route or an app id, or not at all.
const MOUNT = "App.jsx";

it("nothing outside the Canvas app imports the Canvas app", async () => {
  const offenders = [];
  for (const path of sourceFiles(ROOT)) {
    const name = relative(ROOT, path).replaceAll("\\", "/");
    if (name.startsWith(APP_TREE) || name === MOUNT) continue;
    for (const spec of await importSpecifiers(readFileSync(path, "utf8"), path)) {
      const reached = spec.startsWith(".")
        ? relative(ROOT, resolve(dirname(path), spec)).replaceAll("\\", "/").startsWith(APP_TREE)
        : APP_OWNED.has(spec);
      if (reached) offenders.push(`${name}: ${spec}`);
    }
  }
  assert.deepEqual(offenders, []);
});

it("Preferences offers nothing that belongs to an app", () => {
  const settings = readFileSync(resolve(ROOT, "components/Settings.jsx"), "utf8");
  assert.doesNotMatch(settings, /canvas/i, "a settings group is not one of the host's doors (ADR-0109)");
});
