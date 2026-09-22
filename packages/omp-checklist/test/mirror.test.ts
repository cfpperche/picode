// The mirror's decision table: one event with one context per case, and the
// POSTs the extension produced are the verdict. Positive cases run in-process
// and await the recorder's own request event; silence cases run the harness
// in a child process, because a pending client request would hold it open —
// a clean exit is the proof that nothing was published.
import test from "node:test";
import assert from "node:assert/strict";
import { createServer } from "node:http";
import { execFile } from "node:child_process";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import ompChecklist from "../extensions/checklist.ts";

const here = dirname(fileURLToPath(import.meta.url));
const harness = join(here, "harness.mjs");
const ENV_KEYS = ["PICODE_URL", "PICODE_TERM_URL", "PICODE_DATA", "PICODE_AGENT_ID", "PICODE_TERM_ID", "HOME"];
const savedEnv = new Map(ENV_KEYS.map((k) => [k, process.env[k]]));
test.after(() => {
	for (const [k, v] of savedEnv) {
		if (v === undefined) delete process.env[k];
		else process.env[k] = v;
	}
});

function sandbox(env) {
	process.env.PICODE_DATA = mkdtempSync(join(tmpdir(), "omp-checklist-"));
	process.env.HOME = process.env.PICODE_DATA;
	process.env.PICODE_URL = "";
	process.env.PICODE_AGENT_ID = env.agentID || "";
	process.env.PICODE_TERM_ID = env.termID || "";
}

function recorder() {
	const got = [];
	const waiters = [];
	const state = { url: "" };
	const server = createServer((req, res) => {
		let raw = "";
		req.on("data", (c) => (raw += c));
		req.on("end", () => {
			got.push({ path: req.url || "", body: raw ? JSON.parse(raw) : null });
			res.statusCode = 200;
			res.end();
			for (let i = waiters.length - 1; i >= 0; i--) {
				if (got.length >= waiters[i].count) {
					waiters[i].resolve();
					waiters.splice(i, 1);
				}
			}
		});
	});
	const listen = Promise.withResolvers();
	const closed = Promise.withResolvers();
	server.on("close", closed.resolve);
	server.listen(0, "127.0.0.1", listen.resolve);
	const started = listen.promise.then(() => {
		state.url = "http://127.0.0.1:" + server.address().port;
		process.env.PICODE_TERM_URL = state.url;
		process.env.PICODE_URL = "";
		process.env.PICODE_DATA = mkdtempSync(join(tmpdir(), "omp-checklist-"));
		process.env.HOME = process.env.PICODE_DATA;
	});
	const waitFor = (count) => {
		if (got.length >= count) return Promise.resolve();
		const entry = Promise.withResolvers();
		waiters.push({ count, resolve: entry.resolve });
		return entry.promise;
	};
	const close = () => {
		server.close(closed.resolve);
		return closed.promise;
	};
	return { got, started, waitFor, close, get url() { return state.url; } };
}

function fakePi() {
	const handlers = new Map();
	ompChecklist({ on(name, handler) { handlers.set(name, handler); } });
	return handlers;
}

function ctx(mode, branch = []) {
	return {
		mode,
		sessionManager: {
			getSessionId: () => "native-omp-1",
			getBranch: () => branch,
		},
	};
}

const phases = (tasks, name = "Setup") => ({ toolName: "todo", details: { phases: [{ name, tasks }] } });
const task = (content, status, extra = {}) => ({ content, status, ...extra });
const item = (text, status) => ({ text, status });
const branchWith = (status) => [
	{
		type: "message",
		message: { role: "toolResult", toolName: "todo", details: { phases: [{ name: "Setup", tasks: [task("only", status)] }] } },
	},
];

// A silent case publishes nothing: the harness child must exit with the
// recorder at zero. The child cannot exit while a request is pending, so the
// join is the deterministic signal — no wall-clock wait.
async function silence(t, rec, event, payload, mode = "tui", branch = "", ids = { termID: "t1" }) {
	const env = {
		...process.env,
		PICODE_TERM_URL: rec.url,
		PICODE_URL: "",
		PICODE_AGENT_ID: ids.agentID || "",
		PICODE_TERM_ID: ids.termID || "",
	};
	await new Promise((resolve, reject) => {
		execFile(process.execPath, [harness, event, payload ? JSON.stringify(payload) : "", mode, branch], { env }, (err) => {
			if (err) reject(err);
			else resolve();
		});
	});
	assert.equal(rec.got.length, 0, "a silent case must publish nothing");
}

test("todo results publish the mapped list to the terminal route", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({ termID: "t1" });
	const handlers = fakePi();
	await handlers.get("tool_result")(phases([task("read the code", "completed"), task("edit", "in_progress"), task("test", "pending")]), ctx("tui"));
	await rec.waitFor(1);
	assert.deepEqual(rec.got, [{ path: "/api/terminals/t1/checklist", body: { items: [item("read the code", "completed"), item("edit", "in-progress"), item("test", "pending")] } }]);
	await rec.close();
});

test("a bound agent id wins over the terminal", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({ agentID: "a1", termID: "t1" });
	const handlers = fakePi();
	await handlers.get("tool_result")(phases([task("only", "in_progress")]), ctx("tui"));
	await rec.waitFor(1);
	assert.deepEqual(rec.got, [{ path: "/api/agents/a1/checklist", body: { items: [item("only", "in-progress")] } }]);
	await rec.close();
});

test("no identity publishes nothing", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({});
	await silence(t, rec, "tool_result", phases([task("only", "pending")]), "tui", "", {});
	await rec.close();
});

test("blocked tasks stay visible and name their blocker", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({ termID: "t1" });
	const handlers = fakePi();
	await handlers.get("tool_result")(phases([task("ship", "blocked", { blocker: "ci red" })]), ctx("tui"));
	await rec.waitFor(1);
	assert.deepEqual(rec.got, [{ path: "/api/terminals/t1/checklist", body: { items: [item("ship (blocked: ci red)", "pending")] } }]);
	await rec.close();
});

test("abandoned tasks leave the plan", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({ termID: "t1" });
	const handlers = fakePi();
	await handlers.get("tool_result")(phases([task("done thing", "completed"), task("given up", "abandoned")]), ctx("tui"));
	await rec.waitFor(1);
	assert.deepEqual(rec.got, [{ path: "/api/terminals/t1/checklist", body: { items: [item("done thing", "completed")] } }]);
	await rec.close();
});

test("multi-phase plans carry the phase name", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({ termID: "t1" });
	const handlers = fakePi();
	const event = {
		toolName: "todo",
		details: { phases: [{ name: "Setup", tasks: [task("read", "completed")] }, { name: "Ship", tasks: [task("land", "pending")] }] },
	};
	await handlers.get("tool_result")(event, ctx("tui"));
	await rec.waitFor(1);
	assert.deepEqual(rec.got, [{ path: "/api/terminals/t1/checklist", body: { items: [item("Setup — read", "completed"), item("Ship — land", "pending")] } }]);
	await rec.close();
});

test("non-todo results and todo results without phases are ignored", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({ termID: "t1" });
	await silence(t, rec, "tool_result", { toolName: "bash", details: { phases: [] } });
	await silence(t, rec, "tool_result", { toolName: "todo", details: {} });
	await rec.close();
});

test("an empty plan resets the row", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({ termID: "t1" });
	const handlers = fakePi();
	await handlers.get("tool_result")(phases([]), ctx("tui"));
	await rec.waitFor(1);
	assert.deepEqual(rec.got, [{ path: "/api/terminals/t1/checklist", body: { reset: true } }]);
	await rec.close();
});

test("session start replays the branch snapshot", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({ termID: "t1" });
	const handlers = fakePi();
	await handlers.get("session_start")({}, ctx("tui", branchWith("in_progress")));
	await rec.waitFor(1);
	assert.deepEqual(rec.got, [{ path: "/api/terminals/t1/checklist", body: { sessionId: "native-omp-1", items: [item("only", "in-progress")] } }]);
	await rec.close();
});

test("session start with an empty branch resets the row", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({ termID: "t1" });
	const handlers = fakePi();
	await handlers.get("session_start")({}, ctx("tui"));
	await rec.waitFor(1);
	assert.deepEqual(rec.got, [{ path: "/api/terminals/t1/checklist", body: { sessionId: "native-omp-1", reset: true } }]);
	await rec.close();
});

test("headless print runs are silent", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({ termID: "t1" });
	await silence(t, rec, "tool_result", phases([task("only", "pending")]), "print");
	await silence(t, rec, "session_start", "", "print", JSON.stringify(branchWith("pending")));
	await rec.close();
});

test("plans over the store cap clip to fifty", { timeout: 10000 }, async (t) => {
	const rec = recorder();
await rec.started;
	sandbox({ termID: "t1" });
	const handlers = fakePi();
	const many = Array.from({ length: 55 }, (_, i) => task("task " + i, "pending"));
	await handlers.get("tool_result")(phases(many), ctx("tui"));
	await rec.waitFor(1);
	assert.equal(rec.got.length, 1);
	assert.equal(rec.got[0].body.items.length, 50);
	await rec.close();
});
