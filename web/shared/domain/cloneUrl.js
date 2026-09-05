// Pure helpers for the Clone-repository mode of the New-workspace form.
// Tolerant mirror of the server's gitclone.ParseRemote: the server is the
// authority; here bad input just yields "" so the form derives nothing.

export function deriveRepo(raw) {
  let s = String(raw || "").trim();
  if (!s || /[\s;|&$`<>'"\\]/.test(s) || s.startsWith("-")) return { name: "", branch: "" };
  const scheme = /^(https?|ssh|git):\/\//.test(s);
  const scpLike = !scheme && /^[^\s/]+@[^\s/:]+:[^\s]+$/.test(s);
  if (!scheme && !scpLike) return { name: "", branch: "" };
  let branch = "";
  if (scheme) {
    const i = s.indexOf("/tree/");
    if (i > 0) {
      branch = s.slice(i + 6).replace(/\/+$/, "");
      s = s.slice(0, i);
    }
  }
  s = s.replace(/\/+$/, "");
  const rest = scheme ? s.replace(/^\w+:\/\/[^/]+\/?/, "") : s.slice(s.indexOf(":") + 1);
  if (!rest) return { name: "", branch: "" };
  const name = (rest.split("/").pop() || "").replace(/\.git$/, "");
  if (!name || name === "." || name === "..") return { name: "", branch: "" };
  return { name, branch };
}

export function looksLikeRepoUrl(raw) {
  return deriveRepo(raw).name !== "";
}

// cloneDest joins the remembered parent folder with the repo name.
export function cloneDest(parent, name) {
  const p = String(parent || "").replace(/\/+$/, "") || "~";
  return p + "/" + name;
}

// parentDir is the folder above path, "" when there is none to remember.
export function parentDir(path) {
  const s = String(path || "").replace(/\/+$/, "");
  const i = s.lastIndexOf("/");
  return i > 0 ? s.slice(0, i) : "";
}
