// Composer slash list. Matrix + statuses: docs/design/slash-parity.md
// Target: each /x opens PiCode UI. run:term is a recorded debt.
export const SLASH = [
  { id: "settings", label: "/settings", hint: "Pi JSON for this agent", run: "go-settings" },
  { id: "scoped-models", label: "/scoped-models", hint: "Models the chip cycles", run: "go-scoped" },
  { id: "login", label: "/login", hint: "Sign in a provider", run: "go-providers-new" },
  { id: "logout", label: "/logout", hint: "Sign out a provider", run: "go-providers" },
  { id: "copy", label: "/copy", hint: "Copy the last reply", run: "copy" },
  { id: "quit", label: "/quit", hint: "Stop this agent", run: "quit" },
  { id: "trust", label: "/trust", hint: "Trust this folder", run: "trust" },
  { id: "model", label: "/model", hint: "Jump to the model chip", run: "focus-model" },
  { id: "thinking", label: "/thinking", hint: "Jump to thinking level", run: "focus-thinking" },
  { id: "provider", label: "/provider", hint: "Jump to the provider chip", run: "focus-provider" },
  { id: "compact", label: "/compact", hint: "Summarize older context", run: "compact" },
  { id: "new", label: "/new", hint: "Start a blank session", run: "session-new" },
  { id: "resume", label: "/resume", hint: "Open a past session", run: "session-resume" },
  { id: "name", label: "/name", hint: "Rename this session", run: "session-name" },
  { id: "session", label: "/session", hint: "File, tokens, and cost", run: "session-info" },
  { id: "tree", label: "/tree", hint: "Prompts on a timeline", run: "session-tree" },
  { id: "fork", label: "/fork", hint: "New session from a prompt", run: "session-fork" },
  { id: "clone", label: "/clone", hint: "Copy this branch", run: "session-clone" },
  { id: "reload", label: "/reload", hint: "Reload skills and config", run: "reload" },
  { id: "export", label: "/export", hint: "Download this session", run: "export" },
  { id: "import", label: "/import", hint: "Resume from a JSONL file", run: "import" },
  { id: "hotkeys", label: "/hotkeys", hint: "PiCode shortcuts", run: "hotkeys" },
  { id: "changelog", label: "/changelog", hint: "pi version history", run: "changelog" },
  { id: "share", label: "/share", hint: "Secret GitHub gist", run: "share" },
  { id: "llama", label: "/llama", hint: "llama.cpp router models", run: "llama" },
];

export function extraSlash(skills, templates) {
  const out = [];
  for (const s of skills || []) {
    out.push({
      id: "skill:" + s.name,
      label: "/skill:" + s.name,
      hint: s.hint || "Skill",
      run: "insert",
      insert: "/skill:" + s.name + " ",
      docs: false,
    });
  }
  for (const t of templates || []) {
    out.push({
      id: "tpl:" + t.name,
      label: "/" + t.name,
      hint: t.hint || "Template",
      run: "insert",
      insert: "/" + t.name + " ",
      docs: false,
    });
  }
  return out;
}

export function filterSlash(q, extras) {
  const s = (q || "").trim();
  if (!s.startsWith("/")) return [];
  const needle = s.slice(1).toLowerCase();
  return SLASH.concat(extras || []).filter((c) => c.label.slice(1).startsWith(needle) || c.id.startsWith(needle));
}
