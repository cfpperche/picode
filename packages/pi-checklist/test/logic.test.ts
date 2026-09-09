import assert from "node:assert/strict";
import { test } from "node:test";
import {
	buildPayload,
	contractPrompt,
	currentStep,
	decideGate,
	decideReminder,
	MAX_REMINDERS,
	normalizeItems,
	parseLevel,
	publishTarget,
	reconstruct,
	REFUSAL,
	renderLines,
	resolveServerUrl,
	shouldPublish,
	summarize,
} from "../src/logic.ts";

test("publishTarget: agent wins, terminal is the fallback, raw pi is silent", () => {
	assert.deepEqual(publishTarget({ PICODE_AGENT_ID: "ag-1", PICODE_TERM_ID: "term-2" }), { kind: "agent", id: "ag-1" });
	assert.deepEqual(publishTarget({ PICODE_TERM_ID: "agent-cli-f5dd51" }), { kind: "terminal", id: "agent-cli-f5dd51" });
	assert.equal(publishTarget({ PICODE_AGENT_ID: "ag 1" }), null); // malformed agent id is not a terminal either
	assert.equal(publishTarget({ PICODE_TERM_ID: "" }), null);
	assert.equal(publishTarget({}), null);
});

test("level: PICODE_CHECKLIST parses, anything else is changes", () => {
	assert.equal(parseLevel({}), "changes");
	assert.equal(parseLevel({ PICODE_CHECKLIST: "ALWAYS" }), "always");
	assert.equal(parseLevel({ PICODE_CHECKLIST: "never" }), "never");
	assert.equal(parseLevel({ PICODE_CHECKLIST: "sometimes" }), "changes");
});

test("gate: mutators are refused until planned; reads and never pass", () => {
	assert.deepEqual(decideGate({ level: "changes", toolName: "edit", planned: false }), { block: true, reason: REFUSAL });
	assert.deepEqual(decideGate({ level: "changes", toolName: "bash", planned: false }), { block: true, reason: REFUSAL });
	assert.deepEqual(decideGate({ level: "changes", toolName: "read", planned: false }), { block: false });
	assert.deepEqual(decideGate({ level: "changes", toolName: "some_unknown_tool", planned: false }), { block: false });
	assert.deepEqual(decideGate({ level: "changes", toolName: "edit", planned: true }), { block: false });
	assert.deepEqual(decideGate({ level: "never", toolName: "edit", planned: false }), { block: false });
	assert.deepEqual(decideGate({ level: "always", toolName: "write", planned: false }), { block: true, reason: REFUSAL });
});

test("reminder: only always, only unplanned, capped", () => {
	assert.equal(decideReminder({ level: "changes", planned: false, sent: 0 }), false);
	assert.equal(decideReminder({ level: "always", planned: true, sent: 0 }), false);
	assert.equal(decideReminder({ level: "always", planned: false, sent: 0 }), true);
	assert.equal(decideReminder({ level: "always", planned: false, sent: MAX_REMINDERS }), false);
});

test("contract: never is silent, always covers read-only answers", () => {
	assert.equal(contractPrompt("never"), "");
	assert.match(contractPrompt("changes"), /needs no checklist/);
	assert.match(contractPrompt("always"), /still gets a short checklist/);
	assert.match(contractPrompt("changes"), /`checklist` tool/);
});

test("items: normalized, defaulted, bounded", () => {
	const items = normalizeItems([{ text: "  read   the code " }, { text: "edit", status: "in-progress" }]);
	assert.deepEqual(items, [
		{ text: "read the code", status: "pending" },
		{ text: "edit", status: "in-progress" },
	]);
	assert.throws(() => normalizeItems([]), /at least one/);
	assert.throws(() => normalizeItems([{ text: "" }]), /text is required/);
	assert.throws(() => normalizeItems([{ text: "x", status: "done" }]), /status must be one of/);
	assert.throws(() => normalizeItems("nope"), /must be an array/);
	assert.equal(normalizeItems([{ text: "a".repeat(500) }])[0]!.text.length, 200);
});

test("current step: in-progress, else first pending, else the last", () => {
	assert.deepEqual(currentStep([{ text: "a", status: "completed" }, { text: "b", status: "in-progress" }, { text: "c", status: "pending" }]), { text: "b", position: 2, total: 3 });
	assert.deepEqual(currentStep([{ text: "a", status: "completed" }, { text: "c", status: "pending" }]), { text: "c", position: 2, total: 2 });
	assert.deepEqual(currentStep([{ text: "a", status: "completed" }, { text: "b", status: "completed" }]), { text: "b", position: 2, total: 2 });
	assert.equal(currentStep([]), undefined);
});

test("summary and lines", () => {
	const items = normalizeItems([{ text: "a", status: "completed" }, { text: "b", status: "in-progress" }]);
	assert.equal(summarize(items), "Checklist saved: 1/2 completed. Current step (2/2): b");
	assert.equal(summarize(normalizeItems([{ text: "a", status: "completed" }])), "Checklist saved: 1/1 completed. All steps completed.");
	assert.deepEqual(renderLines(items), ["☑ a", "◐ b"]);
});

test("payload: items, optional session, markers only when set", () => {
	const items = normalizeItems([{ text: "a" }]);
	assert.deepEqual(buildPayload({}, items), { items: [{ text: "a", status: "pending" }] });
	assert.deepEqual(buildPayload({ sessionId: "s1", blocked: true }, items), { items: [{ text: "a", status: "pending" }], sessionId: "s1", blocked: true });
	assert.deepEqual(buildPayload({ absent: true }), { items: [], absent: true });
	assert.deepEqual(buildPayload({ sessionId: "s2", reset: true }), { items: [], sessionId: "s2", reset: true });
});

test("publish: a blocked/absent marker must not clobber a plan this task already wrote", () => {
	const items = { items: [{ text: "edit", status: "in-progress" as const }] };
	const blocked = { items: [], blocked: true as const };
	const absent = { items: [], absent: true as const };
	const reset = { items: [], reset: true as const };
	assert.equal(shouldPublish(blocked, false), true); // new task, clear the previous plan
	assert.equal(shouldPublish(blocked, true), false); // parallel bash after checklist
	assert.equal(shouldPublish(absent, false), true); // always-mode reminder
	assert.equal(shouldPublish(absent, true), false);
	assert.equal(shouldPublish(items, false), true);
	assert.equal(shouldPublish(items, true), true);
	assert.equal(shouldPublish(reset, false), true);
	assert.equal(shouldPublish(reset, true), true);
});

test("payload: absent and blocked markers carry no items — stale steps must not read as this task's plan", () => {
	const items = normalizeItems([{ text: "old task" }]);
	assert.deepEqual(buildPayload({ sessionId: "s", blocked: true }), { items: [], sessionId: "s", blocked: true });
	assert.deepEqual(buildPayload({ sessionId: "s", absent: true }), { items: [], sessionId: "s", absent: true });
	assert.ok(items.length === 1); // the caller's list is untouched, just not sent with the markers
});

test("server url: PICODE_URL rejects userinfo and paths, server.json keeps its looser shape", () => {
	assert.equal(resolveServerUrl({ PICODE_URL: "https://user:pass@box:8445" }, null).ok, false);
	assert.equal(resolveServerUrl({ PICODE_URL: "https://box:8445/prefix" }, null).ok, false);
	assert.deepEqual(resolveServerUrl({ PICODE_URL: "https://box:8445/" }, null), { ok: true, url: "https://box:8445" });
	assert.deepEqual(resolveServerUrl({ PICODE_URL: "http://127.0.0.1:9999" }, null), { ok: true, url: "http://127.0.0.1:9999" });
});

test("items: truncation is code-point safe (no lone surrogates)", () => {
	// 199 code points of 'a' + one astral emoji = 201 code units / 200 code
	// points. A code-unit slice cuts the emoji in half; a code-point slice
	// keeps it whole.
	const cut = normalizeItems([{ text: "a".repeat(199) + "🎉" }])[0]!.text;
	assert.equal(Array.from(cut).length, 200);
	assert.ok(cut.endsWith("🎉"), "astral char survives whole");
});

test("reconstruct: the last checklist toolResult wins; other entries ignored", () => {
	const entries = [
		{ type: "message", message: { role: "user" } },
		{ type: "message", message: { role: "toolResult", toolName: "checklist", details: { items: [{ text: "old" }] } } },
		{ type: "message", message: { role: "toolResult", toolName: "bash", details: {} } },
		{ type: "message", message: { role: "toolResult", toolName: "checklist", details: { items: [{ text: "new", status: "completed" }] } } },
	];
	assert.deepEqual(reconstruct(entries), [{ text: "new", status: "completed" }]);
	assert.equal(reconstruct([]), null);
});
