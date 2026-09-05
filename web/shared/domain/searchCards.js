export function isSearchTool(name) {
  const n = String(name || "").toLowerCase();
  return n === "web_search" || n === "url_context" || n.includes("web_search") || n.includes("websearch") || n === "search";
}

export function searchQuery(args) {
  if (!args) return "";
  if (typeof args === "string") {
    try { args = JSON.parse(args); } catch { return args; }
  }
  if (typeof args.query === "string") return args.query;
  return "";
}

export function hitsFromTool(it) {
  if (!it) return [];
  const a = hitsFromResult(it.result);
  if (a.length) return a;
  return hitsFromResult(parseJSON(it.detail));
}

export function hitsFromResult(raw) {
  if (raw == null) return [];
  if (typeof raw === "string") {
    const obj = parseJSON(raw);
    if (obj) return hitsFromResult(obj);
    return parseMarkdownSources(raw);
  }
  if (typeof raw !== "object") return [];
  const details = raw.details && typeof raw.details === "object" ? raw.details : raw;
  const out = [];
  const seen = new Set();
  const lists = [details.sources, details.searchResults, raw.sources, raw.searchResults];
  for (const list of lists) {
    if (!Array.isArray(list)) continue;
    for (const s of list) {
      if (!s || typeof s !== "object") continue;
      const url = String(s.url || s.link || "").trim();
      if (!url || seen.has(url)) continue;
      seen.add(url);
      out.push({
        title: String(s.title || s.name || hostOf(url)),
        url,
        snippet: String(s.snippet || s.description || "").slice(0, 180),
      });
    }
  }
  if (out.length) return out.slice(0, 12);
  return parseMarkdownSources(contentText(raw)).slice(0, 12);
}

function parseJSON(s) {
  if (typeof s !== "string" || !s.trim()) return null;
  try { return JSON.parse(s); } catch { return null; }
}

function contentText(raw) {
  const parts = raw && raw.content;
  if (!Array.isArray(parts)) return typeof raw.text === "string" ? raw.text : "";
  return parts.map((p) => (p && p.text) || "").join("\n");
}

function parseMarkdownSources(text) {
  const out = [];
  const seen = new Set();
  const re = /\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g;
  let m;
  while ((m = re.exec(String(text || "")))) {
    const url = m[2];
    if (seen.has(url)) continue;
    seen.add(url);
    out.push({ title: m[1], url, snippet: "" });
  }
  return out;
}

function hostOf(url) {
  try { return new URL(url).hostname; } catch { return url; }
}
