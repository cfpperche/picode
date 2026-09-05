// The name a source is known by: npm:pi-web-search → pi-web-search,
// ../packages/pi-inbox → pi-inbox, git:github.com/u/repo → repo.
export function pkgName(source) {
  let s = String(source || "").trim();
  s = s.replace(/^(npm|git|https?):(\/\/)?/, "");
  s = s.replace(/[\/\\]+$/, "");
  if (s.startsWith("@")) return s.split("@").slice(0, 2).join("@"); // scoped npm, drop a pinned version
  const last = s.split(/[\/\\]/).pop() || s;
  return last.replace(/@[^@]*$/, "").replace(/\.git$/, "") || s;
}
