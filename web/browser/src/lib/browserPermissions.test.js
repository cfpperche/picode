import { test } from "node:test";
import assert from "node:assert/strict";
import { ALL_SITES, askHost, askTitle, permissionPush } from "./browserPermissions.js";

// The load-time push is the half that can quietly change policy: a reported
// decision must never be handed back as a standing, or a one-off Allow
// becomes permanent.

test("permissionPush: the every-site rows are always policy", () => {
  assert.deepEqual(permissionPush({ origin: ALL_SITES, kind: "camera", decision: "ask" }), {
    kind: "camera",
    state: "ask",
    origin: null,
  });
  assert.deepEqual(permissionPush({ origin: ALL_SITES, kind: "camera", decision: "allow", standing: false }), {
    kind: "camera",
    state: "allow",
    origin: null,
  });
});

test("permissionPush: a site row is pushed only when it is a standing", () => {
  assert.equal(permissionPush({ origin: "https://meet.example.com/room", kind: "camera", decision: "allow", standing: false }), null);
  assert.deepEqual(permissionPush({ origin: "https://meet.example.com/room", kind: "camera", decision: "allow", standing: true }), {
    kind: "camera",
    state: "allow",
    origin: "https://meet.example.com/room",
  });
});

test("permissionPush: a row missing its words is skipped, not guessed", () => {
  assert.equal(permissionPush(null), null);
  assert.equal(permissionPush({ origin: ALL_SITES, decision: "allow" }), null);
  assert.equal(permissionPush({ origin: ALL_SITES, kind: "camera" }), null);
});

test("askHost reads the site out of the URI", () => {
  assert.equal(askHost("https://meet.example.com/room?x=1"), "meet.example.com");
  assert.equal(askHost("http://localhost:5173/"), "localhost:5173");
  assert.equal(askHost(""), "This site");
  assert.equal(askHost("not a url"), "not a url");
});

test("askTitle names the request in plain words", () => {
  assert.equal(askTitle({ origin: "https://meet.example.com/room", kind: "camera" }), "meet.example.com wants to use your camera");
  assert.equal(askTitle({ origin: "https://meet.example.com", kind: "microphone" }), "meet.example.com wants to use your microphone");
  assert.equal(askTitle({ origin: "https://meet.example.com", kind: "telepathy" }), "meet.example.com wants to use a permission");
});
