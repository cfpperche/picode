import test from "node:test";
import assert from "node:assert/strict";
import { CLI_PROVIDERS, cliProvidersHash, cliProvidersReturnTo, supportsCliProviders } from "./cliProviders.js";
import { cliLocation } from "./cliLaunch.js";
import { normalizeTerminalCli, terminalCliLabel } from "./terminalCli.js";

test("provider list/new URLs are the CLI's pane; the Pi-era aliases are retired", () => {
  // #/providers, #/more/providers and #/clis/providers[/<cli>] stopped naming
  // Pi's pane on 2026-09-25.
  for (const hash of ["#/providers", "#/providers/new", "#/more/providers", "#/clis/providers", "#/clis/providers/pi"]) {
    assert.notEqual(cliLocation(hash).pane, "providers", hash);
  }
  for (const add of [false, true]) {
    const route = cliLocation(cliProvidersHash("pi", { add }));
    assert.equal(route.id, "pi"); assert.equal(route.pane, "providers");
    assert.equal(!!route.add, add);
    assert.equal(route.redirect, undefined); assert.equal(route.invalid, undefined);
  }
  assert.equal(cliProvidersHash("pi", { custom: true }), "#/clis/pi/providers/custom");
  assert.equal(cliProvidersHash("pi", { custom: true, customId: "a b" }), "#/clis/pi/providers/custom/a%20b");
  assert.deepEqual(cliLocation(cliProvidersHash("pi", { custom: true })), { view: "clis", id: "pi", pane: "providers", custom: "new" });
  assert.deepEqual(cliLocation(cliProvidersHash("pi", { custom: true, customId: "cheap" })), { view: "clis", id: "pi", pane: "providers", custom: "edit", customId: "cheap" });
});

test("the one providers pane covers all nine CLIs, pi included", () => {
  assert.deepEqual(CLI_PROVIDERS.map((cli) => cli.id),
    ["pi", "claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy", "omp"]);
  for (const row of CLI_PROVIDERS) {
    // A launch-catalog id with terminalCli's own label: one spelling per CLI,
    // so the pane's heading and the terminal badge never disagree.
    assert.equal(normalizeTerminalCli(row.id), row.id, row.id);
    assert.equal(terminalCliLabel(row.id), row.name, row.id);
    assert.equal(supportsCliProviders(row.id), true, row.id);
  }
  for (const unknown of ["", "unknown", "claude", "%ZZ", null, undefined]) {
    assert.equal(supportsCliProviders(unknown), false, String(unknown));
  }
});

test("malformed provider paths are invalid, and explicit scope never selects the machine store", () => {
  for (const path of ["#/clis/pi/providers/extra", "#/clis/pi/providers/new/extra"]) {
    assert.equal(cliLocation(path).invalid, true, path);
  }
  for (const query of ["agentId=A", "workspaceId=W", "scope=agent", "agentId="]) {
    assert.equal(cliLocation("#/clis/pi/providers?" + query).scoped, true, query);
  }
});

test("OAuth return preserves the app and query while closing the add flow", () => {
  for (const path of ["/", "/desktop/", "/mobile/"]) {
    const url = cliProvidersReturnTo("https://picode.test:8445" + path + "?theme=light#/clis/pi/providers/new");
    assert.equal(url, "https://picode.test:8445" + path + "?theme=light#/clis/pi/providers");
  }
});
