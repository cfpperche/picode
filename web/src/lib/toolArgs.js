// One-line summary of a tool call's arguments for a collapsed row. Plain
// JS (no JSX) so the agent-event reducer and its node tests can use it.
export function summarizeArgs(args) {
  if (!args) return "";
  if (typeof args.query === "string") return args.query;
  if (typeof args.command === "string") return args.command;
  if (typeof args.path === "string") return args.path;
  const s = JSON.stringify(args);
  return s.length > 2 ? s : "";
}
