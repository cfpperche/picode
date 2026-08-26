export function safeImgSrc(src) {
  const s = String(src || "").trim();
  if (!s) return "";
  const low = s.toLowerCase();
  if (low.startsWith("https://") || low.startsWith("http://")) return s;
  if (low.startsWith("data:image/")) return s;
  return "";
}
