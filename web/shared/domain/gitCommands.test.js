import test from "node:test";
import assert from "node:assert/strict";
import { askedNote } from "./gitCommands.js";

// The git graph composes thirty-odd actions the rail's own verb table never
// knew (ADR-0096). When the server named one, that name is what the toast
// says; the table stays the fallback for the six it does know.
test("a composed verb wins over the built-in table", () => {
  assert.equal(askedNote("Atlas", "merge", {}, "merge feat/x"), "Asked Atlas to merge feat/x.");
  assert.equal(askedNote("Atlas", "commit", {}), "Asked Atlas to commit.");
  assert.equal(askedNote("Atlas", "reset-hard", {}), "Asked Atlas to do it.", "an unknown action still reads as a sentence");
  assert.equal(askedNote("Atlas", "reset-hard", {}, "reset to 0123456, discarding the changes"),
    "Asked Atlas to reset to 0123456, discarding the changes.");
});

test("the channel the prompt took still decides the tense", () => {
  assert.equal(askedNote("Atlas", "merge", { mode: "interactive" }, "merge feat/x"), "Asked Atlas to merge feat/x in its terminal.");
  assert.equal(askedNote("Atlas", "merge", { busy: true }, "merge feat/x"), "Queued for Atlas: merge feat/x after its current turn.");
});
