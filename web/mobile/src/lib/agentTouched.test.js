import test from "node:test";
import assert from "node:assert/strict";
import { setAgentTouched, readAgentTouched } from "./agentTouched.js";

test("agentTouched records the session's paths per agent and stays null when unknown", () => {
  assert.equal(readAgentTouched("ag1"), null);
  setAgentTouched("ag1", ["a.js", "src/b.js"]);
  assert.deepEqual(readAgentTouched("ag1"), ["a.js", "src/b.js"]);
  // Reads return the stored array by identity until a real change —
  // useSyncExternalStore loops if getSnapshot fabricates a fresh one.
  const snapshot = readAgentTouched("ag1");
  setAgentTouched("ag1", ["a.js", "src/b.js"]);
  assert.equal(readAgentTouched("ag1"), snapshot);
  setAgentTouched("ag1", ["a.js"]);
  assert.deepEqual(readAgentTouched("ag1"), ["a.js"]);
  assert.notEqual(readAgentTouched("ag1"), snapshot);
  // Identical writes do not notify and cost nothing; unknown ids are ignored.
  setAgentTouched("", ["x.js"]);
  assert.equal(readAgentTouched(""), null);
});
