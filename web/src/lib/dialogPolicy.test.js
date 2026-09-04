import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative, sep } from "node:path";
import { fileURLToPath } from "node:url";

// ADR-0046: every modal goes through components/ResponsiveDialog.jsx, so it
// is a centred dialog on the desktop and a bottom sheet on the phone. A raw
// dialog/drawer import anywhere else is a second, unresponsive modal — the
// exact drift the owner hit (New agent was a sheet, Choose folder was not).
const SRC = fileURLToPath(new URL("../", import.meta.url));
const ALLOW = new Set([
  "components/ResponsiveDialog.jsx", // the primitive itself
  "components/Palette.jsx",          // Ctrl+K command palette — desktop-only, cmdk inside a dialog
  "components/Hotkeys.jsx",          // keyboard shortcut sheet — desktop-only by definition
]);
const RAW = /from\s+["'](@radix-ui\/react-dialog|@radix-ui\/react-alert-dialog|vaul)["']/;

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (/\.(jsx|js)$/.test(name) && !name.endsWith(".test.js")) out.push(p);
  }
  return out;
}

describe("dialog policy", () => {
  it("only ResponsiveDialog (and the two desktop-only surfaces) import a raw dialog or drawer", () => {
    const offenders = walk(SRC)
      .filter((p) => RAW.test(readFileSync(p, "utf8")))
      .map((p) => relative(SRC, p).split(sep).join("/"))
      .filter((rel) => !ALLOW.has(rel));
    assert.deepEqual(offenders, [], "use components/ResponsiveDialog.jsx: " + offenders.join(", "));
  });
  it("the allowlist names files that exist", () => {
    for (const rel of ALLOW) statSync(join(SRC, rel));
  });
});
