import test from "node:test";
import assert from "node:assert/strict";
import { quickSettingsFor, readQuickValue, applyQuickValue, argLine } from "./cliLaunchPresets.js";
import { launchArgs } from "./cliLaunch.js";

const MODEL = { key: "model", type: "text", flags: ["--model"] };
const CODEX_MODEL = { key: "model", type: "text", flags: ["--model", "-m"] };
const THINKING = { key: "thinking", type: "select", flags: ["--thinking"] };
const REASONING = { key: "reasoning", type: "select", flags: ["-c"], kv: "model_reasoning_effort" };
const AUTO = { key: "auto", type: "boolean", flags: ["--auto"] };

test("quick settings exist only for CLIs with verified flags", () => {
  for (const id of ["pi", "claude-code", "codex", "opencode"]) assert.ok(quickSettingsFor(id), id);
  for (const id of ["grok", "hermes", "omp", "muse", "agy", "unknown"]) assert.equal(quickSettingsFor(id), null, id);
});

test("a value is appended when the flag is absent", () => {
  assert.deepEqual(applyQuickValue([], MODEL, "sonnet"), ["--model", "sonnet"]);
  assert.deepEqual(applyQuickValue(["--plan"], MODEL, "sonnet"), ["--plan", "--model", "sonnet"]);
});

test("an existing flag is replaced, not duplicated", () => {
  assert.deepEqual(applyQuickValue(["--model", "haiku"], MODEL, "sonnet"), ["--model", "sonnet"]);
  assert.deepEqual(applyQuickValue(["--thinking", "low", "--model", "haiku"], MODEL, "sonnet"), ["--thinking", "low", "--model", "sonnet"]);
});

test("an empty value removes the flag", () => {
  assert.deepEqual(applyQuickValue(["--model", "haiku"], MODEL, ""), []);
  assert.deepEqual(applyQuickValue(["--thinking", "low", "--model", "haiku"], MODEL, ""), ["--thinking", "low"]);
});

test("a joined form is replaced by the bare pair", () => {
  assert.deepEqual(applyQuickValue(["--model=haiku"], MODEL, "sonnet"), ["--model", "sonnet"]);
  assert.deepEqual(applyQuickValue(["--model=haiku"], MODEL, ""), []);
  assert.equal(readQuickValue(["--model=haiku"], MODEL), "haiku");
});

test("aliases are replaced too", () => {
  assert.deepEqual(applyQuickValue(["-m", "gpt-5.3-codex"], CODEX_MODEL, "gpt-5.4"), ["--model", "gpt-5.4"]);
  assert.deepEqual(applyQuickValue(["-m", "gpt-5.3-codex", "--fast"], CODEX_MODEL, ""), ["--fast"]);
});

test("unknown and user-typed arguments survive untouched", () => {
  const args = ["--add-dir", "/tmp/x", "--model", "haiku", "--dangerously-skip-permissions", "", "--trailing"];
  assert.deepEqual(applyQuickValue(args, MODEL, "sonnet"), ["--add-dir", "/tmp/x", "--model", "sonnet", "--dangerously-skip-permissions", "", "--trailing"]);
});

test("a malformed flag at the end is dropped cleanly", () => {
  assert.deepEqual(applyQuickValue(["--model"], MODEL, "sonnet"), ["--model", "sonnet"]);
  assert.deepEqual(applyQuickValue(["--model"], MODEL, ""), []);
  assert.equal(readQuickValue(["--model"], MODEL), "");
});

test("a flag-shaped argument is not eaten as a value", () => {
  assert.deepEqual(applyQuickValue(["--model", "--plan"], MODEL, ""), ["--plan"]);
  assert.equal(readQuickValue(["--model", "--plan"], MODEL), "");
});

test("kv pairs replace only their own key and keep other -c overrides", () => {
  const args = ["-c", "model_reasoning_effort=low", "-c", "web_search=live"];
  assert.deepEqual(applyQuickValue(args, REASONING, "high"), ["-c", "model_reasoning_effort=high", "-c", "web_search=live"]);
  assert.deepEqual(applyQuickValue(args, REASONING, ""), ["-c", "web_search=live"]);
  assert.equal(readQuickValue(args, REASONING), "low");
  assert.equal(readQuickValue(["-c", "web_search=live"], REASONING), "");
});

test("boolean flags turn on and off", () => {
  assert.deepEqual(applyQuickValue([], AUTO, "on"), ["--auto"]);
  assert.deepEqual(applyQuickValue(["--auto"], AUTO, ""), []);
  assert.equal(readQuickValue(["--auto"], AUTO), "on");
  assert.equal(readQuickValue([], AUTO), "");
});

test("readQuickValue returns a value the select does not list, so the UI can show it", () => {
  assert.equal(readQuickValue(["--thinking", "ultrathink"], THINKING), "ultrathink");
});

test("argLine quotes only what the textarea format needs, and the round trip holds", () => {
  assert.equal(argLine("sonnet"), "sonnet");
  assert.equal(argLine(""), '""');
  assert.equal(argLine("model with space"), '"model with space"');
  assert.equal(argLine('"quoted"'), '"\\"quoted\\""');
  for (const a of ["sonnet", "", "model with space", '"quoted"', "--flag"]) {
    assert.deepEqual(launchArgs(argLine(a)), [a]);
  }
});

test("the full draft round trip: quick patch survives launchArgs parsing", () => {
  const args = applyQuickValue(launchArgs('"--flag with space"\n--model\nhaiku'), MODEL, "sonnet:high");
  assert.deepEqual(args, ["--flag with space", "--model", "sonnet:high"]);
});
