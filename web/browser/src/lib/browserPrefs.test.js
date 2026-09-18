import { test } from "node:test";
import assert from "node:assert/strict";
import { DEFAULT_BROWSER_PREFS, readBrowserPrefs } from "./browserPrefs.js";

// The save path answers with the saved object; if the reader drops a field,
// the switch it belongs to turns itself off on every click. This is that
// regression: askDownload came back undefined from the PUT's answer.

test("readBrowserPrefs keeps every field of a full payload", () => {
    const saved = {
      showFullUrl: false,
      webOpenDest: "external",
      localOpenDest: "app",
      passwordAutosave: false,
      generalAutofill: false,
      askDownload: true,
      scriptsEnabled: false,
      historyAccess: "allow",
      agentAccess: false,
      developerMode: true,
      annotationShots: "always",
    };
  assert.deepEqual(readBrowserPrefs(saved), saved);
});

test("developer mode defaults off and survives a round trip", () => {
  assert.equal(DEFAULT_BROWSER_PREFS.developerMode, false);
  assert.equal(readBrowserPrefs({}).developerMode, false);
  const on = readBrowserPrefs({ ...DEFAULT_BROWSER_PREFS, developerMode: true });
  assert.equal(readBrowserPrefs(on).developerMode, true);
});

test("askDownload survives the round trip the save path makes", () => {
  const on = readBrowserPrefs({ ...DEFAULT_BROWSER_PREFS, askDownload: true });
  assert.equal(on.askDownload, true);
  assert.equal(readBrowserPrefs(on).askDownload, true);
});

test("history access fails closed and survives a round trip", () => {
  // ADR-0146: only the exact word "allow" opens it — a missing field, a
  // boolean, a typo and a stale payload all read as never.
  assert.equal(DEFAULT_BROWSER_PREFS.historyAccess, "never");
  for (const payload of [{}, null, { historyAccess: undefined }, { historyAccess: true }, { historyAccess: "sometimes" }]) {
    assert.equal(readBrowserPrefs(payload).historyAccess, "never", JSON.stringify(payload));
  }
  const on = readBrowserPrefs({ ...DEFAULT_BROWSER_PREFS, historyAccess: "allow" });
  assert.equal(on.historyAccess, "allow");
  assert.equal(readBrowserPrefs(on).historyAccess, "allow");
});

test("readBrowserPrefs defaults what a payload omits", () => {
  assert.deepEqual(readBrowserPrefs({}), DEFAULT_BROWSER_PREFS);
  assert.deepEqual(readBrowserPrefs(null), DEFAULT_BROWSER_PREFS);
  const odd = readBrowserPrefs({ agentAccess: undefined, askDownload: "yes" });
  assert.equal(odd.agentAccess, true);
  assert.equal(odd.askDownload, false);
  assert.equal(odd.webOpenDest, "app");
});

test("the annotation screenshot policy defaults to ask and never widens", () => {
  assert.equal(readBrowserPrefs({}).annotationShots, "ask");
  assert.equal(readBrowserPrefs({ annotationShots: "always" }).annotationShots, "always");
  assert.equal(readBrowserPrefs({ annotationShots: "never" }).annotationShots, "never");
  assert.equal(readBrowserPrefs({ annotationShots: "sometimes" }).annotationShots, "ask");
});
