// Shared by docs-shots (capture) and docs-check (parity gate): the hash of
// the working-tree files that PRODUCE what the docs screenshots show — UI
// source, server routes, the fixture world, and the pipeline itself.
// Docs-only commits (changelog, handoff) don't change it, so images stay
// valid across them; any UI/server/fixture change does, forcing a re-capture.
// Hashes the working tree, so a capture works before the code is committed.

import { createHash } from "node:crypto";
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

export function uiTreeHash(root) {
  const files = [];
  const collect = (dir, re) => {
    for (const e of readdirSync(join(root, dir), { withFileTypes: true })) {
      const rel = dir + "/" + e.name;
      if (e.isDirectory()) collect(rel, re);
      else if (re.test(e.name)) files.push(rel);
    }
  };
  collect("web/src", /\.(jsx?|css|svg)$/);
  collect("internal/server", /\.go$/);
  files.push("cmd/picode-docs-fixture/main.go", "scripts/docs-shots.mjs");
  files.sort();

  const h = createHash("sha256");
  for (const f of files) {
    h.update(f + "\0");
    h.update(readFileSync(join(root, f)));
  }
  return h.digest("hex");
}
