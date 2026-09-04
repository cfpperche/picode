#!/usr/bin/env node
// docs-video-manifest — copy rendered tutorial MP4s into www/public/video/
// and write the parity manifest.
//
// The manifest records, per video: sha256 of the composition source, every
// still it uses, the rendered MP4, and the UI tree hash captured at render
// time. `--check` is the fast CI floor: committed inputs and both MP4 copies
// must match, but unrelated UI-tree drift does not block delivery. `--fresh`
// adds that strict tree comparison for an explicit maintenance audit.

import { createHash } from "node:crypto";
import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { uiTreeHash } from "./lib/uitree.mjs";

const scriptPath = fileURLToPath(import.meta.url);
const root = join(dirname(scriptPath), "..");
const checkOnly = process.argv.includes("--check") || process.argv.includes("--fresh");
const requireFresh = process.argv.includes("--fresh");

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

export function buildManifest(rootDir = root) {
  const project = join(rootDir, "docs-videos");
  const stillsDir = join(project, "assets", "stills");
  const tree = uiTreeHash(rootDir);
  const videos = {};
  for (const v of VIDEOS) {
    const mp4Path = join(project, "renders", v.mp4);
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

export function manifestFailures(
  committed,
  fresh,
  { strict = false, publishedHash = () => null } = {},
) {
  const fails = [];
  if (strict && committed.uiTreeHash !== fresh.uiTreeHash) {
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
    for (const v of VIDEOS) {
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
      ? "videos-fresh ok: tutorial videos match the current UI tree"
      : "videos-check ok: tutorial video inputs and shipped files match the manifest",
  );
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(scriptPath)) main();
