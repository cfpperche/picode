import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
	AUTO_SUMMARIZERS,
	DEFAULT_CONFIG,
	applyLayer,
	applyWhenPreset,
	combineInstructions,
	effectiveConfig,
	parseCompactArgs,
	parseConfig,
	parseModelId,
	parsePercentInput,
	parseTokenInput,
	pickSummarizer,
	shouldTrigger,
	statusLine,
	summarizerCandidates,
	thresholdTokens,
	whenPresetId,
	overlayRel,
	parseAgentKey,
	serializeLayer,
	type CompactConfig,
} from "../src/logic.ts";

const cfg = (raw: unknown): CompactConfig => {
	const parsed = parseConfig(raw);
	if (!parsed.ok) throw new Error(parsed.error);
	return effectiveConfig(parsed.layer);
};

describe("parseModelId", () => {
	it("splits on the first slash so ids may contain slashes", () => {
		assert.deepEqual(parseModelId("openrouter/anthropic/claude-sonnet-4"), {
			provider: "openrouter",
			id: "anthropic/claude-sonnet-4",
		});
	});
	it("rejects bare ids", () => {
		assert.equal(parseModelId("gpt-4"), null);
	});
});

describe("parseConfig", () => {
	it("accepts an empty object as all-defaults", () => {
		assert.deepEqual(cfg({}), DEFAULT_CONFIG);
	});
	it("rejects non-objects", () => {
		const parsed = parseConfig([]);
		assert.equal(parsed.ok, false);
	});
	it("ignores unknown keys", () => {
		const parsed = parseConfig({ enabled: false, extra: 1 });
		assert.equal(parsed.ok, true);
		if (parsed.ok) assert.equal(effectiveConfig(parsed.layer).enabled, false);
	});
	it("accepts atTokens null to disable the absolute knob", () => {
		assert.equal(cfg({ atTokens: null }).atTokens, null);
	});
	it("rejects a bad model id", () => {
		const parsed = parseConfig({ model: "noshift" });
		assert.equal(parsed.ok, false);
	});
	it("rejects atPercent outside (0, 1]", () => {
		assert.equal(parseConfig({ atPercent: 0 }).ok, false);
		assert.equal(parseConfig({ atPercent: 1.2 }).ok, false);
		assert.equal(parseConfig({ atPercent: 0.5 }).ok, true);
	});
});

describe("effectiveConfig overlay", () => {
	it("lets overlay keys win and inherits the rest", () => {
		const got = effectiveConfig({ atTokens: 80_000 }, { model: "google/gemini-2.5-flash" });
		assert.equal(got.atTokens, 80_000);
		assert.equal(got.atPercent, 0.5);
		assert.equal(got.model, "google/gemini-2.5-flash");
	});
	it("treats overlay fallback [] as a real override", () => {
		const got = effectiveConfig({ fallback: ["anthropic/claude-haiku-4-5"] }, { fallback: [] });
		assert.deepEqual(got.fallback, []);
	});
	it("round-trips unknown keys through serializeLayer", () => {
		const out = serializeLayer({ enabled: false }, { extra: true, enabled: true });
		assert.equal(out.extra, true);
		assert.equal(out.enabled, false);
	});
});

describe("parseCompactArgs", () => {
	it("empty args trigger compact", () => {
		assert.deepEqual(parseCompactArgs(""), { kind: "trigger" });
		assert.deepEqual(parseCompactArgs("  "), { kind: "trigger" });
	});
	it("sole reserved words are subcommands, case-insensitive", () => {
		assert.deepEqual(parseCompactArgs("edit"), { kind: "edit" });
		assert.deepEqual(parseCompactArgs("EDIT"), { kind: "edit" });
		assert.deepEqual(parseCompactArgs("on"), { kind: "on" });
		assert.deepEqual(parseCompactArgs("off"), { kind: "off" });
		assert.deepEqual(parseCompactArgs("model"), { kind: "model" });
	});
	it("reserved word plus more text is instructions", () => {
		assert.deepEqual(parseCompactArgs("edit the previous summary"), {
			kind: "trigger",
			instructions: "edit the previous summary",
		});
	});
	it("free text is instructions", () => {
		assert.deepEqual(parseCompactArgs("focus on auth"), {
			kind: "trigger",
			instructions: "focus on auth",
		});
	});
});

describe("shouldTrigger", () => {
	const base = {
		config: DEFAULT_CONFIG,
		sessionEnabled: true,
		contextWindow: 200_000,
		turnsSinceCompact: 2,
	};

	it("does not fire when the session lock is off", () => {
		assert.equal(shouldTrigger({ ...base, sessionEnabled: false, tokens: 150_000 }).reason, "disabled");
	});
	it("session on still fires even if the file says enabled false", () => {
		assert.equal(
			shouldTrigger({
				...base,
				config: { ...DEFAULT_CONFIG, enabled: false },
				sessionEnabled: true,
				tokens: 150_000,
			}).trigger,
			true,
		);
	});
	it("does not fire when tokens are unknown", () => {
		assert.equal(shouldTrigger({ ...base, tokens: null }).reason, "unknown-tokens");
	});
	it("does not fire below the floor even if percent would", () => {
		const config = { ...DEFAULT_CONFIG, atTokens: null, atPercent: 0.5, floorTokens: 32_000 };
		assert.equal(
			shouldTrigger({ ...base, config, contextWindow: 40_000, tokens: 25_000 }).reason,
			"below-floor",
		);
	});
	it("honors cooldown", () => {
		assert.equal(shouldTrigger({ ...base, tokens: 150_000, turnsSinceCompact: 0 }).reason, "cooldown");
		assert.equal(shouldTrigger({ ...base, tokens: 150_000, turnsSinceCompact: 1 }).reason, "cooldown");
		assert.equal(shouldTrigger({ ...base, tokens: 150_000, turnsSinceCompact: 2 }).trigger, true);
	});
	it("fires at atTokens", () => {
		const d = shouldTrigger({ ...base, contextWindow: 1_000_000, tokens: 100_000 });
		assert.deepEqual(d, { trigger: true, reason: "tokens" });
	});
	it("fires at atPercent on a small window", () => {
		const d = shouldTrigger({
			...base,
			config: { ...DEFAULT_CONFIG, atTokens: null, atPercent: 0.5 },
			contextWindow: 200_000,
			tokens: 100_000,
		});
		assert.deepEqual(d, { trigger: true, reason: "percent" });
	});
	it("fires when both knobs are crossed", () => {
		const d = shouldTrigger({ ...base, contextWindow: 200_000, tokens: 120_000 });
		assert.deepEqual(d, { trigger: true, reason: "tokens+percent" });
	});
	it("stays quiet under both knobs", () => {
		assert.equal(shouldTrigger({ ...base, tokens: 40_000, contextWindow: 200_000 }).reason, "under-threshold");
	});
	it("never early-triggers when both knobs are null", () => {
		const config = { ...DEFAULT_CONFIG, atTokens: null, atPercent: null };
		assert.equal(shouldTrigger({ ...base, config, tokens: 500_000 }).reason, "under-threshold");
	});
});

describe("thresholdTokens", () => {
	it("is the earlier of the two knobs, not below the floor", () => {
		assert.equal(thresholdTokens(DEFAULT_CONFIG, 1_000_000), 100_000);
		assert.equal(thresholdTokens(DEFAULT_CONFIG, 200_000), 100_000);
		assert.equal(thresholdTokens({ ...DEFAULT_CONFIG, atTokens: null }, 200_000), 100_000);
	});
	it("is null when no early trigger is configured", () => {
		assert.equal(thresholdTokens({ ...DEFAULT_CONFIG, atTokens: null, atPercent: null }, 200_000), null);
	});
});

describe("summarizerCandidates / pickSummarizer", () => {
	it("orders model, fallback, auto chain, then session", () => {
		const got = summarizerCandidates(
			{ ...DEFAULT_CONFIG, model: "xai/grok-4.6", fallback: ["anthropic/claude-haiku-4-5"] },
			"openai/gpt-5.2",
		);
		assert.equal(got[0], "xai/grok-4.6");
		assert.equal(got[1], "anthropic/claude-haiku-4-5");
		assert.equal(got[2], AUTO_SUMMARIZERS[0]);
		assert.equal(got.at(-1), "openai/gpt-5.2");
	});
	it("dedupes and skips empty", () => {
		const got = summarizerCandidates(
			{ ...DEFAULT_CONFIG, model: AUTO_SUMMARIZERS[0], fallback: [AUTO_SUMMARIZERS[0]] },
			AUTO_SUMMARIZERS[0],
		);
		assert.equal(got.filter((m) => m === AUTO_SUMMARIZERS[0]).length, 1);
	});
	it("picks the first usable candidate", () => {
		const cands = ["missing/x", "google/gemini-2.5-flash", "openai/gpt-5.2"];
		assert.equal(
			pickSummarizer(cands, (m) => m.startsWith("google/")),
			"google/gemini-2.5-flash",
		);
		assert.equal(pickSummarizer(cands, () => false), null);
	});
});

describe("when presets", () => {
	it("recognizes the recommended default", () => {
		assert.equal(whenPresetId(DEFAULT_CONFIG), "recommended");
	});
	it("applies a preset without touching other fields", () => {
		const got = applyWhenPreset({ ...DEFAULT_CONFIG, model: "xai/grok-4.6" }, "tokens");
		assert.equal(got.atTokens, 100_000);
		assert.equal(got.atPercent, null);
		assert.equal(got.model, "xai/grok-4.6");
	});
});

describe("inputs", () => {
	it("parses token input", () => {
		assert.deepEqual(parseTokenInput("100000", false), { ok: true, value: 100000 });
		assert.deepEqual(parseTokenInput("100_000", false), { ok: true, value: 100000 });
		assert.deepEqual(parseTokenInput("none", true), { ok: true, value: null });
		assert.equal(parseTokenInput("nope", false).ok, false);
	});
	it("parses percent as fraction or 1–100", () => {
		assert.deepEqual(parsePercentInput("0.5", false), { ok: true, value: 0.5 });
		assert.deepEqual(parsePercentInput("50", false), { ok: true, value: 0.5 });
		assert.deepEqual(parsePercentInput("50%", false), { ok: true, value: 0.5 });
		assert.deepEqual(parsePercentInput("none", true), { ok: true, value: null });
	});
});

describe("status and instructions", () => {
	it("formats a status line", () => {
		assert.equal(
			statusLine({
				configured: true,
				enabled: true,
				tokens: 92000,
				contextWindow: 200000,
				threshold: 100000,
				model: "google/gemini-3.6-flash",
			}),
			"compact on · 92,000 / 100,000 · google/gemini-3.6-flash",
		);
	});
	it("formats the not-configured line", () => {
		assert.equal(
			statusLine({
				configured: false,
				enabled: true,
				tokens: 92000,
				contextWindow: 200000,
				threshold: 100000,
				model: null,
			}),
			"compact: not configured · /compact edit",
		);
	});
	it("combines config and per-call instructions", () => {
		assert.equal(combineInstructions({ ...DEFAULT_CONFIG, instructions: "keep files" }), "keep files");
		assert.equal(
			combineInstructions({ ...DEFAULT_CONFIG, instructions: "keep files" }, "auth"),
			"keep files\nauth",
		);
		assert.equal(combineInstructions(DEFAULT_CONFIG, "  "), undefined);
	});
});

describe("agent key", () => {
	it("accepts PiCode ids and builds the overlay path", () => {
		assert.equal(parseAgentKey("agent_1"), "agent_1");
		assert.equal(overlayRel("agent_1"), ".pi/compact/agent_1.json");
		assert.equal(parseAgentKey("../x"), null);
		assert.equal(parseAgentKey(""), null);
	});
});

describe("sparse wizard save", () => {
	it("writes only touched keys and preserves the layer's other keys", () => {
		const out = serializeLayer({ model: "google/gemini-2.5-flash", fallback: ["anthropic/claude-haiku-4-5"], thinking: "off" }, { atTokens: 80_000, note: "kept" });
		assert.equal(out.atTokens, 80_000);
		assert.equal(out.note, "kept");
		assert.equal(out.model, "google/gemini-2.5-flash");
		assert.equal("enabled" in out, false);
		assert.equal("floorTokens" in out, false);
	});
	it("a sparse overlay still inherits unset knobs from the workspace", () => {
		const ws = parseConfig({ atTokens: 80_000, floorTokens: 10_000 });
		assert.equal(ws.ok, true);
		if (!ws.ok) return;
		const ov = parseConfig({ model: "google/gemini-2.5-flash" });
		assert.equal(ov.ok, true);
		if (!ov.ok) return;
		const got = effectiveConfig(ws.layer, ov.layer);
		assert.equal(got.atTokens, 80_000);
		assert.equal(got.floorTokens, 10_000);
		assert.equal(got.model, "google/gemini-2.5-flash");
	});
});

describe("applyLayer", () => {
	it("does not treat missing overlay keys as defaults", () => {
		const got = applyLayer(DEFAULT_CONFIG, { enabled: false });
		assert.equal(got.enabled, false);
		assert.equal(got.atTokens, 100_000);
	});
});
