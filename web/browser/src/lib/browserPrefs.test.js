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
      agentAccess: false,
    };
  assert.deepEqual(readBrowserPrefs(saved), saved);
});

test("askDownload survives the round trip the save path makes", () => {
  const on = readBrowserPrefs({ ...DEFAULT_BROWSER_PREFS, askDownload: true });
  assert.equal(on.askDownload, true);
  assert.equal(readBrowserPrefs(on).askDownload, true);
});

test("readBrowserPrefs defaults what a payload omits", () => {
  assert.deepEqual(readBrowserPrefs({}), DEFAULT_BROWSER_PREFS);
  assert.deepEqual(readBrowserPrefs(null), DEFAULT_BROWSER_PREFS);
  const odd = readBrowserPrefs({ agentAccess: undefined, askDownload: "yes" });
  assert.equal(odd.agentAccess, true);
  assert.equal(odd.askDownload, false);
  assert.equal(odd.webOpenDest, "app");
});
