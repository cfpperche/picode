import assert from "node:assert/strict";
import { test } from "node:test";
import { alertFromPi, extractPiError } from "./piError.js";

test("credits error is plain language", () => {
  const a = alertFromPi({
    type: "message_end",
    message: { stopReason: "error", errorMessage: "CreditsError Insufficient balance code 401 Unauthorized" },
  });
  assert.equal(a.level, "error");
  assert.match(a.text, /credit|authorized/i);
});

test("retry warn", () => {
  const a = alertFromPi({ type: "auto_retry_start", attempt: 1, maxAttempts: 3, errorMessage: "529 Overloaded" });
  assert.equal(a.level, "warn");
  assert.match(a.text, /Retrying/);
});

test("aborted", () => {
  const a = alertFromPi({ type: "message_end", message: { stopReason: "aborted" } });
  assert.equal(a.level, "warn");
  assert.equal(a.text, "Stopped.");
});

test("extract nested", () => {
  assert.match(extractPiError({ error: { message: "nope" } }), /nope/);
});
