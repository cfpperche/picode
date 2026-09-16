import { test } from "node:test";
import assert from "node:assert/strict";
import { requestBrowserDialog, takeBrowserDialog } from "./browserDialogs.js";

test("a request survives until the page takes it, once", () => {
  requestBrowserDialog("history");
  assert.equal(takeBrowserDialog(), "history");
  assert.equal(takeBrowserDialog(), "");
});

test("an unknown dialog name is ignored, not carried over", () => {
  requestBrowserDialog("teleport");
  assert.equal(takeBrowserDialog(), "");
});

test("the last request wins when the menu is used twice", () => {
  requestBrowserDialog("downloads");
  requestBrowserDialog("wipe");
  assert.equal(takeBrowserDialog(), "wipe");
});
