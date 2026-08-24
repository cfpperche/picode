// Native pi slash commands we surface in the composer (parity with the TUI).
// run: login | focus-model | focus-thinking | focus-provider | term
export const SLASH = [
  { id: "login", label: "/login", hint: "Sign in a provider", run: "login" },
  { id: "logout", label: "/logout", hint: "Clear credentials", run: "term" },
  { id: "model", label: "/model", hint: "Switch model", run: "focus-model" },
  { id: "thinking", label: "/thinking", hint: "Thinking level", run: "focus-thinking" },
  { id: "provider", label: "/provider", hint: "Switch provider", run: "focus-provider" },
  { id: "compact", label: "/compact", hint: "Compact context", run: "term" },
  { id: "new", label: "/new", hint: "New session", run: "term" },
  { id: "resume", label: "/resume", hint: "Resume a session", run: "term" },
  { id: "session", label: "/session", hint: "Session info", run: "term" },
  { id: "tree", label: "/tree", hint: "Session tree", run: "term" },
  { id: "reload", label: "/reload", hint: "Reload skills and config", run: "term" },
];

export function filterSlash(q) {
  const s = (q || "").trim();
  if (!s.startsWith("/")) return [];
  const needle = s.slice(1).toLowerCase();
  return SLASH.filter((c) => c.label.slice(1).startsWith(needle) || c.id.startsWith(needle));
}
