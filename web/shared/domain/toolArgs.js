// One-line summary of a tool call's arguments for a collapsed row. Plain
// JS (no JSX) so the agent-event reducer and its node tests can use it.
import { currentStep, countDone } from "./checklist.js";

export function summarizeArgs(args) {
  if (!args) return "";
  if (Array.isArray(args.items)) {
    // A checklist call (ADR-0055): progress and the current step, not JSON.
    const step = currentStep(args.items);
    return step ? countDone(args.items) + "/" + step.total + " · " + step.text : "";
  }
  if (typeof args.action === "string") {
    // The computer tool (ADR-0148) and any action-shaped call: the action,
    // then the one argument that says where or what.
    const where = Array.isArray(args.coordinate) ? " [" + args.coordinate.join(",") + "]"
      : typeof args.text === "string" ? " " + JSON.stringify(args.text.length > 40 ? args.text.slice(0, 37) + "…" : args.text)
      : typeof args.target === "string" ? " " + args.target
      : typeof args.window === "number" ? " window " + args.window : "";
    return args.action + where;
  }
  if (typeof args.query === "string") return args.query;
  if (typeof args.command === "string") return args.command;
  if (typeof args.path === "string") return args.path;
  const s = JSON.stringify(args);
  return s.length > 2 ? s : "";
}
