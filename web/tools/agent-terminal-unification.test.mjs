import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync, existsSync } from "node:fs";

const read = path => readFileSync(new URL("../" + path, import.meta.url), "utf8");
test("mobile agents delegate the entire TUI to the Agent CLIs screen", () => {
  const agent = read("mobile/src/screens/Agent.jsx");
  assert.match(agent, /<TerminalScreen/);
  assert.doesNotMatch(agent, /<TermSurface|<ShellTerm|<TerminalDock|useTermAccessory|new Terminal/);
});
for (const app of ["mobile", "browser"]) {
  test(app + " uses the common runtime and contains no legacy renderer", () => {
    const shell = read(app + "/src/components/ShellTerm.jsx");
    assert.match(shell, /wireTerminalRuntime\(entry/);
    assert.doesNotMatch(shell, /connectTermSocket\(|wireTermWheel\(|term\.onData\(/);
    assert.match(shell, /cached\.session !== session\) closeTerm\(id\)/);
    assert.equal(existsSync(new URL("../" + app + "/src/components/TerminalDock.jsx", import.meta.url)), false);
  });
}
test("touch handling belongs to the terminal engine, never to its route", () => {
  assert.match(read("shared/client/terminalRuntime.js"), /wireTermTouch\(paneEl\)/);
  assert.doesNotMatch(read("mobile/src/screens/Terminal.jsx"), /touchmove|touchstart/);
});
test("browser explicit TUI navigation clears Chat and visible hosts reclaim links", () => {
  const app = read("browser/src/App.jsx");
  const open = app.slice(app.indexOf("async function openInteractive"), app.indexOf("async function stopAgent"));
  assert.match(open, /opts\.dock !== false[\s\S]*setChatWanted[\s\S]*next\.delete\(loc\.agent\.id\)/);
  const shell = read("browser/src/components/ShellTerm.jsx");
  assert.match(shell, /if \(!active \|\| !agentId\) return;[\s\S]*bindLinksRef\.current\?\.\(entry\)/);
  assert.match(shell, /else window\.open\(href, "_blank", "noopener,noreferrer"\)/);
});
