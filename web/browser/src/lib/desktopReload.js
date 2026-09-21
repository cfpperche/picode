// Desktop reload (Ctrl+R / F5). WebView2's engine accelerators stay on the
// work-page webviews; the chrome child turns them off so a terminal can keep
// Ctrl+R (readline reverse-search) and a visible work tab reloads the page
// instead of PiCode. This module is the chrome's decision table.

import { boundWorkTab, tabWebId } from "./routes.js";

export function isReloadKey(ev) {
  if (!ev || ev.altKey || ev.shiftKey || ev.repeat || ev.isComposing || ev.keyCode === 229) return false;
  if (ev.key === "F5") return !ev.ctrlKey && !ev.metaKey;
  return !!(ev.ctrlKey || ev.metaKey) && String(ev.key).toLowerCase() === "r";
}

// null → leave the chord to the terminal. `{kind:"page", id}` reloads that
// work-browser webview. `{kind:"app"}` reloads PiCode.
export function desktopReloadAction({ inTerminal, selectedTab, panes, onPane }) {
  if (inTerminal) return null;
  if (!onPane) {
    const tab = boundWorkTab(selectedTab, panes);
    if (tab) return { kind: "page", id: tabWebId(tab) };
  }
  return { kind: "app" };
}
