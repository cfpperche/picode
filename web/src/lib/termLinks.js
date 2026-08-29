// Ctrl/Cmd+click in xterm: http → browser; path under cwd → file tab.
// No modifier = select. Paths outside cwd are not links.

export function hasOpenModifier(ev) {
  return !!(ev && (ev.ctrlKey || ev.metaKey));
}

export function stripLineCol(s) {
  let t = String(s || "");
  t = t.replace(/[:#](\d+)(?:[:-]\d+)?$/, "");
  t = t.replace(/[.,;!?)]+$/g, "");
  return t;
}

export function normalizeAbs(p) {
  const parts = String(p || "").split("/");
  const out = [];
  for (const part of parts) {
    if (!part || part === ".") continue;
    if (part === "..") {
      out.pop();
      continue;
    }
    out.push(part);
  }
  return "/" + out.join("/");
}

export function underCwd(cwd, abs) {
  if (!cwd || !abs || String(abs).startsWith("~/")) return false;
  const c = normalizeAbs(cwd);
  const a = normalizeAbs(abs);
  return a === c || a.startsWith(c + "/");
}

export function relPath(cwd, abs) {
  const c = normalizeAbs(cwd);
  const a = normalizeAbs(abs);
  if (a === c) return "";
  if (a.startsWith(c + "/")) return a.slice(c.length + 1);
  return "";
}

export function resolvePath(cwd, path) {
  if (!path) return "";
  if (path.startsWith("~/")) return path;
  if (path.startsWith("/")) return normalizeAbs(path);
  if (!cwd) return "";
  return normalizeAbs(String(cwd).replace(/\/$/, "") + "/" + path);
}

export function classify(raw, cwd) {
  const s = String(raw || "").trim();
  if (!s) return null;
  if (/^https?:\/\//i.test(s)) {
    try {
      const u = new URL(s);
      if (u.protocol !== "http:" && u.protocol !== "https:") return null;
      return { kind: "http", href: u.href };
    } catch {
      return null;
    }
  }
  if (/^(javascript|data|vbscript|vscode):/i.test(s)) return null;
  let path = s;
  if (/^file:\/\//i.test(path)) {
    try {
      const u = new URL(path);
      if (u.protocol !== "file:") return null;
      path = decodeURIComponent(u.pathname);
    } catch {
      return null;
    }
  }
  path = stripLineCol(path);
  if (!path || path === "/" || path === "~" || path === "." || path === "..") return null;
  if (path.startsWith("~/")) return { kind: "file", path };
  const abs = resolvePath(cwd, path);
  if (!abs || !underCwd(cwd, abs)) return null;
  const rel = relPath(cwd, abs);
  return { kind: "file", path: rel || path };
}

const HTTP_RE = /\bhttps?:\/\/[^\s<>"'\\]+/gi;
const PATH_RE = /(?:file:\/\/[^\s<>"'\\]+|(?:~\/|\.\/|\.\.\/)[^\s<>"'\\]+|(?<=^|[\s(["'])\/[^\s<>"'\\]+|(?<=^|[\s(["'])(?:[\w.@+-]+\/)+[\w.@+-]+)/g;

export function findLinks(line, cwd) {
  const s = String(line || "");
  const hits = [];
  const seen = [];
  function add(start, raw) {
    const end = start + raw.length;
    if (start < 0 || end <= start) return;
    if (seen.some((r) => start < r.end && end > r.start)) return;
    const hit = classify(raw, cwd);
    if (!hit) return;
    seen.push({ start, end });
    hits.push({ start, end, raw, kind: hit.kind, href: hit.href, path: hit.path });
  }
  let m;
  const http = new RegExp(HTTP_RE.source, "gi");
  while ((m = http.exec(s))) add(m.index, m[0]);
  const path = new RegExp(PATH_RE.source, "g");
  while ((m = path.exec(s))) add(m.index, m[0]);
  hits.sort((a, b) => a.start - b.start);
  return hits;
}

export function wireTermLinks(term, getCwd, onFile) {
  if (!term || typeof term.registerLinkProvider !== "function") return () => {};
  const activate = (ev, text) => {
    if (!hasOpenModifier(ev)) return;
    const hit = classify(text, typeof getCwd === "function" ? getCwd() : getCwd);
    if (!hit) return;
    if (hit.kind === "http") {
      window.open(hit.href, "_blank", "noopener,noreferrer");
      return;
    }
    if (hit.kind === "file" && onFile) onFile(hit.path);
  };
  term.options.linkHandler = {
    activate,
    allowNonHttpProtocols: true,
  };
  const disp = term.registerLinkProvider({
    provideLinks(y, cb) {
      const buf = term.buffer && term.buffer.active;
      const row = buf && buf.getLine(y - 1);
      if (!row) {
        cb(undefined);
        return;
      }
      const text = row.translateToString(true);
      const cwd = typeof getCwd === "function" ? getCwd() : getCwd;
      const links = findLinks(text, cwd).map((l) => ({
        text: l.raw,
        range: { start: { x: l.start + 1, y }, end: { x: l.start + l.raw.length, y } },
        activate: (ev) => activate(ev, l.raw),
      }));
      cb(links.length ? links : undefined);
    },
  });
  return () => {
    try { disp.dispose(); } catch { /* ignore */ }
  };
}
