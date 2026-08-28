export function groupTurns(items) {
  const out = [];
  let cur = emptyTurn();
  function flush() {
    if (!cur.user && cur.work.length === 0 && cur.replies.length === 0 && cur.loose.length === 0) return;
    out.push(cur);
    cur = emptyTurn();
  }
  for (const it of items || []) {
    if (it.kind === "sys" || it.kind === "files" || it.kind === "bash" || it.kind === "ask") {
      if (cur.user || cur.work.length || cur.replies.length) flush();
      out.push({ kind: "loose", item: it });
      continue;
    }
    if (it.kind === "block" && it.cls === "user") {
      flush();
      cur.user = it;
      continue;
    }
    if (it.kind === "tool" || (it.kind === "block" && it.cls === "thinking")) {
      cur.work.push(it);
      continue;
    }
    if (it.kind === "alert") {
      cur.replies.push(it);
      continue;
    }
    if (it.kind === "block") {
      cur.replies.push(it);
      continue;
    }
    cur.loose.push(it);
  }
  flush();
  return out;
}

// Index of the turn that is actually running. A user message sent while
// another turn is still open is queued — it must not look like Working.
export function workingIndex(turns, streaming) {
  if (!streaming) return -1;
  let lastPending = -1;
  let lastBusy = -1;
  turns.forEach((t, i) => {
    if (t.kind !== "turn" || t.replies.length) return;
    lastPending = i;
    if (t.work.length) lastBusy = i;
  });
  return lastBusy >= 0 ? lastBusy : lastPending;
}

export function pathsFromTurn(turn) {
  const out = [];
  const seen = new Set();
  for (const it of (turn && turn.work) || []) {
    const p = it.change && it.change.path;
    if (!p || seen.has(p)) continue;
    seen.add(p);
    out.push(p);
  }
  return out;
}

function emptyTurn() {
  return { kind: "turn", user: null, work: [], replies: [], loose: [] };
}

export function turnDurationMs(turn) {
  const stamps = [];
  for (const it of [turn.user, ...turn.work, ...turn.replies]) {
    if (it && it.ts) stamps.push(Number(it.ts));
  }
  if (stamps.length < 2) return 0;
  const a = Math.min(...stamps);
  const b = Math.max(...stamps);
  return Math.max(0, b - a);
}

export function fmtElapsed(ms) {
  const s = Math.max(0, Math.round((ms || 0) / 1000));
  if (s < 60) return s + "s";
  const m = Math.floor(s / 60);
  const r = s % 60;
  return r ? m + "m " + r + "s" : m + "m";
}

export function fmtWorked(ms) {
  if (!ms || ms < 400) return "Worked";
  return "Worked for " + fmtElapsed(ms);
}

export function dayKey(ts) {
  if (!ts) return "";
  const d = new Date(Number(ts));
  if (Number.isNaN(d.getTime())) return "";
  return d.getFullYear() + "-" + (d.getMonth() + 1) + "-" + d.getDate();
}

export function fmtDayMark(ts, now = Date.now()) {
  if (!ts) return "";
  const d = new Date(Number(ts));
  if (Number.isNaN(d.getTime())) return "";
  const time = d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
  const today = new Date(now);
  const yday = new Date(now - 86400000);
  const same = (a, b) => a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
  if (same(d, today)) return "Today at " + time;
  if (same(d, yday)) return "Yesterday at " + time;
  return d.toLocaleDateString(undefined, { weekday: "short", month: "short", day: "numeric" }) + " at " + time;
}

export function firstTs(turn) {
  const stamps = [];
  for (const it of [turn.user, ...(turn.work || []), ...(turn.replies || [])]) {
    if (it && it.ts) stamps.push(Number(it.ts));
  }
  return stamps.length ? Math.min(...stamps) : 0;
}

export function stepLabel(it) {
  if (!it) return "";
  if (it.cls === "thinking") return "Thought";
  if (it.kind !== "tool") return it.actor || "";
  const n = String(it.name || "").toLowerCase();
  const arg = String(it.args || "").trim();
  const clip = (s) => (s.length > 64 ? s.slice(0, 61) + "…" : s);
  if (n.includes("web_search") || n.includes("websearch") || n === "search") {
    return arg ? "Searched " + clip(arg) : "Searched the web";
  }
  if (n === "read") return arg ? "Read " + base(arg) : "Read a file";
  if (n === "write" || n === "edit") return arg ? "Edited " + base(arg) : "Edited a file";
  if (n === "bash") return arg ? "Ran " + clip(arg) : "Ran a command";
  if (n === "grep" || n === "find") return arg ? "Searched " + clip(arg) : "Searched files";
  if (n === "ls") return arg ? "Listed " + clip(arg) : "Listed files";
  return it.name + (arg ? " " + clip(arg) : "");
}

function base(p) {
  const s = p.replace(/^@/, "").replace(/\\/g, "/");
  const i = s.lastIndexOf("/");
  return i >= 0 ? s.slice(i + 1) : s;
}
