import assert from "node:assert/strict";
import { test } from "node:test";
import {
	MAX_BODY,
	MAX_TITLE,
	agentIdentity,
	buildAskPayload,
	buildNotifyPayload,
	parseServerJson,
	parseToken,
	rejectUnauthorizedFor,
	resolveDataDir,
	resolveServerUrl,
	resolveToken,
} from "../src/logic.ts";

test("resolveDataDir prefers PICODE_DATA, falls back to ~/.picode", () => {
	assert.equal(resolveDataDir({ PICODE_DATA: "/tmp/x" }, "/home/u"), "/tmp/x");
	assert.equal(resolveDataDir({ PICODE_DATA: "  " }, "/home/u"), "/home/u/.picode");
	assert.equal(resolveDataDir({}, "/home/u/"), "/home/u/.picode");
});

test("parseServerJson accepts PiCode's shape, rejects garbage", () => {
	const good = parseServerJson('{"url":"https://localhost:8445","scheme":"https","port":8445}');
	assert.deepEqual(good, { ok: true, url: "https://localhost:8445" });
	const slash = parseServerJson('{"url":"http://localhost:8611/"}');
	assert.deepEqual(slash, { ok: true, url: "http://localhost:8611" });
	assert.equal(parseServerJson("not json").ok, false);
	assert.equal(parseServerJson("{}").ok, false);
	assert.equal(parseServerJson('{"url":"ftp://x"}').ok, false);
});

test("agentIdentity uses PICODE_AGENT_ID, falls back honestly", () => {
	assert.deepEqual(agentIdentity({ PICODE_AGENT_ID: "helper-1" }), { sourceKind: "agent", sourceId: "helper-1" });
	assert.deepEqual(agentIdentity({}), { sourceKind: "system", sourceId: "pi (unmanaged)" });
	assert.deepEqual(agentIdentity({ PICODE_AGENT_ID: " " }).sourceKind, "system");
});

test("buildNotifyPayload validates, defaults and clips", () => {
	const p = buildNotifyPayload({ title: "deploy done", body: "all green" }, { PICODE_AGENT_ID: "a1" });
	assert.equal(p.kind, "fyi");
	assert.equal(p.blocking, false);
	assert.equal(p.sourceKind, "agent");
	assert.equal(p.sourceId, "a1");
	assert.equal(p.reason, "agent notification");
	assert.throws(() => buildNotifyPayload({ title: "  " }, {}));
	const long = buildNotifyPayload({ title: "x".repeat(MAX_TITLE + 50), body: "y".repeat(MAX_BODY + 50) }, {});
	assert.equal(long.title.length, MAX_TITLE);
	assert.equal(long.body.length, MAX_BODY);
});

test("buildAskPayload files a blocking question with context", () => {
	const p = buildAskPayload({ question: "Which port?", context: "8080 vs 8445" }, { PICODE_AGENT_ID: "a1" });
	assert.equal(p.kind, "question");
	assert.equal(p.blocking, true);
	assert.equal(p.reason, "agent needs your input");
	assert.equal(p.title, "Which port?");
	assert.ok(p.body.includes("8080 vs 8445"));
	assert.throws(() => buildAskPayload({ question: "" }, {}));
	const bare = buildAskPayload({ question: "Go?" }, {});
	assert.equal(bare.body, "Go?");
	assert.equal(bare.sourceKind, "system");
	assert.equal(bare.sessionPath, undefined);
	const exact = buildAskPayload({ question: "Go?" }, { PICODE_AGENT_ID: "a1" }, " /tmp/a1/session.jsonl ");
	assert.equal(exact.sessionPath, "/tmp/a1/session.jsonl");
});

test("parseToken accepts hex tokens only", () => {
	assert.equal(parseToken(" " + "a".repeat(64) + "\n"), "a".repeat(64));
	assert.equal(parseToken("not a token"), "");
	assert.equal(parseToken(null), "");
});

test("rejectUnauthorizedFor trusts nothing but loopback", () => {
	assert.equal(rejectUnauthorizedFor("https://localhost:8445"), false);
	assert.equal(rejectUnauthorizedFor("https://127.0.0.1:8445"), false);
	assert.equal(rejectUnauthorizedFor("https://box.tail.ts.net:8445"), true);
	assert.equal(rejectUnauthorizedFor("nonsense"), true);
});

test("resolveServerUrl: PICODE_URL wins, else server.json", () => {
	const good = { url: "https://localhost:8445" };
	assert.deepEqual(resolveServerUrl({ PICODE_URL: "https://box.tail.ts.net:8445/" }, JSON.stringify(good)), { ok: true, url: "https://box.tail.ts.net:8445" });
	assert.deepEqual(resolveServerUrl({}, JSON.stringify(good)), { ok: true, url: "https://localhost:8445" });
	assert.equal(resolveServerUrl({}, null).ok, false);
	assert.equal(resolveServerUrl({ PICODE_URL: "box:8445" }, null).ok, false);
	assert.equal(resolveServerUrl({ PICODE_URL: "https://box:8445/api" }, null).ok, false);
});

test("resolveToken: PICODE_TOKEN wins over the file", () => {
	const file = "a".repeat(64);
	const env = "b".repeat(64);
	assert.equal(resolveToken({ PICODE_TOKEN: env }, file), env);
	assert.equal(resolveToken({}, file), file);
	assert.equal(resolveToken({ PICODE_TOKEN: "nope" }, file), file);
	assert.equal(resolveToken({}, null), "");
});
