// Header globe: launcher for the work browser.
// Pure on purpose — click, Shift+click and the right-click menu share one
// table, so the component stays a renderer and the rows stay testable
// without a browser.
//
// Bind target matches the pane's "Open browser" (ADR-0135): an agent tab,
// or a terminal that still wears an agent CLI. A bare shell has no agent
// to bind, so the globe falls through to a tab with no agent.

import { isAgentTab, isTermTab } from "./routes.js";
import { terminalDisplayCli } from "@picode/shared/domain/terminalCli.js";

const SEP = { sep: true };

export const GLOBE_MENU_KEYS = {
  "new-tab": "Shift+Click",
};

// globeBindId is the agentPanes key for the selected tab, or "" when this
// tab cannot host a split.
export function globeBindId(selectedId, terminal) {
  const id = selectedId || "";
  if (!id) return "";
  if (isAgentTab(id)) return id;
  if (isTermTab(id) && terminalDisplayCli(terminal)) return id;
  return "";
}

// globeClickAction is the left-click table. "noop" is a split that is
// already open: minting another webview would orphan the live one.
export function globeClickAction({ shiftKey = false, bindId = "", splitOn = false } = {}) {
  if (shiftKey) return "new-tab";
  if (bindId && !splitOn) return "split";
  if (bindId && splitOn) return "noop";
  return "new-tab";
}

export function buildGlobeMenu({ tabId = "", splitOn = false } = {}) {
  const rows = [];
  if (tabId) {
    rows.push(splitOn
      ? { id: "close-browser", label: "Close browser split", icon: "x" }
      : { id: "open-browser", label: "Open browser", icon: "globe" });
    rows.push(SEP);
  }
  rows.push({ id: "new-tab", label: "Open in new tab", icon: "plus", key: GLOBE_MENU_KEYS["new-tab"] });
  return rows;
}
