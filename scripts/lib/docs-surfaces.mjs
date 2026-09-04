import { createHash } from "node:crypto";
import { existsSync, readFileSync, statSync } from "node:fs";
import { dirname, extname, join, relative, resolve, sep } from "node:path";

// Bump whenever the hashing semantics change, so old manifests fail closed.
export const FINGERPRINT_VERSION = 1;

// Files shared by every captured surface. These are deliberately runtime
// inputs, not whole directories: tests, unrelated server handlers and docs do
// not change pixels. Screen components below bring their local imports along
// automatically, so a newly extracted child cannot escape the fingerprint.
const COMMON_FILES = [
  "web/index.html",
  "web/package.json",
  "web/package-lock.json",
  "web/vite.config.js",
  "web/src/main.jsx",
  "web/src/index.css",
  "web/src/styles/app.css",
  "cmd/picode-docs-fixture/main.go",
  "internal/server/server.go",
  "internal/server/workspaces.go",
  "internal/server/terminals.go",
  "internal/server/system.go",
  "internal/server/apps.go",
  "internal/server/tui_working.go",
  "internal/store/store.go",
  "internal/store/json.go",
  "internal/store/workspaces.go",
  "internal/store/agents.go",
  "internal/store/agent_sessions.go",
  "internal/store/terminals.go",
  "internal/store/providers.go",
  "internal/store/migrations/001_init.sql",
  "internal/store/migrations/004_free_workspace.sql",
  "internal/store/migrations/005_agent_work_path.sql",
  "internal/store/migrations/011_terminals.sql",
  "internal/store/migrations/013_terminal_workspace.sql",
];

const SHELLS = {
  desktop: {
    shallow: ["web/src/desktop/App.jsx"],
    logic: ["web/src/desktop/App.jsx", "web/src/main.jsx"],
    entries: ["web/src/components/Sidebar.jsx"],
  },
  mobile: {
    shallow: ["web/src/mobile/App.jsx", "web/src/mobile/mobile.css"],
    logic: ["web/src/mobile/App.jsx", "web/src/main.jsx"],
    entries: ["web/src/mobile/components/TabBar.jsx"],
  },
};

// Each profile names only the screen roots and response producers that can
// affect that capture. Local JSX/JS/CSS imports are followed recursively.
export const SURFACE_PROFILES = Object.freeze({
  "desktop-dashboard": {
    shell: "desktop",
    entries: ["web/src/components/DashboardView.jsx"],
    files: ["internal/server/session_ops.go", "internal/server/session_stats.go"],
  },
  "desktop-agents": {
    shell: "desktop",
    entries: ["web/src/components/DashboardView.jsx"],
    files: ["internal/server/session_ops.go", "internal/server/session_stats.go"],
  },
  "desktop-create-agent": {
    shell: "desktop",
    entries: ["web/src/components/DashboardView.jsx", "web/src/components/CreateForm.jsx"],
    files: ["internal/server/session_ops.go", "internal/server/session_stats.go"],
  },
  "desktop-agent": {
    shell: "desktop",
    entries: ["web/src/components/AgentTabs.jsx", "web/src/components/ChatSurface.jsx"],
    files: [
      "internal/server/agents.go",
      "internal/server/roles_state.go",
      "internal/server/slash_ops.go",
      "internal/server/slash_res.go",
    ],
  },
  "desktop-automations": {
    shell: "desktop",
    entries: ["web/src/components/Automations.jsx"],
    files: ["internal/server/automations.go", "internal/store/automations.go"],
  },
  "desktop-inbox": {
    shell: "desktop",
    entries: ["web/src/components/AgentTabs.jsx", "web/src/components/AppSurface.jsx"],
    files: [
      "internal/apps/apps.go",
      "internal/apps/inbox.go",
      "internal/apps/primitives.go",
      "internal/server/inbox.go",
      "internal/store/inbox.go",
      "internal/store/tasks.go",
    ],
  },
  "mobile-now": {
    shell: "mobile",
    entries: ["web/src/mobile/screens/Now.jsx"],
    files: [
      "internal/server/inbox.go",
      "internal/server/session_ops.go",
      "internal/server/session_stats.go",
      "internal/store/inbox.go",
    ],
  },
  "mobile-work": {
    shell: "mobile",
    entries: ["web/src/mobile/screens/Work.jsx"],
    files: [],
  },
  "mobile-agent": {
    shell: "mobile",
    entries: ["web/src/mobile/screens/Agent.jsx"],
    files: [
      "internal/server/agents.go",
      "internal/server/roles_state.go",
      "internal/server/slash_ops.go",
      "internal/server/slash_res.go",
    ],
  },
  "mobile-inbox": {
    shell: "mobile",
    entries: ["web/src/mobile/screens/Inbox.jsx"],
    files: [
      "internal/apps/apps.go",
      "internal/apps/inbox.go",
      "internal/apps/primitives.go",
      "internal/server/inbox.go",
      "internal/store/inbox.go",
      "internal/store/tasks.go",
    ],
  },
});

export const DOC_SCREENSHOT_SURFACES = Object.freeze({
  "app-fleet": "desktop-dashboard",
  "app-mobile-inbox": "mobile-inbox",
  "app-mobile": "mobile-now",
});

export const VIDEO_STILL_SURFACES = Object.freeze({
  "v1-1-dashboard": "desktop-dashboard",
  "v1-1b-agents-tab": "desktop-agents",
  "v1-2-form": "desktop-create-agent",
  "v1-3-running": "desktop-agent",
  "v2-1-list": "desktop-automations",
  "v2-2-detail": "desktop-automations",
  "v2-3-inbox": "desktop-inbox",
  "v3-1-fleet": "mobile-work",
  "v3-2-agent": "mobile-agent",
  "v3-3-inbox": "mobile-inbox",
});

export const TUTORIAL_VIDEOS = Object.freeze([
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
]);

const PIPELINE_FILES = {
  screenshots: "scripts/docs-shots.mjs",
  video: "scripts/docs-video-stills.mjs",
};

const SOURCE_EXTENSIONS = [".js", ".jsx", ".mjs", ".css", ".svg", ".json"];
const TEST_FILE = /(?:\.test\.[cm]?[jt]sx?|_test\.go)$/;

function posixPath(path) {
  return path.split(sep).join("/");
}

function checkedFile(root, rel) {
  const abs = resolve(root, rel);
  const inside = relative(resolve(root), abs);
  if (inside.startsWith("..") || inside === "") {
    throw new Error(`invalid docs surface input: ${rel}`);
  }
  if (!existsSync(abs) || !statSync(abs).isFile()) {
    throw new Error(`docs surface input missing: ${rel}`);
  }
  return posixPath(inside);
}

function localImports(source) {
  const specs = [];
  const patterns = [
    /\b(?:import|export)\s+(?:[^"']*?\s+from\s*)?["']([^"']+)["']/g,
    /\bimport\s*\(\s*["']([^"']+)["']\s*\)/g,
    /@import\s+(?:url\(\s*)?["']([^"']+)["']/g,
  ];
  for (const re of patterns) {
    for (const match of source.matchAll(re)) specs.push(match[1]);
  }
  return specs.filter((spec) => spec.startsWith("."));
}

function resolveLocalImport(root, from, spec) {
  const base = resolve(root, dirname(from), spec);
  const candidates = extname(base)
    ? [base]
    : [
        base,
        ...SOURCE_EXTENSIONS.map((ext) => base + ext),
        ...SOURCE_EXTENSIONS.map((ext) => join(base, "index" + ext)),
      ];
  for (const abs of candidates) {
    if (!existsSync(abs) || !statSync(abs).isFile()) continue;
    const rel = posixPath(relative(resolve(root), abs));
    if (rel.startsWith("..")) throw new Error(`docs surface import escapes root: ${from} -> ${spec}`);
    return rel;
  }
  throw new Error(`docs surface import missing: ${from} -> ${spec}`);
}

function collectImports(root, entry, files, allow = () => true, walked = new Set()) {
  const rel = checkedFile(root, entry);
  if (TEST_FILE.test(rel) || walked.has(rel)) return;
  walked.add(rel);
  files.add(rel);
  if (!/[.](?:[cm]?[jt]sx?|css)$/.test(rel)) return;
  const source = readFileSync(join(root, rel), "utf8");
  for (const spec of localImports(source)) {
    const unresolved = posixPath(relative(resolve(root), resolve(root, dirname(rel), spec)));
    if (!allow(unresolved)) continue;
    const target = resolveLocalImport(root, rel, spec);
    if (allow(target)) collectImports(root, target, files, allow, walked);
  }
}

function shellLogicInput(path) {
  return path.startsWith("web/src/lib/") || path.startsWith("web/src/mobile/hooks/") || path.endsWith(".css");
}

export function surfaceInputFiles(root, profile, { pipeline = "screenshots" } = {}) {
  const spec = SURFACE_PROFILES[profile];
  if (!spec) throw new Error(`unknown docs surface profile: ${profile}`);
  const shell = SHELLS[spec.shell];
  const pipelineFile = PIPELINE_FILES[pipeline];
  if (!pipelineFile) throw new Error(`unknown docs capture pipeline: ${pipeline}`);

  const files = new Set();
  for (const rel of [...COMMON_FILES, ...shell.shallow, ...spec.files, pipelineFile]) {
    const checked = checkedFile(root, rel);
    if (!TEST_FILE.test(checked)) files.add(checked);
  }
  const shellWalked = new Set();
  for (const entry of shell.logic) collectImports(root, entry, files, shellLogicInput, shellWalked);
  const screenWalked = new Set();
  for (const entry of [...shell.entries, ...spec.entries]) {
    collectImports(root, entry, files, () => true, screenWalked);
  }
  return [...files].sort();
}

export function surfaceFingerprint(root, profile, options = {}) {
  const pipeline = options.pipeline ?? "screenshots";
  const files = surfaceInputFiles(root, profile, { pipeline });
  const hash = createHash("sha256");
  hash.update(`picode-docs-surface-v${FINGERPRINT_VERSION}\0${pipeline}\0${profile}\0`);
  for (const rel of files) {
    hash.update(rel + "\0");
    hash.update(readFileSync(join(root, rel)));
  }
  return hash.digest("hex");
}

export function screenshotInputFailures(root, manifest) {
  const fails = [];
  if (manifest.fingerprintVersion !== FINGERPRINT_VERSION) {
    fails.push("screenshot fingerprint schema changed — run `make docs-shots`");
  }
  for (const [name, profile] of Object.entries(DOC_SCREENSHOT_SURFACES)) {
    const captured = manifest.surfaces?.[name];
    if (!captured) {
      fails.push(`${name}: missing from screenshot manifest`);
      continue;
    }
    if (captured.profile !== profile) {
      fails.push(`${name}: surface profile changed — run \`make docs-shots\``);
      continue;
    }
    const current = surfaceFingerprint(root, profile, { pipeline: "screenshots" });
    if (captured.inputHash !== current) {
      fails.push(`${name}: ${profile} inputs changed — run \`make docs-shots\``);
    }
  }
  for (const name of Object.keys(manifest.surfaces ?? {})) {
    if (!DOC_SCREENSHOT_SURFACES[name]) fails.push(`${name}: unknown screenshot surface in manifest`);
  }
  return fails;
}
