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
	reconstruct,
	REFUSAL,
	renderLines,
	summarize,
} from "../src/logic.ts";

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
	assert.deepEqual(buildPayload(items), { items: [{ text: "a", status: "pending" }] });
	assert.deepEqual(buildPayload(items, { sessionId: "s1", blocked: true }), { items: [{ text: "a", status: "pending" }], sessionId: "s1", blocked: true });
	assert.deepEqual(buildPayload([], { absent: true }), { items: [], absent: true });
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
