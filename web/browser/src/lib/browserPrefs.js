// One reader for the browser preferences the daemon returns.
//
// The settings page reads them twice — once on load, once after every save
// (the PUT answers with the saved object) — and the two copies drifted: the
// save path forgot a field, so the switch it belonged to snapped back off
// after every click. One function, one shape.

export const DEFAULT_BROWSER_PREFS = {
  showFullUrl: true,
  webOpenDest: "app",
  localOpenDest: "app",
  passwordAutosave: true,
  generalAutofill: true,
  askDownload: false,
  scriptsEnabled: true,
  agentAccess: true,
  developerMode: false,
};

// readBrowserPrefs folds a payload into the shape the page holds: missing
// fields take the product's default, so an older daemon cannot blank a row.
export function readBrowserPrefs(payload) {
  const p = payload && typeof payload === "object" ? payload : {};
  return {
    showFullUrl: p.showFullUrl !== false,
    webOpenDest: p.webOpenDest || DEFAULT_BROWSER_PREFS.webOpenDest,
    localOpenDest: p.localOpenDest || DEFAULT_BROWSER_PREFS.localOpenDest,
    passwordAutosave: p.passwordAutosave !== false,
    generalAutofill: p.generalAutofill !== false,
    askDownload: p.askDownload === true,
    scriptsEnabled: p.scriptsEnabled !== false,
    agentAccess: p.agentAccess !== false,
    // Fail closed like the daemon does (ADR-0144): a payload that omits it
    // must not read as "raw CDP is on".
    developerMode: p.developerMode === true,
  };
}
