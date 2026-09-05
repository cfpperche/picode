// Roadmap A3: a whole-draft `!cmd` is a shell line for the agent cwd.
// `!!` (hidden output) stays a TUI feature. Mixed prose is a prompt.
export function bashLine(draft) {
  const s = (draft || "").trim();
  if (!s.startsWith("!")) return null;
  if (s.startsWith("!!")) return { refused: "!!" };
  const command = s.slice(1).trim();
  if (!command) return null; // bare "!" is a no-op
  return { command };
}
