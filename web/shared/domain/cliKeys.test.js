import assert from "node:assert/strict";
import { test } from "node:test";
import { KEYBOARD_CLIS, blockNote, keyboardRow, noteIsExternal, pickupLine } from "./cliKeys.js";

// A shipped row carries the editor the pane mounts, and no other row does: a
// row that ships without one is a pane that renders nothing.
test("a shipped row names its editor, and nothing else does", () => {
  for (const cli of KEYBOARD_CLIS) {
    if (cli.state === "shipped") {
      assert.ok(["pi-settings", "keymap"].includes(cli.editor), cli.id + " ships without an editor");
    } else {
      assert.equal(cli.editor, undefined, cli.id + " is not shipped and names an editor");
    }
  }
});

test("every registry CLI answers, in sidebar order", () => {
  assert.equal(KEYBOARD_CLIS.length, 9);
  assert.equal(KEYBOARD_CLIS[0].id, "pi");
  assert.equal(KEYBOARD_CLIS.at(-1).id, "omp");
  for (const cli of KEYBOARD_CLIS) {
    assert.equal(keyboardRow(cli.id), cli);
    assert.ok(cli.label.length > 1, cli.id + " has no label");
  }
  assert.equal(keyboardRow("nonesuch"), undefined);
  assert.equal(pickupLine("nonesuch"), "");
});

// The pickup sentence is a claim about someone else's software: a vendor that
// documents nothing says so, and the shipped CLI names its own reload.
test("the pickup line matches what the CLI actually does", () => {
  assert.equal(pickupLine("pi"), "Pi applies a change on /reload or the next run.");
  assert.match(pickupLine("claude-code"), /as you save it/);
  assert.match(pickupLine("codex"), /restart it/);
  // Omp's pickup was measured from its own bundle: the manager reads the files
  // when it is created and nothing re-reads them, so the sentence a user can act
  // on is "restart it".
  assert.match(pickupLine("omp"), /restart it/);
  for (const id of ["hermes", "opencode", "agy"]) {
    assert.match(pickupLine(id), /is not documented/);
  }
});

// A CLI without an editor says why: a vendor that refuses remapping is a
// different fact from an adapter PiCode has not written, and both come with one
// action (chrome carries state and the next action).
test("a pane with no editor carries one line and one action", () => {
  const grok = blockNote("grok");
  assert.match(grok.line, /built in/);
  assert.match(grok.line, /Ctrl\+\./);
  assert.equal(grok.action, "Open the key list");
  assert.match(grok.href, /^https:\/\//);

  const muse = blockNote("muse");
  assert.match(muse.line, /does not allow remapping/);
  assert.match(muse.line, /\/keymap/);

  const codex = blockNote("codex");
  assert.match(codex.line, /has not shipped yet/);
  assert.equal(codex.action, "Open the documentation");
  assert.match(codex.href, /^https:\/\//);

  // A CLI with no key map file at all is not waiting for an adapter: its three
  // keys are rows PiCode already edits, so the action is navigation, not a
  // vendor's page.
  const hermes = blockNote("hermes");
  assert.match(hermes.line, /a few keys/);
  assert.match(hermes.line, /built in/);
  assert.equal(hermes.action, "Open Settings");
  assert.equal(hermes.href, "#/clis/hermes/settings");
  assert.equal(noteIsExternal(hermes), false, "an in-app route is not a new tab");
  assert.equal(noteIsExternal(codex), true);

  assert.equal(blockNote("pi"), null, "a shipped CLI has no blocked note");
  assert.equal(blockNote("nonesuch"), null);
});

// Every blocked row must have somewhere to send the reader, or the action is a
// dead end.
test("every row that cannot be edited links somewhere", () => {
  for (const cli of KEYBOARD_CLIS) {
    if (cli.state === "shipped") continue;
    const note = blockNote(cli.id);
    assert.ok(note, cli.id + " has no note");
    assert.match(note.href, /^(https:\/\/|#\/)/, cli.id + " has no link");
  }
});
