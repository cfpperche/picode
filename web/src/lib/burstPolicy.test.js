import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const read = (path) => readFileSync(new URL(path, import.meta.url), "utf8");

test("TUI inbox replies have a same-tab burst route and no chat route", () => {
  const app = read("../desktop/App.jsx");
  const surface = read("../components/AppSurface.jsx");
  assert.match(app, /agentburst:/);
  assert.match(app, /<BurstSurface/);
  assert.match(app, /setTermWanted\(\(s\) => new Set\(s\)\.add\(id\)\)/);
  assert.doesNotMatch(app, /agentchat:/);
  assert.match(app, /agent && agent\.burst/);
  assert.match(app, /terminalUnavailable\) await openInteractive\(agent\.id, \{ restart: true \}\)/);
  assert.match(app, /forceRestart \? "\?restart=1"/);
  assert.match(app, /forceRestart\) closeShellTerm\(loc\.agent\.id\)/);
  assert.match(app, /termEpochs\[agent\.id\]/);
  const rows = read("../components/WorkspaceRows.jsx");
  assert.match(rows, /!ag\.burst \? <button[^>]+title="Chat"/);
  assert.match(surface, /_burst/);
  assert.match(surface, /action\.id === "accept" \|\| action\.id === "decline"/);
  assert.doesNotMatch(surface, /_switch/);
  const mobile = read("../mobile/screens/Agent.jsx");
  assert.match(mobile, /burst \? \(/);
  assert.match(mobile, /<BurstSurface/);
  assert.match(mobile, /interactive && !burst/);
  assert.match(mobile, /terminalUnavailable[\s\S]*closeShellTerm\(id\)[\s\S]*\/open\?restart=1/);
  assert.match(mobile, /terminalEpoch/);
  const mobileApp = read("../mobile/App.jsx");
  assert.match(mobileApp, /api\/agents\/" \+ agent\.id \+ "\/close/);
  assert.doesNotMatch(mobileApp, /api\/workspaces\/" \+ workspace\.id \+ "\/close/);
  const mobileState = read("../mobile/components/StateChip.jsx");
  assert.match(mobileState, /a\.burst.*"working"/);
});

test("the transient surface is status-only, not a second chat surface", () => {
  const burst = read("../components/BurstSurface.jsx");
  assert.match(burst, /Receiving/);
  assert.match(burst, /Processing/);
  assert.match(burst, /Returning/);
  assert.doesNotMatch(burst, /ChatSurface|Composer|model picker|textarea/i);
  const css = read("../styles/app.css");
  assert.match(css, /burst-done[\s\S]*animation: burst-exit/);
  assert.match(css, /prefers-reduced-motion[\s\S]*burst-done/);
});
