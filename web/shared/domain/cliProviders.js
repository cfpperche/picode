// Every CLI that has a Providers pane, which is every CLI PiCode can read a
// credential for: pi writes auth.json through the vault like the rest, one
// pane serves all nine (ADR-0166, ADR-0169). Keys and names are the launch
// catalog's ids and terminalCli's labels, so one CLI never has two spellings
// in the product.
export const CLI_PROVIDERS = [
  { id: "pi", name: "Pi" },
  { id: "claude-code", name: "Claude Code" },
  { id: "codex", name: "Codex" },
  { id: "grok", name: "Grok" },
  { id: "hermes", name: "Hermes Agent" },
  { id: "opencode", name: "OpenCode" },
  { id: "muse", name: "Muse Code" },
  { id: "agy", name: "Antigravity" },
  { id: "omp", name: "Omp" },
];
export const supportsCliProviders = id => CLI_PROVIDERS.some(cli => cli.id === id);

// Custom provider definitions live in each CLI's own file; today pi
// (models.json, ADR-0129) and omp (models.yml, the ADR-0169 amendment) read
// them. The roster's `custom` door only appears where the server can write.
export const supportsCustomProviders = id => id === "pi" || id === "omp";

export function cliProvidersHash(cli = "pi", { add = false, custom = false, customId = "" } = {}) {
  const tail = custom ? "/custom" + (customId ? "/" + encodeURIComponent(customId) : "") : add ? "/new" : "";
  return "#/clis/" + encodeURIComponent(cli || "pi") + "/providers" + tail;
}

// Preserve the explicit app path (and its theme/layout query), clear the
// add route so OAuth completion cannot reopen a fresh login dialog.
export function cliProvidersReturnTo(href) {
  const url = new URL(href);
  url.hash = cliProvidersHash();
  return url.href;
}
