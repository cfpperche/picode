import { test } from "node:test";
import assert from "node:assert/strict";
import { INHERIT, selectedKey, choicesFor, effectText, isOverridden, withChoice, inheritedValueFor, matchesQuery, groupCatalog } from "./termSettings.js";

const MOUSE = { key: "mouse", label: "Mouse", values: ["on", "off"], effect: "live" };

test("a field absent from this scope reads as inherited", () => {
  assert.equal(selectedKey({}, "mouse"), INHERIT);
  assert.equal(selectedKey(undefined, "mouse"), INHERIT);
});

// The trap: a stored value that matches what it would inherit is still an
// override, and has to keep showing as one — otherwise the panel would move
// the selection on its own when the global changes underneath it.
test("a stored value equal to the inherited one is still an override", () => {
  assert.equal(selectedKey({ mouse: "on" }, "mouse"), "on");
  assert.equal(isOverridden({ mouse: "on" }, "mouse"), true);
  assert.equal(isOverridden({}, "mouse"), false);
});

test("the inherit segment shows the value it falls back to", () => {
  const [head] = choicesFor(MOUSE, { mouse: "off" }, false);
  assert.equal(head.key, INHERIT);
  assert.equal(head.label, "Inherit (Off)");
});

test("the global panel calls it the default, not inherit", () => {
  const [head] = choicesFor(MOUSE, { mouse: "on" }, true);
  assert.equal(head.label, "Default (On)");
});

test("choices carry every value the flag takes, in order", () => {
  const keys = choicesFor(MOUSE, { mouse: "on" }, false).map((c) => c.key);
  assert.deepEqual(keys, [INHERIT, "on", "off"]);
});

test("a flag with no inherited value still renders a head segment", () => {
  const [head] = choicesFor(MOUSE, {}, false);
  assert.equal(head.label, "Inherit");
});

test("every effect the registry can send has words for it", () => {
  assert.match(effectText("live"), /right away/);
  assert.match(effectText("new-panes"), /from now on/);
  assert.equal(effectText("something-new"), "");
});

test("choosing a value stores it", () => {
  assert.deepEqual(withChoice({}, "mouse", "off"), { mouse: "off" });
});

// The same rule the server enforces, enforced here too: the optimistic view
// has to agree with what the server will store, or the selection jumps when
// the response lands.
test("choosing inherit removes the key rather than storing the inherited value", () => {
  const after = withChoice({ mouse: "off" }, "mouse", INHERIT);
  assert.deepEqual(after, {});
  assert.equal(selectedKey(after, "mouse"), INHERIT);
});

test("withChoice does not mutate its input", () => {
  const before = { mouse: "off" };
  withChoice(before, "mouse", "on");
  assert.deepEqual(before, { mouse: "off" });
});

test("inheritedValueFor prefers the settings layer, then the catalog", () => {
  assert.equal(inheritedValueFor("mouse", { mouse: "off" }, "on"), "off");
  assert.equal(inheritedValueFor("mode-keys", {}, "emacs"), "emacs");
  assert.equal(inheritedValueFor("mode-keys", undefined, undefined), "");
});

test("matchesQuery finds by name and by warning text", () => {
  const row = { name: "set-clipboard", danger: "Governs how programs reach your clipboard" };
  assert.equal(matchesQuery(row, "clip"), true);
  assert.equal(matchesQuery(row, "CLIPBOARD"), true);
  assert.equal(matchesQuery(row, "mouse"), false);
  assert.equal(matchesQuery(row, ""), true);
});

test("groupCatalog keeps server options off a terminal's page", () => {
  const rows = [
    { name: "mouse", scope: "session", curated: true },
    { name: "mode-keys", scope: "window" },
    { name: "escape-time", scope: "server" },
  ];
  const term = groupCatalog(rows, { isGlobal: false });
  assert.deepEqual(term.perTerminal.map((r) => r.name), ["mode-keys"]);
  assert.deepEqual(term.server, []);
  const glob = groupCatalog(rows, { isGlobal: true });
  assert.deepEqual(glob.server.map((r) => r.name), ["escape-time"]);
});
