import { it } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, statSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { relative, resolve } from "node:path";
import { sourceFiles } from "./boundaries.mjs";

// ADR-0072 gives each app one owned primitive. The desktop remains responsive;
// the mobile primitive is always a sheet, including on a wide preview screen.
// The desktop surface's sources live in web/browser (the shell composes
  // them); mobile keeps its own package.
for (const [app, dir] of [["desktop", "browser"], ["mobile", "mobile"]]) it(`${app} uses its own dialog primitive`, () => {
  const root = fileURLToPath(new URL(`../${dir}/src/`, import.meta.url));
  const allowed = app === "desktop" ? ["components/ResponsiveDialog.jsx", "components/Palette.jsx", "components/Hotkeys.jsx"] : ["components/MobileSheet.jsx"];
  const raw = /from\s+["'](@radix-ui\/react-dialog|@radix-ui\/react-alert-dialog|vaul)["']/;
  const offenders = sourceFiles(root).filter(path => raw.test(readFileSync(path, "utf8"))).map(path => relative(root, path).replaceAll("\\", "/")).filter(path => !allowed.includes(path));
  assert.deepEqual(offenders, []);
  for (const path of allowed) statSync(resolve(root, path));
});
