export const DOCS_BASE = "https://cfpperche.github.io/picode";

export function commandDocUrl(id) {
  const slug = String(id || "").toLowerCase();
  if (!/^[a-z0-9-]+$/.test(slug)) return DOCS_BASE + "/commands";
  return DOCS_BASE + "/commands#" + slug;
}
