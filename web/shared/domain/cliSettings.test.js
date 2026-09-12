import test from "node:test";
import assert from "node:assert/strict";
import { cliSettingsLocation, cliSettingsHash, supportsCliSettings, loadPiSettingsContext } from "./cliSettings.js";
import { cliLocation, cliPaneHash } from "./cliLaunch.js";

test("native settings routes preserve identity, legacy context and explicit global scope", () => {
  for (const old of ["#/settings", "#/more/settings"]) {
    assert.equal(cliSettingsLocation(old, "agent / A").redirect, "#/clis/pi/settings?agentId=agent+%2F+A");
    assert.equal(cliSettingsLocation(old + "?agentId=B", "A").agentId, "B");
    assert.equal(cliSettingsLocation(old + "?agentId=", "A").agentId, "");
  }
  const hash = cliSettingsHash("pi", { agentId: "agent / A", focus: "scoped-models" });
  assert.equal(hash, "#/clis/pi/settings?agentId=agent+%2F+A&focus=scoped-models");
  assert.equal(cliLocation(hash).agentId, "agent / A");
  assert.equal(cliLocation(hash).focus, "scoped-models");
  assert.equal(cliLocation(hash).pane, "settings");
  assert.equal(cliSettingsLocation("#/clis/settings", "A").redirect, "#/clis/pi/settings");
  assert.equal(cliSettingsLocation("#/clis/settings/pi", "A").redirect, "#/clis/pi/settings");
  assert.equal(cliSettingsLocation("#/clis/pi/settings", "A").agentId, "");
  assert.equal(cliSettingsLocation("#/clis/pi/settings", "A").redirect, "");
  assert.equal(cliSettingsLocation("#/clis/pi/settings/extra").invalid, true);
  assert.equal(cliSettingsLocation("#/preferences"), null);
});

test("the edited layer survives a reload, and guesses are dropped", () => {
  assert.equal(cliSettingsHash("pi", { agentId: "A", layer: "project" }), "#/clis/pi/settings?agentId=A&layer=project");
  assert.equal(cliSettingsHash("pi", { layer: "root" }), "#/clis/pi/settings");
  const round = cliSettingsLocation("#/clis/pi/settings?agentId=A&layer=agent");
  assert.equal(round.layer, "agent");
  assert.equal(round.view, "clis");
  assert.equal(cliSettingsLocation("#/clis/pi/settings?layer=user").layer, "");
  // A legacy link keeps the layer it was opened with.
  assert.equal(cliSettingsLocation("#/settings?layer=project", "A").redirect, "#/clis/pi/settings?agentId=A&layer=project");
});

test("the keyboard map is a pane of its own, and its sub-tab links still land", () => {
  // `#/clis/pi/keyboard` is an ordinary pane; the settings context rides along
  // so a round trip lands back on the agent and the layer it left.
  const pane = cliLocation("#/clis/pi/keyboard");
  assert.equal(pane.pane, "keyboard");
  assert.equal(pane.id, "pi");
  assert.equal(cliPaneHash("pi", "keyboard"), "#/clis/pi/keyboard");
  const scoped = cliLocation("#/clis/pi/keyboard?agentId=A&layer=project");
  assert.equal(scoped.agentId, "A");
  assert.equal(scoped.layer, "project");
  assert.equal(cliLocation("#/clis/pi/keyboard?layer=user").layer, "", "a guessed layer is dropped");
  assert.equal(cliLocation("#/clis/pi/keyboard/extra").invalid, true, "no path after the pane");
  // The sub-tab that lived inside Settings until 2026-09-12 redirects there,
  // keeping whatever else the link carried.
  const old = cliLocation("#/clis/pi/settings?agentId=A&layer=agent&tab=keys");
  assert.equal(old.redirect, "#/clis/pi/keyboard");
  assert.equal(old.keysTab, false);
  const legacy = cliLocation("#/settings?tab=keys", { agentId: "A" });
  assert.equal(legacy.redirect, "#/clis/pi/keyboard");
  assert.equal(cliLocation("#/clis/pi/settings?tab=other").redirect, "");
});

test("unsupported and malformed CLI identities never become Pi", () => {
  assert.equal(supportsCliSettings("pi"), true);
  for (const id of ["codex", "unknown", "%ZZ", "pi/extra", ""]) {
    const route = cliSettingsLocation("#/clis/" + id + "/settings") || cliSettingsLocation("#/clis/settings/" + id);
    assert.equal(supportsCliSettings(route.id), false, id);
  }
  assert.equal(cliLocation("#/clis/sessions?cli=pi").pane, "sessions");
  assert.equal(cliLocation("#/clis/pi").view, "clis");
  assert.equal(cliLocation("#/clis/pi/sessions").pane, "sessions");
});

test("explicit agent context comes from Pi's validated report, including free agents", async () => {
  for (const workspaceId of ["ws_free", "project"]) {
    const calls = [];
    const agent = { id: "A", workspaceId };
    const view = { ...agent, mode: "interactive" };
    const context = await loadPiSettingsContext("A", async path => {
      calls.push(path);
      if (path.startsWith("/api/pi-settings")) return { agent };
      if (path === "/api/agents?free=1") return [view];
      return [{ id: "project", name: "Project", agents: [view] }];
    });
    assert.equal(context.agent, view);
    assert.equal(context.agent.mode, "interactive");
    assert.equal(context.workspace?.id || null, workspaceId === "ws_free" ? null : "project");
    assert.equal(calls.length, 2);
  }
});

test("missing identities and failed reads never choose another agent or workspace", async () => {
  await assert.rejects(loadPiSettingsContext("A", async () => ({ agent: { id: "B" } })), /no longer available/);
  await assert.rejects(loadPiSettingsContext("A", async path => path.startsWith("/api/pi-settings") ? { agent: { id: "A", workspaceId: "gone" } } : [{ id: "other" }]), /workspace is no longer/);
  await assert.rejects(loadPiSettingsContext("A", async () => { throw new Error("offline"); }), /offline/);
});

test("missing context is distinguishable from a recoverable refresh failure", async () => {
  for (const missing of ["http", "agent", "workspace", "fleet"]) {
    await assert.rejects(loadPiSettingsContext("A", async path => {
      if (missing === "http") throw Object.assign(new Error("Not found"), { status: 404 });
      if (path.startsWith("/api/pi-settings")) return { agent: missing === "agent" ? null : { id: "A", workspaceId: "W" } };
      return missing === "workspace" ? [] : [{ id: "W", agents: [] }];
    }), { status: 404 });
  }
  const temporary = Object.assign(new Error("Temporarily unavailable"), { status: 503 });
  await assert.rejects(loadPiSettingsContext("A", async () => { throw temporary; }), error => error === temporary);
});
