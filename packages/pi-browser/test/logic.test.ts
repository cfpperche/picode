import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import {
	imageBlock,
	parseAnswer,
	rejectUnauthorizedFor,
	resolveDataDir,
	resolveServerUrl,
	resolveToken,
	summarizeAx,
	summarizeCdp,
	summarizeEvaluate,
	summarizeEvents,
	summarizeNavigate,
} from "../src/logic.ts";

test("the daemon is found through PICODE_URL or server.json", () => {
	assert.deepEqual(resolveServerUrl({ PICODE_URL: "https://box:8445/" }, null), { ok: true, url: "https://box:8445" });
	assert.deepEqual(resolveServerUrl({ PICODE_URL: "https://user:pass@box/" }, null), { ok: false, error: "PICODE_URL must be an origin like https://box:8445" });
	assert.deepEqual(resolveServerUrl({}, `{"url":"https://localhost:8445/"}`), { ok: true, url: "https://localhost:8445" });
	assert.equal(resolveServerUrl({}, null).ok, false);
	assert.equal(resolveServerUrl({}, "not json").ok, false);
	assert.equal(resolveServerUrl({}, `{"url":"ftp://x"}`).ok, false);
});

test("only a hex token counts, from env or file", () => {
	assert.equal(resolveToken({ PICODE_TOKEN: "a".repeat(32) }, null), "a".repeat(32));
	assert.equal(resolveToken({}, "b".repeat(64)), "b".repeat(64));
	assert.equal(resolveToken({ PICODE_TOKEN: "short" }, null), "");
	assert.equal(resolveToken({}, null), "");
	assert.equal(resolveDataDir({ PICODE_DATA: "/tmp/picode" }, "/home/u"), "/tmp/picode");
	assert.equal(resolveDataDir({}, "/home/u"), "/home/u/.picode");
});

test("localhost certificates are accepted, remote ones verified", () => {
	assert.equal(rejectUnauthorizedFor("https://localhost:8445"), false);
	assert.equal(rejectUnauthorizedFor("https://127.0.0.1:8445"), false);
	assert.equal(rejectUnauthorizedFor("https://box:8445"), true);
	assert.equal(rejectUnauthorizedFor("not a url"), true);
});

test("the daemon's answer is read, refusals carry their message", () => {
	const ok = parseAnswer(200, `{"verb":"snapshot","output":{"nodes":[]}}`);
	assert.equal(ok.ok, true);
	if (ok.ok) assert.equal(ok.answer.verb, "snapshot");
	const forbidden = parseAnswer(403, `{"error":"snapshot needs the act tier; this agent has read"}`);
	assert.deepEqual(forbidden, { ok: false, error: "snapshot needs the act tier; this agent has read" });
	assert.equal(parseAnswer(502, `{"error":"the desktop app is not connected"}`).ok, false);
	assert.equal(parseAnswer(500, "<html>oops</html>").ok, false);
	assert.equal(parseAnswer(200, "nope").ok, false);
});

test("snapshot renders roles and names, and skips the noise", () => {
	const out = {
		nodes: [
			{ role: { value: "RootWebArea" }, name: { value: "Example" } },
			{ role: { value: "generic" }, name: { value: "" } },
			{ role: { value: "heading" }, name: { value: "  Hello   world " } },
			{ role: { value: "InlineTextBox" }, name: { value: "Hello world" } },
			{ role: { value: "link" }, name: { value: "More" } },
			{ role: { value: "image" }, name: { value: "" } },
			{ role: { value: "textbox" }, name: { value: "" } },
		],
	};
	assert.deepEqual(summarizeAx(out).lines, ['RootWebArea "Example"', 'heading "Hello world"', 'link "More"', "image", "textbox"]);
	assert.deepEqual(summarizeAx(out, 2), { lines: ['RootWebArea "Example"', 'heading "Hello world"'], dropped: 3 });
	assert.deepEqual(summarizeAx(undefined).lines, []);
	assert.deepEqual(summarizeAx({ nodes: "no" }).lines, []);
});

test("events render the tail, long params truncated, a quiet page named", () => {
	const out = {
		last: 7,
		events: [
			{ seq: 6, event: "Page.frameNavigated", params: { url: "https://example.com" } },
			{ seq: 7, event: "Runtime.consoleAPICalled", params: { text: "x".repeat(400) } },
		],
	};
	const lines = summarizeEvents(out);
	assert.equal(lines.length, 2);
	assert.match(lines[0]!, /^Page\.frameNavigated \{/);
	assert.ok(lines[1]!.length < 200, "a long payload is truncated");
	assert.deepEqual(summarizeEvents({ last: 7, events: [] }), ["nothing since the cursor (last #7)"]);
	assert.deepEqual(summarizeEvents({}), []);
});

test("a screenshot becomes an image block the model sees", () => {
	assert.deepEqual(imageBlock({ data: "iVBORw0KGgo=" }), { type: "image", data: "iVBORw0KGgo=", mimeType: "image/png" });
	assert.equal(imageBlock({ data: "" }), null, "an empty capture is no image");
	assert.equal(imageBlock({}), null);
	assert.equal(imageBlock(null), null);
});

test("verbs in the tool are the five the daemon knows", () => {
	// The union in extensions/browser.ts is hand-written; this keeps it honest.
	const source = readFileSync(new URL("../extensions/browser.ts", import.meta.url), "utf8");
	for (const verb of ["snapshot", "screenshot", "events", "evaluate", "navigate"]) assert.match(source, new RegExp(`Type\\.Literal\\("${verb}"\\)`));
});

// The act verbs' answers, found broken by using the tool with a full grant
// (2026-09-17): all three ran and all three arrived as "the page has no
// accessible content", because the AX renderer was the only formatter.
test("evaluate answers with the value, not a tree", () => {
	assert.equal(summarizeEvaluate({ result: { type: "number", value: 2 } }), "2");
	assert.equal(summarizeEvaluate({ result: { type: "string", value: "PI-CODE" } }), "PI-CODE");
	assert.equal(summarizeEvaluate({ result: { type: "object", value: { a: 1 } } }), '{"a":1}');
	assert.equal(summarizeEvaluate({ result: { type: "undefined" } }), "undefined");
	assert.equal(summarizeEvaluate({ result: { type: "function", description: "() => 1" } }), "() => 1");
	assert.equal(summarizeEvaluate({}), "evaluate returned nothing");
});

test("a page exception is an answer about the page, said in one line", () => {
	const out = { exceptionDetails: { text: "Uncaught", exception: { description: "Error: boom\n    at <anonymous>:1:1" } } };
	assert.equal(summarizeEvaluate(out), "the page threw: Error: boom");
});

test("navigate says where it went, or why it did not", () => {
	assert.equal(summarizeNavigate({ frameId: "F1" }, "https://example.com/"), "navigated to https://example.com/");
	assert.equal(summarizeNavigate({ errorText: "net::ERR_NAME_NOT_RESOLVED" }, "https://nope.invalid/"), "navigate failed: net::ERR_NAME_NOT_RESOLVED");
	assert.equal(summarizeNavigate({}), "navigated");
});

test("cdp answers with JSON and a cap", () => {
	assert.equal(summarizeCdp({ cookies: [] }), '{"cookies":[]}');
	const big = summarizeCdp({ blob: "x".repeat(5000) }, 100);
	assert.ok(big.startsWith('{"blob":"xxx'));
	assert.match(big, /\(\d+ more chars\)$/);
	assert.equal(summarizeCdp("plain"), '"plain"');
});
