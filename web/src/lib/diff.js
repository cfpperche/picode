// File-change extraction from pi edit/write tool events.
// E1 opens the path; Keep/Undo is Track E3.

export function fileChangeFromTool(name, args, result) {
  if (!args || (name !== "edit" && name !== "write")) return null;
  const path = typeof args.path === "string" ? args.path : "";
  if (!path) return null;

  if (name === "write") {
    const content = typeof args.content === "string" ? args.content : "";
    const lines = content === "" ? [] : content.split("\n");
    return {
      path,
      add: lines.length,
      del: 0,
      hunks: lines.map((text) => ({ kind: "add", text })),
    };
  }

  const fromResult = parseOfficialDiff(result);
  if (fromResult) return { path, ...fromResult };

  const edits = normalizeEdits(args);
  const hunks = [];
  let add = 0, del = 0;
  for (const e of edits) {
    const oldLines = e.oldText === "" ? [] : e.oldText.split("\n");
    const newLines = e.newText === "" ? [] : e.newText.split("\n");
    for (const text of oldLines) { hunks.push({ kind: "del", text }); del++; }
    for (const text of newLines) { hunks.push({ kind: "add", text }); add++; }
    hunks.push({ kind: "gap", text: "" });
  }
  if (hunks.length && hunks[hunks.length - 1].kind === "gap") hunks.pop();
  return { path, add, del, hunks };
}

export function normalizeEdits(args) {
  if (Array.isArray(args.edits)) {
    return args.edits.filter((e) => e && typeof e.oldText === "string" && typeof e.newText === "string");
  }
  if (typeof args.oldText === "string" && typeof args.newText === "string") {
    return [{ oldText: args.oldText, newText: args.newText }];
  }
  return [];
}

export function parseOfficialDiff(result) {
  if (!result || typeof result !== "object") return null;
  const details = result.details || {};
  const raw = typeof details.patch === "string" && details.patch
    ? details.patch
    : (typeof details.diff === "string" ? details.diff : "");
  if (!raw) return null;
  return hunksFromDiff(raw);
}

// Unified diff text → hunks (conversation ```diff fences and tool patches).
export function hunksFromDiff(raw) {
  const hunks = [];
  let add = 0, del = 0;
  for (const line of String(raw || "").split("\n")) {
    if (line.startsWith("+++") || line.startsWith("---") || line.startsWith("diff ") || line.startsWith("index ")) {
      hunks.push({ kind: "meta", text: line });
      continue;
    }
    if (line.startsWith("@@")) {
      hunks.push({ kind: "meta", text: line });
      continue;
    }
    if (line.startsWith("+")) {
      hunks.push({ kind: "add", text: line.slice(1) });
      add++;
      continue;
    }
    if (line.startsWith("-")) {
      hunks.push({ kind: "del", text: line.slice(1) });
      del++;
      continue;
    }
    hunks.push({ kind: "ctx", text: line.startsWith(" ") ? line.slice(1) : line });
  }
  return { add, del, hunks };
}

export function groupHunks(hunks) {
  const groups = [];
  let cur = emptyGroup();
  function flush() {
    if (cur.dels.length || cur.adds.length) groups.push(cur);
    cur = emptyGroup();
  }
  for (const h of hunks || []) {
    if (h.kind === "gap" || h.kind === "meta") {
      flush();
      continue;
    }
    if (h.kind === "del") {
      if (cur.adds.length || cur.ctxAfter.length) flush();
      cur.dels.push(h.text);
      continue;
    }
    if (h.kind === "add") {
      cur.adds.push(h.text);
      continue;
    }
    if (h.kind === "ctx") {
      if (cur.dels.length || cur.adds.length) cur.ctxAfter.push(h.text);
      else cur.ctxBefore.push(h.text);
    }
  }
  flush();
  return groups;
}

function emptyGroup() {
  return { dels: [], adds: [], ctxBefore: [], ctxAfter: [] };
}

export function undoHunkInText(fileText, g) {
  const file = String(fileText ?? "").replace(/\r\n/g, "\n");
  if (!g) return { ok: false };
  const whole = !g.dels.length && !g.ctxBefore.length && !g.ctxAfter.length && g.adds.length;
  if (whole) return { ok: false, whole: true };
  const adds = g.adds.join("\n");
  const dels = g.dels.join("\n");
  const before = g.ctxBefore.join("\n");
  const after = g.ctxAfter.join("\n");
  if (g.adds.length) {
    let needle = adds;
    let put = dels;
    if (before) {
      needle = before + "\n" + needle;
      put = before + (put ? "\n" + put : "");
    }
    if (after) {
      needle = needle + "\n" + after;
      put = put + "\n" + after;
    }
    const i = file.indexOf(needle);
    if (i < 0) return { ok: false };
    return { ok: true, text: file.slice(0, i) + put + file.slice(i + needle.length) };
  }
  if (before && after) {
    const mid = before + "\n" + after;
    const i = file.indexOf(mid);
    if (i < 0) return { ok: false };
    const put = before + "\n" + dels + "\n" + after;
    return { ok: true, text: file.slice(0, i) + put + file.slice(i + mid.length) };
  }
  if (before) {
    const i = file.indexOf(before);
    if (i < 0) return { ok: false };
    return { ok: true, text: file.slice(0, i) + before + "\n" + dels + file.slice(i + before.length) };
  }
  return { ok: false };
}

export function basename(path) {
  if (!path) return "";
  const i = Math.max(path.lastIndexOf("/"), path.lastIndexOf("\\"));
  return i >= 0 ? path.slice(i + 1) : path;
}

export function statLabel(change) {
  if (!change) return "";
  const parts = [];
  if (change.add) parts.push("+" + change.add);
  if (change.del) parts.push("−" + change.del);
  return parts.join(" ");
}
