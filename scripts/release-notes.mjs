#!/usr/bin/env node

import { readFileSync } from "node:fs";

const [, , input, flag] = process.argv;
const version = String(input || "").trim().replace(/^v/i, "");

if (!/^\d+\.\d+\.\d+$/.test(version)) {
  console.error("usage: node scripts/release-notes.mjs <major.minor.patch> [--check]");
  process.exit(2);
}

const changelog = readFileSync("CHANGELOG.md", "utf8");
const escaped = version.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const heading = new RegExp(`^## \\[${escaped}\\](?:[^\\n]*)$`, "m");
const match = heading.exec(changelog);
if (!match) {
  console.error(`CHANGELOG.md has no release section for ${version}`);
  process.exit(1);
}

const start = match.index;
const next = changelog.indexOf("\n## [", start + match[0].length);
const section = changelog.slice(start, next === -1 ? changelog.length : next).trim();
const body = section.slice(match[0].length).trim();
if (!body || !/^###\s+/m.test(body)) {
  console.error(`CHANGELOG.md release section for ${version} is empty`);
  process.exit(1);
}

let catalog;
try {
  catalog = JSON.parse(readFileSync("web/src/data/whats-new.json", "utf8"));
} catch (error) {
  console.error(`web/src/data/whats-new.json could not be read: ${error.message}`);
  process.exit(1);
}
const entry = Array.isArray(catalog) ? catalog.find((item) => item && item.version === version) : null;
if (!entry || !Array.isArray(entry.highlights) || !entry.highlights.length) {
  console.error(`web/src/data/whats-new.json has no highlights for ${version}`);
  process.exit(1);
}

if (flag === "--check") {
  console.log(`release notes found for ${version}`);
} else if (flag) {
  console.error(`unknown flag: ${flag}`);
  process.exit(2);
} else {
  process.stdout.write(section.replace(match[0], `# PiCode v${version}`) + "\n");
}
