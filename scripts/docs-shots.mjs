#!/usr/bin/env node
// docs-shots — capture the CURRENT app UI into docs-site/img/.
//
// Parity principle (docs/benchmarks/2026-09-03-docs-harness.md): every image
// on the docs site is generated from the codebase, never hand-placed and
// never reused from docs/screenshots/ (agent work evidence). This script
// drives agent-browser against the fixture daemon
// (cmd/picode-docs-fixture — synthetic, seeded, ungated) and writes framed
// PNGs plus manifest.json (per-surface input hashes + asset hashes), which
// `make docs-check` verifies.
//
// Lessons baked in (all the hard way):
//  - one browser session per run: a session's window that is not focused
//    paints blank in headless Chromium;
//  - the application path (/desktop/ or /mobile/) wins over a stored pref;
//    the nonce goes in the query,
//    never in the fragment (a fragment nonce breaks the mobile router);
//  - the marker check is scoped to the surface's own container and rejects
//    the Reconnecting banner — a cold load can boot empty (the app swallows
//    a failed boot fetch) and the shutter must never fire on that;
//  - a passed content gate can still yield an unpainted frame, so the PNG is
//    size-checked and re-shot up to three times.

import { execFileSync, spawn } from "node:child_process";
import net from "node:net";
import { createHash } from "node:crypto";
import { mkdirSync, writeFileSync, readFileSync, statSync, existsSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import {
  DOC_SCREENSHOT_SURFACES,
  FINGERPRINT_VERSION,
  surfaceFingerprint,
} from "./lib/docs-surfaces.mjs";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const outDir = process.argv.includes("--out")
  ? process.argv[process.argv.indexOf("--out") + 1]
  : join(root, "docs-site", "img");
// Without --base the run OWNS its fixture: a fresh daemon on a free port,
// seeded on boot, killed at the end. Sharing one long-lived fixture across
// sessions meant every session's restart killed the others' captures
// (2026-09-12: app-mobile-inbox shot against a daemon that had just died).
async function ownedFixture() {
  const bin = "/tmp/picode-docs-shots-fixture";
  spawnSync("go", ["build", "-o", bin, "./cmd/picode-docs-fixture"], { cwd: root, stdio: "inherit" });
  const port = await new Promise((resolve) => {
    const srv = net.createServer();
    srv.listen(0, "127.0.0.1", () => {
      const p = srv.address().port;
      srv.close(() => resolve(p));
    });
  });
  const child = spawn(bin, ["-addr", `127.0.0.1:${port}`], { cwd: root, stdio: "ignore" });
  return { base: `http://127.0.0.1:${port}`, child };
}

const externalBase = process.argv.includes("--base")
  ? process.argv[process.argv.indexOf("--base") + 1]
  : null;

// waitText must be VIEW-SPECIFIC and is checked inside `scope` (both mobile
// screens render the seeded question's title, so the container decides).
const surfaces = [
  { name: "app-fleet", profile: DOC_SCREENSHOT_SURFACES["app-fleet"], path: "/browser/", w: 1440, h: 900, settle: 4000, waitText: "Atlas" },
  // The Inspector rail beside Atlas's conversation: the fixture seeds a dirty
  // repository under the picode workspace, so Changes lists real counts. The
  // agent id is minted per run, so the hash is set from the fleet after load.
  { name: "app-inspector", profile: DOC_SCREENSHOT_SURFACES["app-inspector"], path: "/browser/", w: 1440, h: 900, settle: 4000, waitText: "Uncommitted", scope: "#inspector",
    hashEval: "fetch('/api/workspaces').then(r => r.json()).then(j => { const a = (Array.isArray(j) ? j : j.workspaces).flatMap(w => w.agents || []).find(x => x.name === 'Atlas'); if (a) location.hash = '#/agent/' + a.id; return a ? 'HASH_OK' : 'HASH_NO'; })" },
  // app-automations is off the list for now: a cold deep link to
  // #/automations mounts the workspace dashboard (app deep-link bug,
  // handoff 2026-09-03) — the surface returns once that is fixed.
  // The canvas id is minted per run, so the hash is set from the list after
  // the app loads — the same trick the inspector uses for its agent.
  { name: "app-canvas", profile: DOC_SCREENSHOT_SURFACES["app-canvas"], path: "/browser/", w: 1440, h: 900, settle: 5000, waitText: "Release day",
    hashEval: "fetch('/api/canvases').then(r => r.json()).then(j => { const c = (j.canvases || [])[0]; if (c) location.hash = '#/app/canvas/' + c.id; return c ? 'HASH_OK' : 'HASH_NO'; })" },
  { name: "app-mobile-inbox", profile: DOC_SCREENSHOT_SURFACES["app-mobile-inbox"], path: "/mobile/#/app/inbox", w: 390, h: 844, settle: 4000, waitText: "Bump the Go toolchain", scope: ".m-inbox" },
  { name: "app-mobile", profile: DOC_SCREENSHOT_SURFACES["app-mobile"], path: "/mobile/", w: 390, h: 844, settle: 4000, waitText: "Bump the Go toolchain", scope: ".m-screen" },
];

const ab = (args) =>
  execFileSync("agent-browser", args, { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// Number of pixels that differ by more than a few levels in any channel, or
// Infinity when the two images cannot be compared (size, decode). Counting
// stops just past the budget. pngjs comes with the web app's dependencies.
//
// The budget is absolute, not a share of the frame: 0.05% of 1440×900 was
// 650 px, enough to hide a swapped 16×16 icon behind the tolerance. 128 px
// absorbs a caret, a spinner frame or antialiasing on a few glyphs; the
// count is printed per surface so the budget can be tuned from evidence.
const PIXEL_BUDGET = 128;
function pixelDiff(a, b) {
  let PNG;
  try {
    ({ PNG } = createRequire(join(root, "web", "package.json"))("pngjs"));
  } catch {
    return Infinity;
  }
  let x, y;
  try {
    x = PNG.sync.read(a);
    y = PNG.sync.read(b);
  } catch {
    return Infinity;
  }
  if (x.width !== y.width || x.height !== y.height) return Infinity;
  let diff = 0;
  for (let i = 0; i < x.data.length; i += 4) {
    if (
      Math.abs(x.data[i] - y.data[i]) > 8 ||
      Math.abs(x.data[i + 1] - y.data[i + 1]) > 8 ||
      Math.abs(x.data[i + 2] - y.data[i + 2]) > 8
    ) {
      if (++diff > PIXEL_BUDGET) return diff;
    }
  }
  return diff;
}

function gitSha() {
  try {
    return execFileSync("git", ["rev-parse", "HEAD"], { cwd: root, encoding: "utf8" }).trim();
  } catch {
    return "unknown";
  }
}

async function run({ base, release = null }) {
  mkdirSync(outDir, { recursive: true });
  const manifestPath = join(outDir, "manifest.json");
  const prev = (() => {
    try { return JSON.parse(readFileSync(manifestPath, "utf8")); } catch { return null; }
  })();

  // This run owns one session; other worktrees may be reviewing their UI.
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
        // A surface that needs an id minted by the seed resolves it in the
        // page and navigates there itself (a promise the CLI awaits).
        if (s.hashEval) {
          const went = ab([...sess, "eval", s.hashEval]);
          console.log(`    [${s.name} r${round}] ${went.trim().slice(0, 60)}`);
          await sleep(2500);
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
    const before = existsSync(out) ? readFileSync(out) : null;
    let shot = 0;
    for (;;) {
      ab([...sess, "screenshot", out]);
      shot += 1;
      if (statSync(out).size > 20000 || shot >= 3) break;
      await sleep(1500);
    }
    if (shot > 1) console.log(`    [${s.name}] blank frame — took ${shot} attempts`);
    // Capture noise (antialiasing, a caret, a 1 px reflow) changed a handful
    // of pixels in a third of the recaptures and each one became a binary
    // commit (ADR-0105). Keep the committed bytes when the frame is the same
    // picture within tolerance, so the sha and the repo stay put.
    if (before) {
      const d = pixelDiff(before, readFileSync(out));
      if (d <= PIXEL_BUDGET) {
        writeFileSync(out, before);
        console.log(`    [${s.name}] ${d} px differ (budget ${PIXEL_BUDGET}) — kept the committed image`);
      } else if (d !== Infinity) {
        console.log(`    [${s.name}] more than ${PIXEL_BUDGET} px differ — new capture`);
      }
    }

    captured[s.name] = {
      profile: s.profile,
      inputHash: surfaceFingerprint(root, s.profile, { pipeline: "screenshots" }),
      file,
      url: s.path,
      viewport: `${s.w}x${s.h}`,
      sha256: createHash("sha256").update(readFileSync(out)).digest("hex"),
    };
    console.log(`  ${s.name} -> ${file}`);
  }

  try { ab([...sess, "close"]); } catch { /* already gone */ }

  // When every surface kept its image and its inputs, the manifest keeps its
  // stamp too: a fresh capturedAt/gitSha alone made the tree dirty and each
  // deploy commit a two-line "refresh" with nothing refreshed (2026-09-09).
  const unchanged =
    prev &&
    prev.fingerprintVersion === FINGERPRINT_VERSION &&
    JSON.stringify(prev.surfaces) === JSON.stringify(captured);
  const manifest = {
    fingerprintVersion: FINGERPRINT_VERSION,
    capturedAt: unchanged ? prev.capturedAt : new Date().toISOString(),
    gitSha: unchanged ? prev.gitSha : gitSha(),
    base,
    tool: "agent-browser",
    note: "Generated by scripts/docs-shots.mjs — do not edit images by hand.",
    surfaces: captured,
  };
  writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + "\n");
  if (unchanged) console.log("manifest unchanged — nothing to commit");

  if (prev) {
    const changed = Object.keys(captured).filter(
      (k) => prev.surfaces?.[k]?.sha256 && prev.surfaces[k].sha256 !== captured[k].sha256,
    );
    if (changed.length) console.log(`changed since last run: ${changed.join(", ")}`);
  }
  console.log(`docs-shots: ${surfaces.length} surfaces -> ${outDir}`);
}

async function main() {
  let release = null;
  let base = externalBase;
  if (!externalBase) {
    const owned = await ownedFixture();
    release = () => owned.child.kill();
    base = owned.base;
  }
  try {
    await run({ base, release });
  } catch (e) {
    console.error("docs-shots failed:", e.message);
    process.exit(1);
  } finally {
    release?.();
  }
}

main();
