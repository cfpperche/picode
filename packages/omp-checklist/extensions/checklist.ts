// omp-checklist (ADR-0055): omp's native todo tool is the plan, and this
// extension mirrors its committed snapshot to PiCode's checklist routes so
// the sidebar shows the current step — the projection pi-checklist gives pi.
// No tool is registered and nothing is gated: the todo contract, omp's own
// reminder and the TUI stay authoritative. Silently inert outside a PiCode
// terminal (no identity env) and outside the TUI.
import { readFileSync } from "node:fs";
import { request as httpRequest } from "node:http";
import { request as httpsRequest } from "node:https";
import { homedir } from "node:os";
import { join } from "node:path";

const STATUS = { pending: "pending", in_progress: "in-progress", completed: "completed" };
const MAX_ITEMS = 50; // the store's checklist cap (internal/store/checklists.go)

function dataDir() {
  const explicit = (process.env.PICODE_DATA || "").trim();
  return explicit || homedir().replace(/\/+$/, "") + "/.picode";
}

function readOptional(path) {
  try {
    return readFileSync(path, "utf8");
  } catch {
    return null;
  }
}

// mcptool.ResolveURL's order: an explicit origin, the terminal's own
// instance, then server.json in the data dir.
function serverUrl() {
  for (const key of ["PICODE_URL", "PICODE_TERM_URL"]) {
    const v = (process.env[key] || "").trim();
    if (v && /^https?:\/\/[^/\s@]+\/?$/.test(v)) return v.replace(/\/+$/, "");
  }
  const raw = readOptional(join(dataDir(), "server.json"));
  if (raw) {
    try {
      const parsed = JSON.parse(raw);
      if (typeof parsed.url === "string" && /^https?:\/\//.test(parsed.url)) return parsed.url.replace(/\/+$/, "");
    } catch {
      // a broken server.json is no server
    }
  }
  return null;
}

function publishToken() {
  const fromEnv = (process.env.PICODE_TOKEN || "").trim();
  if (/^[0-9a-f]{32,128}$/i.test(fromEnv)) return fromEnv;
  const fromFile = (readOptional(join(dataDir(), "token")) || "").trim();
  return /^[0-9a-f]{32,128}$/i.test(fromFile) ? fromFile : "";
}

// pi-checklist's publishTarget: a bound agent id wins, the terminal id is
// the second branch, and with neither there is nothing to publish to.
function targetPath() {
  const agent = (process.env.PICODE_AGENT_ID || "").trim();
  if (/^[A-Za-z0-9_.-]{1,128}$/.test(agent)) return "/api/agents/" + encodeURIComponent(agent) + "/checklist";
  const term = (process.env.PICODE_TERM_ID || "").trim();
  if (/^[A-Za-z0-9_.-]{1,128}$/.test(term)) return "/api/terminals/" + encodeURIComponent(term) + "/checklist";
  return null;
}

function rejectUnauthorizedFor(url) {
  try {
    const host = new URL(url).hostname.replace(/^\[|\]$/g, "");
    return !(host === "localhost" || host === "127.0.0.1" || host === "::1");
  } catch {
    return true;
  }
}

function flatten(phases) {
  const items = [];
  const multi = (phases || []).length > 1;
  for (const phase of phases || []) {
    for (const task of (phase && phase.tasks) || []) {
      if (!task || typeof task.content !== "string" || !task.content.trim()) continue;
      if (task.status === "abandoned") continue;
      let text = multi && phase.name ? phase.name + " — " + task.content : task.content;
      if (task.status === "blocked" && task.blocker) text += " (blocked: " + task.blocker + ")";
      items.push({ text: text.trim(), status: STATUS[task.status] || "pending" });
      if (items.length >= MAX_ITEMS) return items;
    }
  }
  return items;
}

// The last committed todo snapshot on the branch (pi-checklist's
// branching-safe reconstruct idiom; omp session entries are pi-shaped).
function committedPhases(entries) {
  let found = null;
  for (const entry of entries || []) {
    const message = entry && entry.type === "message" && entry.message;
    if (!message || message.role !== "toolResult" || message.toolName !== "todo") continue;
    if (message.details && Array.isArray(message.details.phases)) found = message.details.phases;
  }
  return found;
}

export default function ompChecklist(pi) {
  let posting = Promise.resolve();
  let sessionId = "";

  function publish(payload) {
    const path = targetPath();
    const base = serverUrl();
    if (!path || !base) return;
    posting = posting.then(() => new Promise((resolve) => {
      try {
        const body = JSON.stringify(payload);
        const url = new URL(base + path);
        const send = url.protocol === "https:" ? httpsRequest : httpRequest;
        const headers = { "content-type": "application/json", "content-length": Buffer.byteLength(body) };
        const token = publishToken();
        if (token) headers.authorization = "Bearer " + token;
        const req = send(url, { method: "POST", headers, rejectUnauthorized: rejectUnauthorizedFor(url.toString()), timeout: 5000 }, (res) => {
          res.resume();
          res.on("end", resolve);
        });
        req.on("timeout", () => req.destroy(new Error("timeout")));
        req.on("error", resolve);
        req.end(body);
      } catch {
        resolve();
      }
    })).catch(() => {});
  }

  // An empty plan is a reset: whatever the row holds is about older work,
  // and the sidebar shows nothing until this session writes tasks again.
  // An unknown session publishes without the id, like pi-checklist.
  function publishPhases(phases) {
    const items = flatten(phases);
    const payload = {};
    if (sessionId) payload.sessionId = sessionId;
    if (items.length) payload.items = items;
    else payload.reset = true;
    publish(payload);
  }

  pi.on("session_start", async (_event, ctx) => {
    if (!ctx || ctx.mode !== "tui") return;
    try {
      sessionId = (ctx.sessionManager.getSessionId() || "") + "";
    } catch {
      sessionId = "";
    }
    let phases = null;
    try {
      phases = committedPhases(ctx.sessionManager.getBranch());
    } catch {
      phases = null;
    }
    publishPhases(phases);
  });

  pi.on("tool_result", async (event, ctx) => {
    if (!ctx || ctx.mode !== "tui") return;
    if (!event || event.toolName !== "todo") return;
    const phases = event.details && event.details.phases;
    if (!Array.isArray(phases)) return;
    publishPhases(phases);
  });
}
