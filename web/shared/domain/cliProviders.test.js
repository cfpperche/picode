import test from "node:test";
import assert from "node:assert/strict";
import { CLI_PROVIDERS, cliProvidersHash, cliProvidersLocation, cliProvidersReturnTo, supportsCliProviders } from "./cliProviders.js";
import { cliLocation } from "./cliLaunch.js";

test("provider list/new URLs normalize legacy links without adding agent scope", () => {
  for (const prefix of ["#/providers", "#/more/providers"]) {
    for (const add of [false, true]) {
      const route = cliProvidersLocation(prefix + (add ? "/new" : ""));
      assert.equal(route.redirect, cliProvidersHash("pi", { add }));
      assert.equal(route.add, add);
      assert.equal(route.scoped, false);
      assert.equal(cliLocation(prefix).view, "providers");
    }
  }
  assert.equal(cliProvidersLocation("#/clis/providers").redirect, "#/clis/providers/pi");
  for (const add of [false, true]) {
    const route = cliLocation(cliProvidersHash("pi", { add }));
    assert.equal(route.id, "pi"); assert.equal(route.add, add);
    assert.equal(route.redirect, ""); assert.equal(route.invalid, false);
  }
});

test("provider support does not follow launch support or malformed paths", () => {
  assert.deepEqual(CLI_PROVIDERS.map(cli => cli.id), ["pi"]);
  assert.equal(supportsCliProviders("pi"), true);
  for (const cli of ["codex", "claude", "unknown", "%ZZ", "", "pi%2Fextra"]) {
    const route = cliProvidersLocation("#/clis/providers/" + cli);
    assert.equal(supportsCliProviders(route.id), false, cli);
    assert.equal(route.redirect, "");
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
    const url = cliProvidersReturnTo("https://picode.test:8445" + path + "?theme=light#/clis/providers/pi/new");
    assert.equal(url, "https://picode.test:8445" + path + "?theme=light#/clis/providers/pi");
  }
});
