// One request from the work-tab options menu to the Settings ▸ Browser page.
// The menu lives with the page; the dialog lives in Settings (option A, owner
// 2026-09-15), so the request waits in a module slot across the route change
// and the page takes it when it becomes visible. Nothing else should grow
// here — a second surface wanting the same dialogs is the moment to extract
// them instead.

const DIALOGS = new Set(["history", "downloads", "wipe", "passwords", "contact"]);

let pending = "";

export function requestBrowserDialog(name) {
  pending = DIALOGS.has(name) ? name : "";
}

// takeBrowserDialog() returns the request once and clears it.
export function takeBrowserDialog() {
  const value = pending;
  pending = "";
  return value;
}
