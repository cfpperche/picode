#!/usr/bin/env node
// docs-video-manifest — copy rendered tutorial MP4s into www/public/video/
// and write the parity manifest.
//
// The manifest records, per video: sha256 of the composition source, every
// still it uses, the rendered MP4, and the input fingerprint for that still's
// named UI surface. `--check` is the fast CI floor: committed inputs and both
// MP4 copies must match. `--fresh` additionally reports only the tutorials
// whose actual UI surfaces drifted since capture.

import { createHash } from "node:crypto";
import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import {
  FINGERPRINT_VERSION,
  TUTORIAL_VIDEOS,
  VIDEO_STILL_SURFACES,
  surfaceFingerprint,
} from "./lib/docs-surfaces.mjs";

const scriptPath = fileURLToPath(import.meta.url);
const root = join(dirname(scriptPath), "..");
const checkOnly = process.argv.includes("--check") || process.argv.includes("--fresh");
const requireFresh = process.argv.includes("--fresh");

const sha = (p) => createHash("sha256").update(readFileSync(p)).digest("hex");

export function buildManifest(rootDir = root) {
  const project = join(rootDir, "docs-videos");
  const stillsDir = join(project, "assets", "stills");
  const videos = {};
  for (const v of TUTORIAL_VIDEOS) {
    const mp4Path = join(project, "renders", v.mp4);
    const compPath = join(project, v.file);
    const stills = {};
    for (const st of v.stills) {
      const p = join(stillsDir, `${st}.png`);
      const surface = VIDEO_STILL_SURFACES[st];
      if (!surface) throw new Error(`${v.id}: still ${st} has no UI surface profile`);
      stills[st] = {
        sha256: existsSync(p) ? sha(p) : "MISSING",
        surface,
        inputHash: surfaceFingerprint(rootDir, surface, { pipeline: "video" }),
      };
    }
    videos[v.id] = {
      file: v.mp4,
      composition: v.file,
      compositionHash: sha(compPath),
      stills,
      mp4: existsSync(mp4Path) ? sha(mp4Path) : "MISSING",
    };
  }
  return { fingerprintVersion: FINGERPRINT_VERSION, videos };
}

export function manifestFailures(
  committed,
  fresh,
  { strict = false, publishedHash = () => null } = {},
) {
  const fails = [];
  if (committed.fingerprintVersion !== fresh.fingerprintVersion) {
    fails.push("video fingerprint schema changed — run `make docs-videos`");
  }
  for (const [id, v] of Object.entries(fresh.videos)) {
    const c = committed.videos?.[id];
    if (!c) {
      fails.push(`${id}: missing from manifest`);
      continue;
    }
    if (c.compositionHash !== v.compositionHash) {
      fails.push(`${id}: composition changed since render — run \`make docs-videos\``);
    }
    const staleSurfaces = new Set();
    for (const [st, captured] of Object.entries(v.stills)) {
      const prior = c.stills?.[st];
      if (captured.sha256 === "MISSING") {
        fails.push(`${id}: still ${st} missing — run \`make docs-videos\``);
      } else if (!prior || prior.sha256 !== captured.sha256) {
        fails.push(`${id}: still ${st} changed since render — run \`make docs-videos\``);
      }
      if (
        strict &&
        (!prior || prior.surface !== captured.surface || prior.inputHash !== captured.inputHash)
      ) {
        staleSurfaces.add(captured.surface);
      }
    }
    for (const surface of staleSurfaces) {
      fails.push(`${id}: ${surface} inputs changed since capture — run \`make docs-videos\``);
    }
    const shipped = publishedHash(v.file);
    if (!shipped) {
      fails.push(`www/public/video/${v.file} missing`);
    } else if (c.mp4 === "MISSING" || c.mp4 !== v.mp4 || c.mp4 !== shipped) {
      fails.push(`${id}: rendered MP4 does not match manifest — run \`make docs-videos\``);
    }
  }
  return fails;
}

function main() {
  const project = join(root, "docs-videos");
  const outDir = join(root, "www", "public", "video");
  const fresh = buildManifest();
  if (!checkOnly) {
    mkdirSync(outDir, { recursive: true });
    for (const v of TUTORIAL_VIDEOS) {
      copyFileSync(join(project, "renders", v.mp4), join(outDir, v.mp4));
    }
    writeFileSync(join(outDir, "manifest.json"), JSON.stringify(fresh, null, 2) + "\n");
    console.log(`docs-video-manifest: ${Object.keys(fresh.videos).length} videos -> ${outDir}`);
    return;
  }

  const manifestPath = join(outDir, "manifest.json");
  let fails = [];
  if (!existsSync(manifestPath)) {
    fails.push("www/public/video/manifest.json missing — run `make docs-videos`");
  } else {
    const committed = JSON.parse(readFileSync(manifestPath, "utf8"));
    fails = manifestFailures(committed, fresh, {
      strict: requireFresh,
      publishedHash: (file) => {
        const mp4 = join(outDir, file);
        return existsSync(mp4) ? sha(mp4) : null;
      },
    });
  }
  if (fails.length) {
    console.error(`${requireFresh ? "videos-fresh" : "videos-check"} FAILED:`);
    for (const f of fails) console.error("  - " + f);
    process.exit(1);
  }
  console.log(
    requireFresh
      ? "videos-fresh ok: tutorial videos match their current UI surfaces"
      : "videos-check ok: tutorial video inputs and shipped files match the manifest",
  );
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(scriptPath)) main();
