import assert from "node:assert/strict";
import { test } from "node:test";
import { linkOpenTarget } from "./openLink.js";

// The decision table in openLink.js, one row per case.
test("linkOpenTarget walks its decision table", () => {
  // file path — the file pane, whatever the preference says.
  assert.deepEqual(
    linkOpenTarget({ kind: "file", path: "src/App.jsx" }, "external"),
    { action: "file", path: "src/App.jsx" },
  );
  // http elsewhere — always the system browser, whatever the preference says.
  assert.deepEqual(
    linkOpenTarget({ kind: "http", href: "https://example.com/x" }, "app"),
    { action: "external", url: "https://example.com/x" },
  );
  // http loopback — the preference decides, and its default is PiCode.
  assert.equal(linkOpenTarget({ kind: "http", href: "http://localhost:5173/" }, "app").action, "app");
  assert.equal(linkOpenTarget({ kind: "http", href: "http://127.0.0.1:8000/" }, undefined).action, "app");
  assert.deepEqual(
    linkOpenTarget({ kind: "http", href: "http://localhost:5173/" }, "external"),
    { action: "external", url: "http://localhost:5173/" },
  );
  // every loopback spelling the client already knows (brackets are required
  // for IPv6, and a URL without them is not a URL at all).
  for (const href of ["http://[::1]:3000/", "https://dev.localhost:5173/", "http://0.0.0.0:8080/", "http://127.9.9.9:5000/"]) {
    assert.equal(linkOpenTarget({ kind: "http", href }, "app").action, "app", href);
  }
  // no link at all: nothing to open.
  assert.equal(linkOpenTarget(null, "app"), null);
});
