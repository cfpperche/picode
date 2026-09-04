#!/usr/bin/env node
// Classify a GitHub change set before the expensive CI jobs start.
// Unknown, empty, or mixed change sets fail safe to the complete matrix.

import { appendFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

function normalizePath(path) {
  return String(path).replaceAll("\\", "/").replace(/^\.\/+/, "");
}

export function pathScope(rawPath) {
  const path = normalizePath(rawPath);

  // These inputs have their own VitePress/parity/Vale job but cannot affect
  // the product binaries or runtime tests.
  if (
    path.startsWith("www/") ||
    path.startsWith("docs-videos/") ||
    path.startsWith("styles/") ||
    path === ".vale.ini" ||
    /^scripts\/docs-[^/]+\.mjs$/.test(path) ||
    path === "scripts/lib/uitree.mjs" ||
    path === "scripts/lib/docs-surfaces.mjs"
  ) {
    return "docs";
  }

  // Internal project records and root Markdown do not feed shipped assets.
  if (path.startsWith("docs/") || (!path.includes("/") && path.endsWith(".md"))) {
    return "metadata";
  }

  return "full";
}

export function classifyPaths(rawPaths) {
  const paths = [...new Set(rawPaths.map(normalizePath).filter(Boolean))];
  if (paths.length === 0) {
    return { scope: "full", full: true, docs: true, paths };
  }

  const scopes = new Set(paths.map(pathScope));
  if (scopes.has("full")) {
    return { scope: "full", full: true, docs: true, paths };
  }
  if (scopes.has("docs")) {
    return { scope: "docs", full: false, docs: true, paths };
  }
  return { scope: "metadata", full: false, docs: false, paths };
}

function validCommit(sha) {
  if (!sha || /^0+$/.test(sha)) return false;
  try {
    execFileSync("git", ["cat-file", "-e", `${sha}^{commit}`], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

export function changedPaths(base, head) {
  if (!validCommit(base) || !validCommit(head)) return [];
  const out = execFileSync("git", ["diff", "--name-only", "-z", base, head]);
  return out.toString("utf8").split("\0").filter(Boolean);
}

function markdownSummary(result) {
  const rows = result.paths.slice(0, 30).map((path) => `- \`${path.replaceAll("`", "\\`")}\``);
  if (result.paths.length > rows.length) rows.push(`- …and ${result.paths.length - rows.length} more`);
  return [
    "## CI scope",
    "",
    `**${result.scope}** — full matrix: \`${result.full}\`; public docs: \`${result.docs}\``,
    "",
    ...(rows.length ? rows : ["No trustworthy diff was available; running the full matrix."]),
    "",
  ].join("\n");
}

function main() {
  const base = process.env.CI_BASE_SHA ?? process.argv[2];
  const head = process.env.CI_HEAD_SHA ?? process.argv[3] ?? "HEAD";
  const result = classifyPaths(changedPaths(base, head));

  const output = process.env.GITHUB_OUTPUT;
  if (output) {
    appendFileSync(output, `scope=${result.scope}\nfull=${result.full}\ndocs=${result.docs}\n`);
  }
  const summary = process.env.GITHUB_STEP_SUMMARY;
  if (summary) appendFileSync(summary, markdownSummary(result));

  console.log(JSON.stringify(result, null, 2));
}

const invoked = process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url);
if (invoked) main();
