import test from "node:test";
import assert from "node:assert/strict";
import { CLI_PROVIDERS, cliProvidersHash, cliProvidersLocation, cliProvidersReturnTo, supportsCliProviders } from "./cliProviders.js";
import { cliLocation } from "./cliLaunch.js";
import { normalizeTerminalCli, terminalCliLabel } from "./terminalCli.js";

test("provider list/new URLs normalize legacy links without adding agent scope", () => {
  for (const prefix of ["#/providers", "#/more/providers"]) {
    for (const add of [false, true]) {
      const route = cliProvidersLocation(prefix + (add ? "/new" : ""));
      assert.equal(route.redirect, cliProvidersHash("pi", { add }));
      assert.equal(route.add, add);
      assert.equal(route.scoped, false);
      assert.equal(cliLocation(prefix).pane, "providers");
    }
  }
  assert.equal(cliProvidersLocation("#/clis/providers").redirect, "#/clis/pi/providers");
  assert.equal(cliProvidersLocation("#/clis/providers/pi").redirect, "#/clis/pi/providers");
  assert.equal(cliProvidersLocation("#/clis/providers/pi/new").redirect, "#/clis/pi/providers/new");
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

test("provider support does not follow launch support or malformed paths", () => {
  // codex is a declared provider pane now, so the sweep uses ids that are not
  // (the malformed ones keep their own assertion below).
  for (const cli of ["unknown", "%ZZ", "", "pi%2Fextra"]) {
    const route = cliProvidersLocation("#/clis/providers/" + cli);
    assert.equal(supportsCliProviders(route.id), false, cli);
    if (cli === "unknown") {
      assert.equal(route.redirect, cliProvidersHash(cli));
    }
  }
  for (const path of ["#/clis/providers/pi/extra", "#/clis/providers/pi/new/extra", "#/providers/unknown", "#/more/providers/new/extra"]) {
    const route = cliProvidersLocation(path);
    assert.equal(route.invalid, true, path); assert.equal(route.redirect, "");
  }
});

test("explicit scope never silently selects the machine credential store", () => {
  for (const prefix of ["#/providers", "#/more/providers/new", "#/clis/providers/pi"]) {
    for (const query of ["agentId=A", "workspaceId=W", "scope=agent", "agentId="]) {
      const route = cliProvidersLocation(prefix + "?" + query);
      assert.equal(route.scoped, true); assert.equal(route.redirect, "");
    }
  }
});

test("llama aliases and unrelated routes do not enter the provider editor", () => {
  for (const hash of ["#/providers/llama", "#/more/providers/llama?tab=models", "#/llama/models", "#/clis/settings/pi", "#/providers-other", "#/more"]) {
    assert.equal(cliProvidersLocation(hash), null, hash);
  }
});

test("OAuth return preserves the app and query while closing the add flow", () => {
  for (const path of ["/", "/desktop/", "/mobile/"]) {
    const url = cliProvidersReturnTo("https://picode.test:8445" + path + "?theme=light#/clis/pi/providers/new");
    assert.equal(url, "https://picode.test:8445" + path + "?theme=light#/clis/pi/providers");
  }
});
