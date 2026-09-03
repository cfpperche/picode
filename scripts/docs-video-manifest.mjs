#!/usr/bin/env node
// docs-video-manifest — copy rendered tutorial MP4s into www/videos/ and
// write the parity manifest.
//
// The manifest records, per video: sha256 of the composition source, of
// every still it uses, of the MP4, and the UI tree hash. `--check` (used by
// docs-check) regenerates that data and byte-compares: a UI/server change
// without re-capturing + re-rendering fails, exactly like the screenshots
// gate. MP4 bytes are NOT compared — encoder metadata is not reproducible —
// but the manifest must match and the file must exist.

import { createHash } from "node:crypto";
import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { uiTreeHash } from "./lib/uitree.mjs";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const project = join(root, "docs-videos");
const outDir = join(root, "www", "public", "video");
const checkOnly = process.argv.includes("--check");

const sha = (p) => createHash("sha256").update(readFileSync(p)).digest("hex");

const VIDEOS = [
  {
    id: "create-agent",
    file: "index.html",
    mp4: "create-agent.mp4",
    stills: ["v1-1b-agents-tab", "v1-2-form", "v1-3-running"],
  },
  {
    id: "automate-it",
    file: "compositions/automate-it.html",
    mp4: "automate-it.mp4",
    stills: ["v2-1-list", "v2-2-detail", "v2-3-inbox"],
  },
  {
    id: "take-it-anywhere",
    file: "compositions/take-it-anywhere.html",
    mp4: "take-it-anywhere.mp4",
    stills: ["v3-1-fleet", "v3-2-agent", "v3-3-inbox"],
  },
];

const stillsDir = join(project, "assets", "stills");

function buildManifest() {
  const tree = uiTreeHash(root);
  const videos = {};
  for (const v of VIDEOS) {
    const mp4Path = join(root, "docs-videos", "renders", v.mp4);
    const compPath = join(project, v.file);
    const stills = {};
    for (const st of v.stills) {
      const p = join(stillsDir, `${st}.png`);
      stills[st] = existsSync(p) ? sha(p) : "MISSING";
    }
    videos[v.id] = {
      file: v.mp4,
      composition: v.file,
      compositionHash: sha(compPath),
      stills,
      mp4: existsSync(mp4Path) ? sha(mp4Path) : "MISSING",
    };
  }
  // uiTreeHash covers web/src + server + fixture + pipeline; the videos'
  // own inputs live in the per-video hashes above.
  return { uiTreeHash: tree, videos };
}

const fresh = buildManifest();

if (checkOnly) {
  const manifestPath = join(outDir, "manifest.json");
  const fails = [];
  if (!existsSync(manifestPath)) {
    fails.push("www/public/video/manifest.json missing — run `make docs-videos`");
  } else {
    const committed = JSON.parse(readFileSync(manifestPath, "utf8"));
    if (committed.uiTreeHash !== fresh.uiTreeHash) {
      fails.push("video manifest is stale (UI tree changed) — run `make docs-videos`");
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
      for (const [st, h] of Object.entries(v.stills)) {
        if (c.stills?.[st] !== h) {
          fails.push(`${id}: still ${st} changed since render — run \`make docs-videos\``);
        }
      }
      const mp4 = join(outDir, v.file);
      if (!existsSync(mp4)) {
        fails.push(`www/public/video/${v.file} missing`);
      } else if (c.mp4 !== v.mp4 || c.mp4 === "MISSING") {
        fails.push(`${id}: rendered MP4 does not match manifest — run \`make docs-videos\``);
      }
    }
  }
  if (fails.length) {
    console.error("videos-check FAILED:");
    for (const f of fails) console.error("  - " + f);
    process.exit(1);
  }
  console.log("videos-check ok: tutorial videos match the current tree");
} else {
  mkdirSync(outDir, { recursive: true });
  for (const v of VIDEOS) {
    copyFileSync(join(root, "docs-videos", "renders", v.mp4), join(outDir, v.mp4));
  }
  writeFileSync(join(outDir, "manifest.json"), JSON.stringify(fresh, null, 2) + "\n");
  console.log(`docs-video-manifest: ${Object.keys(fresh.videos).length} videos -> ${outDir}`);
}
