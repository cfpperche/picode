import test from "node:test";
import assert from "node:assert/strict";
import { terminalLaunchAgent } from "./cliLaunch.js";

// Binding x ownership x mode: routes and selection never determine ownership.
test("launch ownership resolves workspace and free agents in every Pi mode", () => {
  for (const mode of ["stopped", "interactive", "managed"]) {
    for (const cli of ["pi", "codex", "claude-code"]) {
      const agent = { id: "owner", terminalId: "bound", cli, mode };
      for (const [workspaces, free] of [[[ { agents: [agent] } ], []], [[], [agent]]]) {
        assert.equal(terminalLaunchAgent("bound", workspaces, free), agent);
        assert.equal(terminalLaunchAgent("missing", workspaces, free), null);
        assert.equal(terminalLaunchAgent(undefined, workspaces, free), null);
      }
    }
  }
  assert.equal(terminalLaunchAgent("standalone"), null);
  assert.equal(terminalLaunchAgent("standalone", [{}], [{ id: "unbound" }]), null);
});
