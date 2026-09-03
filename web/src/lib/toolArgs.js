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
  if (typeof args.query === "string") return args.query;
  if (typeof args.command === "string") return args.command;
  if (typeof args.path === "string") return args.path;
  const s = JSON.stringify(args);
  return s.length > 2 ? s : "";
}
