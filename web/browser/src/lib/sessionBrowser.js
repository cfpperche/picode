// Which host tab a session-browser command binds to (ADR-0172).
// Pure: the page opens the split; this only names the tab the split hangs off.
//
// The session the human is watching wins. A caller that names a terminal
// binds to that terminal's tab when it is open; otherwise to the agent's
// bound terminal; otherwise to the agent tab. A command never falls through
// to whatever tab happens to be selected.

import { termTabId } from "./routes.js";

export function sessionHostTab({ agentId = "", termId = "", openTabs = [], agentTerminalId = "" } = {}) {
  const tabs = new Set(openTabs || []);
  const termTab = termId ? termTabId(termId) : "";
  const bound = agentTerminalId ? termTabId(agentTerminalId) : "";
  if (termTab && tabs.has(termTab)) return termTab;
  if (bound && tabs.has(bound)) return bound;
  if (agentId && tabs.has(agentId)) return agentId;
  if (termTab) return termTab;
  if (bound) return bound;
  if (agentId) return agentId;
  return "";
}

// What a session command does to the human's view (ADR-0172). Only `open`
// and the command that creates the split bring the session forward; every
// other verb runs in the split where it is, so an agent driving its page does
// not pull the human away from the tab they are using.
//
//   host tab open | split exists | method     | action
//   no            | any          | any        | "open"   (open + select the host)
//   yes           | no           | any        | "select"
//   yes           | yes          | shell.open | "select"
//   yes           | yes          | other      | "none"
export function sessionReveal({ hostOpen = false, splitExists = false, method = "" } = {}) {
  if (!hostOpen) return "open";
  if (!splitExists || method === "shell.open") return "select";
  return "none";
}
