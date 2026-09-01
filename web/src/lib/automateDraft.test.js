import test from "node:test";
import assert from "node:assert/strict";
import { isAutomateCommand, automatePrompt, parseAutomateReply } from "./automateDraft.js";
import { isValidCron } from "./cron.js";

test("isAutomateCommand recognises the command and its args", () => {
  assert.equal(isAutomateCommand("/automate every weekday at 9 summarize"), "every weekday at 9 summarize");
  assert.equal(isAutomateCommand("  /Automate  "), "");
  assert.equal(isAutomateCommand("/automate"), "");
  assert.equal(isAutomateCommand("/automated thing"), null);
  assert.equal(isAutomateCommand("automate this"), null);
  assert.equal(isAutomateCommand(""), null);
});

test("automatePrompt quotes the description and names the repo", () => {
  const p = automatePrompt("nightly tests\nwith a report", { workspaceName: "picode" });
  assert.match(p, /for the picode repository/);
  assert.match(p, /> nightly tests\n> with a report/);
  assert.match(p, /```json/);
  assert.match(p, /maxRunsWindowMin/);
});

test("parseAutomateReply reads the last json fence", () => {
  const reply = "Here is a first idea:\n```json\n{\"name\":\"Old\",\"prompt\":\"x\"}\n```\nActually better:\n```json\n{\n \"name\": \"Nightly tests\",\n \"prompt\": \"Run make test and report failures.\",\n \"cron\": \"0 2 * * *\",\n \"webhook\": false,\n \"maxCostUsd\": 1,\n \"maxRuns\": 2,\n \"maxRunsWindowMin\": 1440\n}\n```\nDone.";
  const d = parseAutomateReply(reply, isValidCron);
  assert.equal(d.name, "Nightly tests");
  assert.equal(d.cron, "0 2 * * *");
  assert.equal(d.maxRuns, 2);
});

test("parseAutomateReply falls back to a bare object and rejects junk", () => {
  const d = parseAutomateReply("Sure: {\"name\":\"Brief\",\"prompt\":\"Summarize {git log}\",\"cron\":\"0 9 * * 1-5\"} — ok?", isValidCron);
  assert.equal(d.name, "Brief");
  assert.equal(d.prompt, "Summarize {git log}");
  assert.equal(parseAutomateReply("I could not decide.", isValidCron), null);
  assert.equal(parseAutomateReply("```json\n{\"prompt\":\"no name\"}\n```", isValidCron), null);
  const bad = parseAutomateReply("```json\n{\"name\":\"X\",\"prompt\":\"p\",\"cron\":\"daily\"}\n```", isValidCron);
  assert.equal(bad.cron, "");
});
