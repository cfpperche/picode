import test from "node:test";
import assert from "node:assert/strict";
import { setAgentTouched, readAgentTouched } from "./agentTouched.js";

test("agentTouched records the session's paths per agent and stays null when unknown", () => {
  assert.equal(readAgentTouched("ag1"), null);
  setAgentTouched("ag1", ["a.js", "src/b.js"]);
  assert.deepEqual(readAgentTouched("ag1"), ["a.js", "src/b.js"]);
  setAgentTouched("ag1", ["a.js"]);
  assert.deepEqual(readAgentTouched("ag1"), ["a.js"]);
  // Identical writes do not notify and cost nothing; unknown ids are ignored.
  setAgentTouched("", ["x.js"]);
  assert.equal(readAgentTouched(""), null);
});
