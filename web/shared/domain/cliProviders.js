// Native credentials are a separate capability from terminal launches.
export const CLI_PROVIDERS = [{ id: "pi", name: "Pi" }];
export const supportsCliProviders = id => CLI_PROVIDERS.some(cli => cli.id === id);

export function cliProvidersHash(cli = "pi", { add = false, custom = false, customId = "" } = {}) {
  const tail = custom ? "/custom" + (customId ? "/" + encodeURIComponent(customId) : "") : add ? "/new" : "";
  return "#/clis/" + encodeURIComponent(cli || "pi") + "/providers" + tail;
}

export function cliProvidersLocation(hash = "") {
  const [path, query = ""] = hash.replace(/^#/, "").split("?");
  // This compatibility alias belongs to the independent llama.cpp manager.
  if (path === "/providers/llama" || path === "/more/providers/llama") return null;
  const legacy = /^\/(?:more\/)?providers(?:\/|$)/.test(path);
  if (!legacy && path !== "/clis/providers" && !path.startsWith("/clis/providers/")) return null;
  const tail = legacy ? path.replace(/^\/(?:more\/)?providers\/?/, "") : path.slice("/clis/providers".length).replace(/^\//, "");
  const parts = tail.split("/");
  let id = "pi";
  if (!legacy && path !== "/clis/providers") {
    try { id = decodeURIComponent(parts[0]); } catch { id = ""; }
  }
  const add = legacy ? tail === "new" : parts.length === 2 && parts[1] === "new";
  const invalid = legacy ? tail !== "" && tail !== "new" : parts.length > 1 && !add;
  const params = new URLSearchParams(query);
  const scoped = ["agentId", "workspaceId", "scope"].some(key => params.has(key));
  // ADR-0103 amendment 2026-09-11: Providers live on the CLI's pane. The
  // strip address and the old top-level aliases rewrite onto that pane.
  return { view: "clis", id, pane: "providers", add, invalid, scoped, legacy,
    redirect: !invalid && !scoped && id ? cliProvidersHash(id, { add }) : "" };
}

// Preserve the explicit app path (and its theme/layout query), clear the
// add route so OAuth completion cannot reopen a fresh login dialog.
export function cliProvidersReturnTo(href) {
  const url = new URL(href);
  url.hash = cliProvidersHash();
  return url.href;
}
