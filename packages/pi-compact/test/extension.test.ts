/**
 * Handler-level tests for extensions/compact.ts with narrow fakes — no pi
 * runtime. Covers the runtime decision rows the pure logic tests cannot:
 * the trigger chain on agent_settled (never mid-run: ctx.compact() aborts
 * the active run), dormancy without a config file, summarizer selection
 * with per-link fallback, and the /compact command surface.
 */
import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import piCompact from "../extensions/compact.ts";

type Handler = (event: any, ctx: any) => Promise<unknown>;

function makeHarness(opts: { config?: object; authed?: string[] } = {}) {
	const handlers = new Map<string, Handler[]>();
	const commands = new Map<string, (args: string, ctx: any) => Promise<void>>();
	const notifications: string[] = [];
	const compactions: Array<{ customInstructions?: string }> = [];

	const authed = new Set(opts.authed ?? ["google/gemini-3.6-flash", "anthropic/claude-haiku-4-5"]);
	const cwd = mkdtempSync(join(tmpdir(), "picompact-"));
	mkdirSync(join(cwd, ".pi"), { recursive: true });
	if (opts.config !== undefined) {
		writeFileSync(join(cwd, ".pi", "compact.json"), JSON.stringify(opts.config) + "\n");
	}

	const model = (id: string) => ({ provider: id.split("/")[0], id: id.split("/")[1] });
	const available = ["google/gemini-3.6-flash", "anthropic/claude-haiku-4-5", "openai/gpt-5.2"].map((id) =>
		model(id),
	);

	const pi: any = {
		on: (event: string, handler: Handler) => {
			const list = handlers.get(event) ?? [];
			list.push(handler);
			handlers.set(event, list);
		},
		registerCommand: (name: string, cmd: { handler: (args: string, ctx: any) => Promise<void> }) => {
			commands.set(name, cmd.handler);
		},
	};

	const makeCtx = (over: Record<string, unknown> = {}) => ({
		cwd,
		hasUI: true,
		isIdle: () => true,
		ui: { notify: (m: string) => notifications.push(m), setStatus: () => {} },
		model: model("openai/gpt-5.2"),
		modelRegistry: {
			getAvailable: () => available,
			find: (provider: string, id: string) =>
				authed.has(`${provider}/${id}`) ? model(`${provider}/${id}`) : undefined,
			hasConfiguredAuth: (m: any) => authed.has(`${m.provider}/${m.id}`),
			complete: async (_m: unknown, _ctx: unknown, _opts: unknown) => {
				throw new Error("complete() not configured for this test");
			},
		},
		getContextUsage: () => ({ tokens: 10_000, contextWindow: 200_000, percent: 5 }),
		compact: (options: { customInstructions?: string } = {}) => {
			compactions.push(options);
		},
		waitForIdle: async () => {},
		...over,
	});

	const ctx = makeCtx();
	const emit = async (event: string, payload: any = {}) => {
		const results = await Promise.all((handlers.get(event) ?? []).map((h) => h(payload, ctx)));
		return results[0];
	};

	const start = async () => {
		piCompact(pi);
		await emit("session_start", { reason: "startup" });
	};

	return { pi, ctx, makeCtx, commands, notifications, compactions, emit, start, authed };
}

const okResponse = (text = "## Goal\nkeep going") => ({
	content: [{ type: "text", text }],
	stopReason: "stop",
	usage: { input: 1, output: 2 },
});

const errResponse = () => ({
	content: [],
	stopReason: "error",
	usage: { input: 0, output: 0 },
});

describe("agent_settled trigger chain (config file = active)", () => {
	it("does not compact below the floor", async () => {
		const h = await makeHarness({ config: { enabled: true } });
		await h.start();
		await h.emit("agent_settled", {});
		assert.equal(h.compactions.length, 0);
	});

	it("compacts at 100k tokens and again only after the cooldown", async () => {
		const h = await makeHarness({ config: { enabled: true } });
		await h.start();
		(h.ctx as any).getContextUsage = () => ({ tokens: 120_000, contextWindow: 1_000_000, percent: 12 });
		await h.emit("agent_settled", {});
		assert.equal(h.compactions.length, 1);
		// Compact finished: onComplete resets the cooldown clock.
		await h.emit("session_compact", { reason: "manual" });
		// Next turn is inside the cooldown (default 2).
		await h.emit("turn_end", { turnIndex: 1 });
		await h.emit("agent_settled", {});
		assert.equal(h.compactions.length, 1);
	});

	it("does not compact while the session lock is off", async () => {
		const h = await makeHarness({ config: { enabled: true } });
		await h.start();
		(h.ctx as any).getContextUsage = () => ({ tokens: 120_000, contextWindow: 200_000, percent: 60 });
		await h.commands.get("compact-off")!("", h.ctx);
		await h.emit("agent_settled", {});
		assert.equal(h.compactions.length, 0);
		await h.commands.get("compact-on")!("", h.ctx);
		await h.emit("agent_settled", {});
		assert.equal(h.compactions.length, 1);
	});

	it("honors enabled:false in the file until /compact on", async () => {
		const h = await makeHarness({ config: { enabled: false } });
		await h.start();
		(h.ctx as any).getContextUsage = () => ({ tokens: 120_000, contextWindow: 200_000, percent: 60 });
		await h.emit("agent_settled", {});
		assert.equal(h.compactions.length, 0);
		await h.commands.get("compact-on")!("", h.ctx);
		await h.emit("agent_settled", {});
		assert.equal(h.compactions.length, 1);
	});

	it("never triggers from turn_end — compaction mid-run aborts the agent", async () => {
		const h = await makeHarness({ config: { enabled: true } });
		await h.start();
		(h.ctx as any).getContextUsage = () => ({ tokens: 300_000, contextWindow: 1_000_000, percent: 30 });
		await h.emit("turn_end", { turnIndex: 0 });
		assert.equal(h.compactions.length, 0);
	});

	it("skips a busy session (isIdle false)", async () => {
		const h = await makeHarness({ config: { enabled: true } });
		await h.start();
		(h.ctx as any).getContextUsage = () => ({ tokens: 300_000, contextWindow: 1_000_000, percent: 30 });
		(h.ctx as any).isIdle = () => false;
		await h.emit("agent_settled", {});
		assert.equal(h.compactions.length, 0);
	});
});

describe("dormant without a config file", () => {
	it("never triggers on agent_settled and shows the not-configured status", async () => {
		const h = await makeHarness();
		await h.start();
		const statuses: string[] = [];
		(h.ctx as any).ui.setStatus = (_k: string, v: string) => statuses.push(v);
		(h.ctx as any).getContextUsage = () => ({ tokens: 300_000, contextWindow: 1_000_000, percent: 30 });
		await h.emit("agent_settled", {});
		assert.equal(h.compactions.length, 0);
		assert.ok(statuses.some((s) => s.includes("not configured")));
	});

	it("session_before_compact returns undefined without calling any model", async () => {
		const h = await makeHarness();
		await h.start();
		let calls = 0;
		(h.ctx as any).modelRegistry.complete = async () => {
			calls += 1;
			return okResponse();
		};
		const result = await h.emit("session_before_compact", {});
		assert.equal(result, undefined);
		assert.equal(calls, 0);
	});

	it("compact-on/off inform while unconfigured; bare compact is not registered", async () => {
		const h = await makeHarness();
		await h.start();
		await h.commands.get("compact-on")!("", h.ctx);
		await h.commands.get("compact-off")!("", h.ctx);
		assert.equal(h.compactions.length, 0);
		assert.equal(h.notifications.filter((n) => n.includes("not configured")).length, 2);
	});
});

describe("session_before_compact summarizer selection", () => {
	const preparation = {
		firstKeptEntryId: "keep-1",
		messagesToSummarize: [{ role: "user", content: [{ type: "text", text: "hello" }], timestamp: 1 }],
		turnPrefixMessages: [],
		isSplitTurn: false,
		tokensBefore: 90_000,
		fileOps: { readFiles: [], modifiedFiles: [] },
		settings: { enabled: true, reserveTokens: 16_384, keepRecentTokens: 20_000 },
	};
	const event = (over: Record<string, unknown> = {}) => ({
		preparation,
		branchEntries: [],
		customInstructions: undefined,
		reason: "manual",
		willRetry: false,
		signal: new AbortController().signal,
		...over,
	});

	it("uses the auto chain and preserves the cut when it works", async () => {
		const h = await makeHarness({ config: { enabled: true } });
		await h.start();
		let completedWith: any = null;
		(h.ctx as any).modelRegistry.complete = async (m: any) => {
			completedWith = `${m.provider}/${m.id}`;
			return okResponse();
		};
		const result = (await h.emit("session_before_compact", event())) as any;
		assert.equal(completedWith, "google/gemini-3.6-flash");
		assert.equal(result.compaction.firstKeptEntryId, "keep-1");
		assert.match(result.compaction.summary, /Goal/);
	});

	it("falls through a model with no auth and lands on the session model", async () => {
		const h = await makeHarness({ config: { enabled: true }, authed: ["openai/gpt-5.2"] });
		await h.start();
		const tried: string[] = [];
		(h.ctx as any).modelRegistry.complete = async (m: any) => {
			tried.push(`${m.provider}/${m.id}`);
			return okResponse();
		};
		const result = (await h.emit("session_before_compact", event())) as any;
		assert.deepEqual(tried, ["openai/gpt-5.2"]);
		assert.equal(result.compaction.details.summarizer, "openai/gpt-5.2");
	});

	it("advances to the next chain link after an error stop", async () => {
		const h = await makeHarness({ config: { enabled: true } });
		await h.start();
		const tried: string[] = [];
		(h.ctx as any).modelRegistry.complete = async (m: any) => {
			const id = `${m.provider}/${m.id}`;
			tried.push(id);
			return id === "google/gemini-3.6-flash" ? errResponse() : okResponse();
		};
		const result = (await h.emit("session_before_compact", event())) as any;
		assert.deepEqual(tried, ["google/gemini-3.6-flash", "anthropic/claude-haiku-4-5"]);
		assert.equal(result.compaction.details.summarizer, "anthropic/claude-haiku-4-5");
	});

	it("falls through every link on a length stop so Pi's summarizer takes over", async () => {
		const h = await makeHarness({ config: { enabled: true } });
		await h.start();
		(h.ctx as any).modelRegistry.complete = async () => ({
			content: [{ type: "text", text: "partial" }],
			stopReason: "length",
			usage: {},
		});
		const result = await h.emit("session_before_compact", event());
		assert.equal(result, undefined);
		assert.ok(h.notifications.some((n) => n.includes("length")));
	});

	it("falls through every link on an empty summary", async () => {
		const h = await makeHarness({ config: { enabled: true } });
		await h.start();
		(h.ctx as any).modelRegistry.complete = async () => okResponse("   ");
		const result = await h.emit("session_before_compact", event());
		assert.equal(result, undefined);
	});

	it("falls through every link when the model call throws", async () => {
		const h = await makeHarness({ config: { enabled: true } });
		await h.start();
		(h.ctx as any).modelRegistry.complete = async () => {
			throw new Error("boom");
		};
		const result = await h.emit("session_before_compact", event());
		assert.equal(result, undefined);
		assert.ok(h.notifications.some((n) => n.includes("boom")));
		assert.ok(h.notifications.some((n) => n.includes("every configured summarizer failed")));
	});
});

describe("/compact-* command surface", () => {
	it("registers only hyphenated invocations — pi's TUI owns bare /compact", async () => {
		const h = await makeHarness();
		await h.start();
		const names = [...h.commands.keys()].sort();
		assert.deepEqual(names, ["compact-edit", "compact-model", "compact-off", "compact-on"]);
	});

	it("edit and model commands do not compact", async () => {
		const h = await makeHarness({ config: { enabled: true } });
		await h.start();
		// Drive edit with a cancel at the first select — ctx.ui.select returns undefined.
		const cancelCtx = h.makeCtx({
			ui: { notify: (m: string) => h.notifications.push(m), setStatus: () => {}, select: async () => undefined },
		});
		await h.commands.get("compact-edit")!("", cancelCtx);
		await h.commands.get("compact-model")!("", cancelCtx);
		assert.equal(h.compactions.length, 0);
	});
});
