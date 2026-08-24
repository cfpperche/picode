import { fileChangeFromTool } from "./diff.js";

function summarizeArgs(args) {
  if (!args) return "";
  if (typeof args.command === "string") return args.command;
  if (typeof args.path === "string") return args.path;
  try {
    const s = JSON.stringify(args);
    return s.length > 2 ? s : "";
  } catch { return ""; }
}

export function eventsToItems(events) {
  return (events || []).map((e, i) => {
    if (e.kind === "user") {
      return { kind: "block", cls: "user", actor: "You", text: e.text || "" };
    }
    if (e.kind === "thinking") {
      return { kind: "block", cls: "thinking", actor: "thinking", text: e.text || "" };
    }
    if (e.kind === "tool") {
      const args = e.toolArgs || {};
      return {
        kind: "tool",
        id: e.id || ("tool-" + i),
        name: e.name || "tool",
        args: e.args || summarizeArgs(args),
        status: e.status === "···" ? "ok" : (e.status || "ok"),
        detail: e.detail || "",
        expanded: false,
        change: fileChangeFromTool(e.name, args, e.result),
      };
    }
    return { kind: "block", cls: "", actor: "agent", text: e.text || "" };
  });
}
