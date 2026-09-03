#!/usr/bin/env node
// docs-shots — capture the CURRENT app UI into www/img/.
//
// Parity principle (docs/benchmarks/2026-09-03-docs-harness.md): every image
// on the docs site is generated from the codebase, never hand-placed and
// never reused from docs/screenshots/ (agent work evidence). This script
// drives agent-browser against the fixture daemon
// (cmd/picode-docs-fixture — synthetic, seeded, ungated) and writes framed
// PNGs plus manifest.json (UI-tree hash + asset hashes), which
// `make docs-check` verifies.
//
// Lessons baked in (all the hard way):
//  - one browser session per run: a session's window that is not focused
//    paints blank in headless Chromium;
//  - the shell (?desktop=1 / ?mobile=1) goes in the URL query, which always
//    wins over a stored pref (lib/shell.js); the nonce goes in the query too,
//    never in the fragment (a fragment nonce breaks the mobile router);
//  - the marker check is scoped to the surface's own container and rejects
//    the Reconnecting banner — a cold load can boot empty (the app swallows
//    a failed boot fetch) and the shutter must never fire on that;
//  - a passed content gate can still yield an unpainted frame, so the PNG is
//    size-checked and re-shot up to three times.

import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdirSync, writeFileSync, readFileSync, statSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { uiTreeHash } from "./lib/uitree.mjs";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const outDir = process.argv.includes("--out")
  ? process.argv[process.argv.indexOf("--out") + 1]
  : join(root, "www", "img");
const base = process.argv.includes("--base")
  ? process.argv[process.argv.indexOf("--base") + 1]
  : "http://127.0.0.1:18740";

// waitText must be VIEW-SPECIFIC and is checked inside `scope` (both mobile
// screens render the seeded question's title, so the container decides).
const surfaces = [
  { name: "app-fleet", path: "/?desktop=1", w: 1440, h: 900, settle: 4000, waitText: "Atlas" },
  // app-automations is off the list for now: a cold deep link to
  // #/automations mounts the workspace dashboard (app deep-link bug,
  // handoff 2026-09-03) — the surface returns once that is fixed.
  { name: "app-mobile-inbox", path: "/?mobile=1#/app/inbox", w: 390, h: 844, settle: 4000, waitText: "Bump the Go toolchain", scope: ".m-inbox" },
  { name: "app-mobile", path: "/?mobile=1", w: 390, h: 844, settle: 4000, waitText: "Bump the Go toolchain", scope: ".m-screen" },
];

const ab = (args) =>
  execFileSync("agent-browser", args, { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function gitSha() {
  try {
    return execFileSync("git", ["rev-parse", "HEAD"], { cwd: root, encoding: "utf8" }).trim();
  } catch {
    return "unknown";
  }
}

async function main() {
  mkdirSync(outDir, { recursive: true });
  const manifestPath = join(outDir, "manifest.json");
  const prev = (() => {
    try { return JSON.parse(readFileSync(manifestPath, "utf8")); } catch { return null; }
  })();

  // Stale windows from previous runs occlude the new one (headless paints
  // the focused surface only) — sweep them first.
  try { ab(["close", "--all"]); } catch { /* first run: nothing open */ }

  // ONE browser session for the whole run; surfaces stay sequential and
  // focused. A session's window that is not focused paints blank.
  // ONE session for preflight + every surface: a second tab (e.g. a
  // default-session preflight) holds the window focus, and a background tab
  // throttles fetch/render — the shutter then fires on a hollow app.
  const sess = ["--session", `shot-${Date.now().toString(36)}`];

  // Preflight in the run session: the fixture must be serving the SEEDED
  // world. A stale fixture whose data dir was recreated underneath it
  // answers with an empty store, and every capture would photograph
  // nothing.
  ab([...sess, "set", "viewport", "1440", "900"]);
  ab([...sess, "open", `${base}/?desktop=1`]);
  await sleep(1200);
  const probe = ab([...sess, "eval", "fetch('/api/workspaces').then(r => r.text()).then(t => t.includes('Atlas') ? 'seeded' : 'EMPTY ' + t.slice(0, 60)).catch(e => 'DOWN ' + e)"]);
  if (!probe.includes('seeded')) {
    throw new Error(`fixture on ${base} is not serving the seeded world (${probe.trim().slice(0, 160)}) — (re)start it: make fixture`);
  }

  const captured = {};
  for (const s of surfaces) {
    const file = `${s.name}.png`;
    const out = join(outDir, file);
    // The nonce goes in the SEARCH part; a fragment nonce would land inside
    // the hash and break the mobile router.
    const [pathNoHash, hash] = s.path.split("#");
    const glue = pathNoHash.includes("?") ? "&" : "?";
    const url = `${base}${pathNoHash}${glue}_r=${Date.now()}${hash ? "#" + hash : ""}`;

    let has = false;
    for (let round = 0; round < 6 && !has; round++) {
      try {
        ab([...sess, "set", "viewport", String(s.w), String(s.h)]);
        ab([...sess, "open", "about:blank"]);
        ab([...sess, "open", url]);
        // Dark via the app's own theme pref (picode-theme), not media
        // emulation — the emulation raced the compositor and produced blank
        // frames intermittently. The pref survives the reload below.
        ab([...sess, "eval", "localStorage.setItem('picode-theme','dark'); location.reload();"]);
        await sleep(s.settle);
        // Content gate, scoped to the surface's own container: the seeded
        // marker present AND no Reconnecting banner. A cold boot can bounce
        // the deep link back to #/ once the fleet loads — re-set the hash
        // (what clicking the app tab does) and give the router a beat.
        if (s.rehash && !ab([...sess, "eval", `location.hash === ${JSON.stringify(s.rehash)}`]).includes("true")) {
          ab([...sess, "eval", `location.hash = ${JSON.stringify(s.rehash)}`]);
          await sleep(1500);
        }
        const scopeSel = s.scope
          ? `(document.querySelector('${s.scope}') || document.body).innerText`
          : "document.body.innerText";
        // The eval answers in plain words — the CLI JSON-escapes its output,
        // so quoted/long substrings from the page can never be matched
        // reliably in the wrapper's stdout.
        const text = ab([...sess, "eval", `(${scopeSel}).includes(${JSON.stringify(s.waitText)}) ? (document.body.innerText.includes('Reconnecting') ? 'MARKER_RECON' : 'MARKER_OK') : 'MARKER_NO'`]);
        console.log(`    [${s.name} r${round}] ${text.trim().slice(0, 120)}`);
        has = text.includes("MARKER_OK");
      } catch {
        has = false;
        await sleep(1500);
      }
    }
    if (!has) {
      const dbg = `/tmp/docs-shots-debug-${s.name}.png`;
      try { ab([...sess, "screenshot", dbg]); } catch { /* keep the original error */ }
      throw new Error(`${s.name}: waitText ${JSON.stringify(s.waitText)} never appeared — debug shot: ${dbg}`);
    }
    // A passed content gate can still yield an unpainted frame: the DOM has
    // the content but the PNG is a near-solid color, which compresses tiny.
    // A real PiCode surface at these viewports is 30KB+, so a tiny PNG means
    // blank — re-shoot a few times before giving up.
    let shot = 0;
    for (;;) {
      ab([...sess, "screenshot", out]);
      shot += 1;
      if (statSync(out).size > 20000 || shot >= 3) break;
      await sleep(1500);
    }
    if (shot > 1) console.log(`    [${s.name}] blank frame — took ${shot} attempts`);

    captured[s.name] = {
      file,
      url: s.path,
      viewport: `${s.w}x${s.h}`,
      sha256: createHash("sha256").update(readFileSync(out)).digest("hex"),
    };
    console.log(`  ${s.name} -> ${file}`);
  }

  try { ab([...sess, "close"]); } catch { /* already gone */ }

  const manifest = {
    capturedAt: new Date().toISOString(),
    gitSha: gitSha(),
    uiTreeHash: uiTreeHash(root),
    base,
    tool: "agent-browser",
    note: "Generated by scripts/docs-shots.mjs — do not edit images by hand.",
    surfaces: captured,
  };
  writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + "\n");

  if (prev) {
    const changed = Object.keys(captured).filter(
      (k) => prev.surfaces?.[k]?.sha256 && prev.surfaces[k].sha256 !== captured[k].sha256,
    );
    if (changed.length) console.log(`changed since last run: ${changed.join(", ")}`);
  }
  console.log(`docs-shots: ${surfaces.length} surfaces -> ${outDir}`);
}

main().catch((e) => {
  console.error("docs-shots failed:", e.message);
  process.exit(1);
});
