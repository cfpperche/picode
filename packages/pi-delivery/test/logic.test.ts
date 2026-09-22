import assert from "node:assert/strict";
import { test } from "node:test";
import {
	ACTIONS,
	FIELDS_BY_ACTION,
	MAX_REQUEST_ID,
	buildRequest,
	identityFrom,
	parseServerJson,
	parseToken,
	rejectUnauthorizedFor,
	resolveDataDir,
	resolveServerUrl,
	resolveToken,
	stableRequestId,
	summarize,
	type Identity,
} from "../src/logic.ts";

const who: Identity = { agent: "a1", term: "t1" };
const okBody = (input: unknown, identity: Identity = who, sessionId = "s1"): Record<string, unknown> => {
	const built = buildRequest(input, identity, sessionId);
	assert.equal(built.ok, true, built.ok ? "" : built.error);
	return JSON.parse((built as { ok: true; body: string }).body) as Record<string, unknown>;
};
const refused = (input: unknown, identity: Identity = who, sessionId = "s1"): string => {
	const built = buildRequest(input, identity, sessionId);
	assert.equal(built.ok, false, "expected a refusal, got " + JSON.stringify(built));
	return (built as { ok: false; error: string }).error;
};

test("resolveDataDir prefers PICODE_DATA, falls back to ~/.picode", () => {
	assert.equal(resolveDataDir({ PICODE_DATA: "/tmp/x" }, "/home/u"), "/tmp/x");
	assert.equal(resolveDataDir({ PICODE_DATA: "  " }, "/home/u"), "/home/u/.picode");
	assert.equal(resolveDataDir({}, "/home/u/"), "/home/u/.picode");
});

test("parseServerJson accepts PiCode's shape, rejects garbage", () => {
	assert.deepEqual(parseServerJson('{"url":"https://localhost:8445","port":8445}'), { ok: true, url: "https://localhost:8445" });
	assert.equal(parseServerJson("not json").ok, false);
	assert.equal(parseServerJson('{"port":8445}').ok, false);
});

test("the server address is PICODE_URL first, then server.json", () => {
	assert.deepEqual(resolveServerUrl({ PICODE_URL: "https://box:8445/" }, null), { ok: true, url: "https://box:8445" });
	assert.equal(resolveServerUrl({ PICODE_URL: "https://box:8445/path" }, null).ok, false);
	assert.deepEqual(resolveServerUrl({}, '{"url":"https://localhost:8445"}'), { ok: true, url: "https://localhost:8445" });
	assert.equal(resolveServerUrl({}, null).ok, false);
});

test("the bearer is PICODE_TOKEN first, then the token file", () => {
	const token = "a".repeat(40);
	assert.equal(resolveToken({ PICODE_TOKEN: token }, "b".repeat(40)), token);
	assert.equal(resolveToken({}, token), token);
	assert.equal(resolveToken({}, "short"), "");
	assert.equal(parseToken("  " + token + " "), token);
});

test("TLS is relaxed on loopback only", () => {
	assert.equal(rejectUnauthorizedFor("https://localhost:8445"), false);
	assert.equal(rejectUnauthorizedFor("https://127.0.0.1:8445"), false);
	assert.equal(rejectUnauthorizedFor("https://[::1]:8445"), false);
	assert.equal(rejectUnauthorizedFor("https://box:8445"), true);
});

test("identity is the launch: the agent id wins over the terminal", () => {
	assert.deepEqual(identityFrom({ PICODE_AGENT_ID: "a1", PICODE_TERM_ID: "t1" }), { agent: "a1", term: "t1" });
	assert.deepEqual(identityFrom({ PICODE_TERM_ID: "t1" }), { agent: "", term: "t1" });
	assert.deepEqual(identityFrom({}), { agent: "", term: "" });
	assert.equal(refused({ action: "capabilities" }, identityFrom({})), "no PiCode identity: open this agent through PiCode, then call again");
});

test("an unknown action is refused before anything is sent", () => {
	assert.match(refused({ action: "deploy" }), /action must be one of capabilities, register/);
	assert.equal(refused({}), "action must be one of " + ACTIONS.join(", "));
});

// The daemon's shape, mirrored: a field an action does not take is named,
// never dropped (a review action is answered with its own list).
test("each action carries exactly its own fields", () => {
	assert.deepEqual(okBody({ action: "register", title: "T", branch: "b", revision: "r".repeat(40), target: "main" }), {
		action: "register",
		title: "T",
		branch: "b",
		revision: "r".repeat(40),
		target: "main",
		requestId: stableRequestId("s1", "register", { action: "register", title: "T", branch: "b", revision: "r".repeat(40), target: "main" }),
		agent: "a1",
		term: "t1",
	});
	assert.match(refused({ action: "register", title: "T", branch: "b", revision: "r", target: "main", id: "d1" }), /register does not take id — it takes title, branch, revision, target/);
	assert.match(refused({ action: "register", title: "T", branch: "b", revision: "r", target: "main", expectedVersion: 2 }), /register does not take expectedVersion/);
	assert.match(refused({ action: "request-review", id: "d1", expectedVersion: 1, title: "T" }), /request-review does not take title — it takes id, expectedVersion/);
	assert.match(refused({ action: "capabilities", before: 3 }), /capabilities does not take before — it takes no other fields/);
	assert.match(refused({ action: "show", id: "d1", requestId: "k" }), /show does not take requestId/);
	assert.deepEqual(okBody({ action: "list", before: 7 }), { action: "list", before: 7, agent: "a1", term: "t1" });
	assert.deepEqual(okBody({ action: "show", id: "d1" }), { action: "show", id: "d1", agent: "a1", term: "t1" });
	assert.deepEqual(okBody({ action: "capabilities" }), { action: "capabilities", agent: "a1", term: "t1" });
});

test("empty values are left out of the payload", () => {
	assert.deepEqual(okBody({ action: "update", id: "d1", expectedVersion: 2, title: "T", branch: "", revision: null, target: undefined }), {
		action: "update",
		id: "d1",
		expectedVersion: 2,
		title: "T",
		agent: "a1",
		term: "t1",
		requestId: stableRequestId("s1", "update", { action: "update", id: "d1", expectedVersion: 2, title: "T" }),
	});
});

// A retry of the same call must replay, not register twice: the key is
// derived from session + action + payload, and an explicit one wins.
test("mutations carry a retry key that repeats with the call", () => {
	const call = { action: "register", title: "T", branch: "b", revision: "c".repeat(40), target: "main" };
	const first = okBody(call).requestId as string;
	assert.equal(first, okBody(call).requestId);
	assert.equal(first, stableRequestId("s1", "register", { action: "register", title: "T", branch: "b", revision: "c".repeat(40), target: "main" }));
	assert.notEqual(first, okBody(call, who, "s2").requestId);
	assert.notEqual(first, okBody({ ...call, title: "other" }).requestId);
	assert.equal(okBody({ ...call, requestId: "mine" }).requestId, "mine");
	assert.equal((okBody({ ...call, requestId: "x".repeat(MAX_REQUEST_ID + 1) }).requestId as string).length, 40);
	assert.equal(okBody({ action: "capabilities" }).requestId, undefined);
});

test("the terminal identity carries a pi opened as an Agent CLI", () => {
	assert.deepEqual(okBody({ action: "capabilities" }, { agent: "", term: "t9" }), { action: "capabilities", term: "t9" });
});

test("summarize reads the daemon's answer", () => {
	assert.equal(summarize('{"error":"delivery changed"}'), "delivery changed");
	assert.equal(summarize('{"delivery":{"id":"d1","version":3,"branch":"feat/x","target":"main","review":"requested"}}'), "delivery d1 v3 · feat/x → main · review requested");
	assert.equal(summarize('{"delivery":{"id":"d1","version":3,"branch":"feat/x","target":"main","review":"none"},"replayed":true}'), "Replayed delivery d1 v3 · feat/x → main · no review request");
	assert.equal(summarize('{"deliveries":[{"id":"d1"}]}'), "1 delivery record");
	assert.equal(summarize('{"actions":["register"]}'), "no deliveries here yet — register one first");
	assert.equal(summarize(""), "no answer");
});

// Every action the daemon serves is reachable from the tool; the field
// table above is what the schema descriptions promise.
test("the action list matches the field table", () => {
	assert.deepEqual(ACTIONS, ["capabilities", "register", "update", "request-review", "withdraw-review", "show", "list"]);
	assert.deepEqual(Object.keys(FIELDS_BY_ACTION).sort(), [...ACTIONS].sort());
});
