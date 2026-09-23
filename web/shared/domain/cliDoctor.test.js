import assert from "node:assert/strict";
import test from "node:test";
import { DOCTOR_CLIS, supportsCliDoctor, sourceLabel, formatValue, filterEntries, sortFindings } from "./cliDoctor.js";

test("only omp has checks", () => {
  assert.deepEqual(DOCTOR_CLIS, ["omp"]);
  assert.equal(supportsCliDoctor("omp"), true);
  assert.equal(supportsCliDoctor("codex"), false);
});

test("a value no file sets is not called a default", () => {
  assert.equal(sourceLabel("project"), "This workspace");
  assert.equal(sourceLabel("user"), "Global");
  assert.equal(sourceLabel(""), "Not in either file");
});

test("a bound pane names the workspace the value comes from", () => {
  assert.equal(sourceLabel("project", "delivery"), "delivery");
  // A blank or padded name is no name; the generic word stays.
  assert.equal(sourceLabel("project", "  "), "This workspace");
  assert.equal(sourceLabel("user", "delivery"), "Global");
});

test("values print on one line", () => {
  assert.equal(formatValue("yolo"), "yolo");
  assert.equal(formatValue(""), '""');
  assert.equal(formatValue(null), "not set");
  assert.equal(formatValue(["groq"]), '["groq"]');
  assert.equal(formatValue({ default: "openai/gpt-5" }), '{"default":"openai/gpt-5"}');
  assert.equal(formatValue("x".repeat(100), 10), "xxxxxxxxx…");
});

test("the filter reads key and value, never a hidden description or a credential", () => {
  const entries = [
    { key: "tools.approvalMode", description: "Default approval behavior", source: "", value: "yolo" },
    { key: "disabledProviders", description: "", source: "project", value: ["groq"] },
    { key: "goal.statusInFooter", description: "show the token count", source: "", value: true },
    { key: "auth.broker.token", source: "", value: "••••••", redacted: true },
  ];
  assert.deepEqual(filterEntries(entries, { query: "APPROVAL" }).map((e) => e.key), ["tools.approvalMode"]);
  assert.deepEqual(filterEntries(entries, { query: "groq" }).map((e) => e.key), ["disabledProviders"]);
  assert.deepEqual(filterEntries(entries, { query: "token" }).map((e) => e.key), ["auth.broker.token"]);
  assert.deepEqual(filterEntries(entries, { onlySet: true }).map((e) => e.key), ["disabledProviders"]);
});

test("warnings come first", () => {
  const out = sortFindings([{ id: "a", severity: "info" }, { id: "b", severity: "warn" }, { id: "c", severity: "info" }]);
  assert.deepEqual(out.map((f) => f.id), ["b", "a", "c"]);
});
