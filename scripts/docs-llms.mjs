#!/usr/bin/env node
// docs-llms — generate www/public/llms.txt from the docs sources.
//
// llms.txt is the emerging convention for giving LLMs a curated map of a
// site (llmstxt.org): H1 title, blockquote summary, then H2 sections of
// markdown links. We derive it from the VitePress config + page
// frontmatter so it cannot drift from the site structure. The API
// section always points at the generated OpenAPI spec.
//
//   make llms            # write www/public/llms.txt
//   node scripts/docs-llms.mjs --check   # exit 1 if the file is stale

import { createHash } from "node:crypto";
import { existsSync, readFileSync, readdirSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const www = join(root, "www");
// Public base URL — must match www/.vitepress/config.mjs `base` + the
// Pages origin.
const BASE = "https://cfpperche.github.io/picode";

const argv = process.argv.slice(2);
const checkOnly = argv.includes("--check");
const outIdx = argv.indexOf("--out");
const outPath = outIdx >= 0 ? argv[outIdx + 1] : join(www, "public", "llms.txt");

// ── collect pages in sidebar/IA order ──────────────────────────────────
// Importing .vitepress/config.mjs would drag VitePress in; instead read
// the sidebar structure by convention: root *.md, then guide/, then any
// *.md at depth 1. Order = curated below, then alphabetical extras.
const CURATED = [
  ["Start", "index.md", "What is PiCode"],
  ["Start", "guide/getting-started.md", "Getting started"],
  ["Run it somewhere", "guide/security.md", "Security and pairing"],
  ["Run it somewhere", "guide/remote-server.md", "On a server"],
  ["Run it somewhere", "guide/shared-server.md", "Share one server"],
  ["Run it somewhere", "guide/public-access.md", "Open it to the internet"],
  ["Run it somewhere", "guide/mobile.md", "On your phone"],
  ["Guides", "guide/providers.md", "Providers"],
  ["Guides", "guide/packages.md", "Packages"],
  ["Guides", "guide/checklist.md", "Checklist"],
  ["Guides", "guide/compact.md", "Compact earlier"],
  ["Guides", "guide/mcp.md", "MCP"],
  ["Guides", "guide/terminal-status.md", "Terminal status for CLIs"],
  ["Guides", "guide/llama.md", "llama.cpp"],
  ["Guides", "guide/browser-extension.md", "Chrome extension"],
  ["Guides", "guide/automations.md", "Automations"],
  ["Reference", "commands.md", "Commands"],
  ["Reference", "guide/settings.md", "Settings"],
  ["Reference", "api.md", "HTTP API"],
  ["Reference", "license.md", "License"],
];

function pageMeta(rel) {
  const p = join(www, rel);
  if (!existsSync(p)) return null;
  const raw = readFileSync(p, "utf8");
  const lines = raw.split("\n");
  let title = "";
  let description = "";
  let heroText = "";
  let heroTagline = "";
  // frontmatter block — also accepts VitePress hero text/tagline (the
  // homepage has no prose description); first match wins, so the hero
  // block beats the action buttons that reuse `text:` later in the file
  if (lines[0]?.trim() === "---") {
    for (let i = 1; i < lines.length && lines[i].trim() !== "---"; i++) {
      const m = lines[i].match(/^\s*(title|description|text|tagline):\s*(.+?)\s*$/);
      if (m) {
        const v = m[2].replace(/^"|"$/g, "");
        if (m[1] === "title" && !title) title = v;
        else if (m[1] === "description" && !description) description = v;
        else if (m[1] === "text" && !heroText) heroText = v;
        else if (m[1] === "tagline" && !heroTagline) heroTagline = v;
      }
    }
    description = description || heroTagline || heroText;
  }
  // first H1 fallback
  if (!title) {
    const h1 = lines.find((l) => /^#\s+/.test(l));
    if (h1) title = h1.replace(/^#\s+/, "").trim();
  }
  // first prose paragraph fallback — BODY only (after the closing
  // frontmatter fence; require a real sentence, not a stray marker)
  if (!description) {
    let inCode = false;
    const body = lines[0]?.trim() === "---"
      ? lines.slice(1 + lines.slice(1).findIndex((l) => l.trim() === "---"))
      : lines;
    for (const l of body) {
      if (l.trim().startsWith("```")) { inCode = !inCode; continue; }
      if (inCode || l.startsWith("#") || l.startsWith("|") || l.trim() === "---") continue;
      if (/^\s*[\w-]+:/.test(l)) continue; // YAML/hero keys, not prose
      const text = l.trim().replace(/[*_`]/g, "");
      if (text.length < 40) continue;
      description = text;
      break;
    }
  }
  description = (description || "").slice(0, 220);
  return { rel, title, description };
}

function sourceHash() {
  const h = createHash("sha256");
  const walk = (dir) => {
    for (const e of readdirSync(dir, { withFileTypes: true })) {
      const p = join(dir, e.name);
      if (e.isDirectory()) {
        if (["node_modules", ".vitepress", "public", "img"].includes(e.name)) continue;
        walk(p);
      } else if (e.name.endsWith(".md")) {
        h.update(relative(www, p) + "\n");
        h.update(readFileSync(p));
      }
    }
  };
  walk(www);
  return h.digest("hex");
}

const docSections = new Map();
for (const [section, rel, label] of CURATED) {
  const meta = pageMeta(rel);
  if (!meta) continue;
  if (!docSections.has(section)) docSections.set(section, []);
  const href = rel === "index.md" ? BASE + "/" : `${BASE}/${rel.replace(/\.md$/, "")}`;
  docSections
    .get(section)
    .push(`- [${label}](${href}): ${meta.description || meta.title}`);
}

const lines = [
  "# PiCode",
  "",
  "> PiCode is a browser-based Agent Development Environment (ADE) for Pi coding",
  "> agents: one Go daemon serves a web UI to create, configure and orchestrate",
  "> agents across workspaces — designed for people who avoid terminals.",
  "",
];
for (const [section, items] of docSections) {
  lines.push(`## ${section}`, "", ...items, "");
}
lines.push(
  "## Machine-readable",
  "",
  `- [OpenAPI spec](${BASE}/api/openapi.json): every HTTP route the daemon serves, generated from the server's route registration (206+ operations).`,
  `- [API reference UI](${BASE}/api/): the same spec in Scalar's browsable viewer.`,
  "",
);

const content = lines.join("\n");

if (checkOnly) {
  if (!existsSync(outPath)) {
    console.error(`llms-check FAILED: ${outPath} missing — run \`make llms\``);
    process.exit(1);
  }
  if (readFileSync(outPath, "utf8") !== content) {
    console.error("llms-check FAILED: llms.txt is stale (docs sources changed) — run `make llms`");
    process.exit(1);
  }
  console.log(`llms-check ok (${sourceHash().slice(0, 12)})`);
} else {
  writeFileSync(outPath, content);
  console.log(`llms.txt written: ${outPath} (${content.split("\n").length} lines)`);
}
