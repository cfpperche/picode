import assert from "node:assert/strict";
import { test } from "node:test";
import {
  RESERVED_CHORDS,
  isReservedChord,
  firstReservedChord,
  reservedBrowserAction,
} from "./browserChord.js";

test("the reserved set is exactly the browser-consumed chords on Chromium", () => {
  // Spot-check the shape: modifiers first, app chord format, one row per
  // chord so a duplicate can never slip in silently.
  const chords = RESERVED_CHORDS.map((r) => r.chord);
  assert.equal(new Set(chords).size, chords.length, "duplicate reserved chord");
  for (const row of RESERVED_CHORDS) {
    assert.match(row.chord, /^(ctrl|super)\+/, "reserved chords carry ctrl or super");
    assert.ok(row.browser, "every row names what the browser does with it");
  }
  for (const chord of ["ctrl+t", "ctrl+w", "ctrl+9", "ctrl+pageUp", "ctrl+shift+t"]
      .concat(["super+t", "super+w", "super+shift+n"])) {
    assert.ok(isReservedChord(chord), chord + " is reserved");
  }
  assert.equal(reservedBrowserAction("ctrl+t"), "new tab");
  assert.equal(reservedBrowserAction("super+w"), "close tab (macOS)");
  assert.equal(reservedBrowserAction("ctrl+q"), null);
});

test("plain, shift-only and alt chords are not reserved — the page can own them", () => {
  // The app's own chords and the terminal's family live here: if any of
  // these turned reserved, the chord would stop reaching the page at all
  // and nothing in the app could hand it back.
  for (const chord of ["t", "shift+enter", "alt+[", "ctrl+k", "ctrl+shift+enter", "super+k"]) {
    assert.equal(isReservedChord(chord), false, chord + " is not reserved");
  }
});

test("firstReservedChord returns the offender so a bad default fails loudly", () => {
  assert.equal(firstReservedChord(["ctrl+k", "super+k"]), null);
  assert.equal(firstReservedChord(["alt+[", "ctrl+w"]), "ctrl+w");
  assert.equal(firstReservedChord(undefined), null);
  assert.equal(firstReservedChord([]), null);
});
