import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
	parseConfig,
	parseModelId,
	wantsVision,
	decideOnInput,
	lockRole,
	resolveRole,
	type RolesConfig,
	type Mode,
} from "../src/logic.ts";

const cfg = (raw: unknown): RolesConfig => {
	const parsed = parseConfig(raw);
	assert.equal(parsed.ok, true, parsed.ok ? "" : parsed.error);
	if (!parsed.ok) throw new Error(parsed.error);
	return parsed.config;
};

const full = cfg({
	builtin: {
		default: { model: "zai/glm-5.3", thinking: "medium" },
		vision: { model: "xai/grok-4.6", thinking: "high" },
		plan: { model: "zai/glm-5.3", thinking: "max" },
	},
	custom: [{ name: "redteam", model: "kimi-coding/k3", thinking: "low" }],
});

const auto: Mode = { kind: "auto" };

describe("parseModelId", () => {
	it("splits on the first slash so ids may contain slashes", () => {
		assert.deepEqual(parseModelId("openrouter/anthropic/claude-sonnet-4"), {
			provider: "openrouter",
			id: "anthropic/claude-sonnet-4",
		});
	});
	it("rejects missing provider or id", () => {
		assert.equal(parseModelId("glm-5.3"), null);
		assert.equal(parseModelId("/id"), null);
		assert.equal(parseModelId("prov/"), null);
	});
});

describe("parseConfig", () => {
	it("accepts an empty object", () => {
		const p = parseConfig({});
		assert.equal(p.ok, true);
		if (p.ok) {
			assert.deepEqual(p.config.builtin, {});
			assert.deepEqual(p.config.custom, []);
		}
	});
	it("ignores unknown root and builtin keys (forward compatible)", () => {
		const p = parseConfig({
			extra: true,
			builtin: { default: { model: "zai/glm-5.3" }, commit: { model: "xai/grok-4.6" } },
		});
		assert.equal(p.ok, true);
		if (p.ok) {
			assert.deepEqual(p.config.builtin.default, { model: "zai/glm-5.3" });
			assert.equal(p.config.builtin.vision, undefined);
		}
	});
	it("rejects a non-object", () => {
		assert.equal(parseConfig([]).ok, false);
		assert.equal(parseConfig("x").ok, false);
		assert.equal(parseConfig(null).ok, false);
	});
	it("rejects a model without provider/id", () => {
		const p = parseConfig({ builtin: { default: { model: "glm-5.3" } } });
		assert.equal(p.ok, false);
	});
	it("rejects an invalid thinking level", () => {
		const p = parseConfig({
			builtin: { default: { model: "zai/glm-5.3", thinking: "ultra" } },
		});
		assert.equal(p.ok, false);
	});
	it("rejects reserved and duplicate custom names", () => {
		assert.equal(parseConfig({ custom: [{ name: "vision", model: "xai/grok-4.6" }] }).ok, false);
		assert.equal(parseConfig({ custom: [{ name: "auto", model: "xai/grok-4.6" }] }).ok, false);
		assert.equal(
			parseConfig({
				custom: [
					{ name: "redteam", model: "xai/grok-4.6" },
					{ name: "redteam", model: "zai/glm-5.3" },
				],
			}).ok,
			false,
		);
	});
	it("rejects a custom name with spaces", () => {
		assert.equal(parseConfig({ custom: [{ name: "red team", model: "xai/grok-4.6" }] }).ok, false);
	});
});

describe("wantsVision", () => {
	it("is true when images are attached", () => {
		assert.equal(wantsVision("hello", [{ mime: "image/png" }]), true);
	});
	it("is true when the text names an image path", () => {
		assert.equal(wantsVision("look at screenshot.png please", undefined), true);
		assert.equal(wantsVision("docs/foo.JPEG", []), true);
	});
	it("is false for plain text", () => {
		assert.equal(wantsVision("fix the tests", undefined), false);
		assert.equal(wantsVision("", []), false);
	});
});

describe("decideOnInput — decision table", () => {
	const row = (
		name: string,
		input: Parameters<typeof decideOnInput>[0],
		want: ReturnType<typeof decideOnInput>["kind"],
		extra?: (d: ReturnType<typeof decideOnInput>) => void,
	) => {
		it(name, () => {
			const d = decideOnInput(input);
			assert.equal(d.kind, want);
			if (extra) extra(d);
		});
	};

	row("1 missing config is dormant", { config: null, mode: auto, text: "hi" }, "noop");

	row(
		"2 extension-sourced input is ignored",
		{ config: full, mode: auto, text: "x.png", source: "extension", images: [{}] },
		"noop",
	);

	row(
		"3 auto + attached image → vision",
		{ config: full, mode: auto, text: "what is this", images: [{}] },
		"switch",
		(d) => {
			assert.equal(d.kind, "switch");
			if (d.kind === "switch") {
				assert.equal(d.target.model, "xai/grok-4.6");
				assert.equal(d.why, "image detected");
			}
		},
	);

	row(
		"4 auto + image path in text → vision",
		{ config: full, mode: auto, text: "see foo/bar.webp" },
		"switch",
		(d) => {
			assert.equal(d.kind, "switch");
			if (d.kind === "switch") assert.equal(d.target.model, "xai/grok-4.6");
		},
	);

	row(
		"5 auto + image but vision unset → noop (fall through, like omp empty slots)",
		{
			config: cfg({ builtin: { default: { model: "zai/glm-5.3" } } }),
			mode: auto,
			text: "x.png",
		},
		"noop",
	);

	row(
		"6 auto + text → default",
		{ config: full, mode: auto, text: "refactor this" },
		"switch",
		(d) => {
			assert.equal(d.kind, "switch");
			if (d.kind === "switch") {
				assert.equal(d.target.model, "zai/glm-5.3");
				assert.equal(d.target.thinking, "medium");
				assert.equal(d.why, "text");
			}
		},
	);

	row(
		"7 auto + text but default unset → noop",
		{
			config: cfg({ builtin: { vision: { model: "xai/grok-4.6" } } }),
			mode: auto,
			text: "hello",
		},
		"noop",
	);

	row(
		"8 vision lock forces vision even on text",
		{ config: full, mode: { kind: "lock", role: "vision" }, text: "no image here" },
		"switch",
		(d) => {
			assert.equal(d.kind, "switch");
			if (d.kind === "switch") {
				assert.equal(d.target.model, "xai/grok-4.6");
				assert.equal(d.why, "lock /vision");
			}
		},
	);

	row(
		"9 plan lock forces plan on any input",
		{ config: full, mode: { kind: "lock", role: "plan" }, text: "x.png", images: [{}] },
		"switch",
		(d) => {
			assert.equal(d.kind, "switch");
			if (d.kind === "switch") {
				assert.equal(d.target.thinking, "max");
				assert.equal(d.why, "lock /plan");
			}
		},
	);

	row(
		"10 custom lock uses the named preset",
		{ config: full, mode: { kind: "lock", role: "redteam" }, text: "go" },
		"switch",
		(d) => {
			assert.equal(d.kind, "switch");
			if (d.kind === "switch") assert.equal(d.target.model, "kimi-coding/k3");
		},
	);

	row(
		"11 lock on a missing custom role → error",
		{ config: full, mode: { kind: "lock", role: "nope" }, text: "go" },
		"error",
	);

	row(
		"12 vision lock with vision unset → error",
		{
			config: cfg({ builtin: { default: { model: "zai/glm-5.3" } } }),
			mode: { kind: "lock", role: "vision" },
			text: "go",
		},
		"error",
	);

	row(
		"13 plan lock with plan unset → error",
		{
			config: cfg({}),
			mode: { kind: "lock", role: "plan" },
			text: "go",
		},
		"error",
	);
});

describe("lockRole", () => {
	it("errors when there is no config", () => {
		const d = lockRole(null, "vision");
		assert.equal(d.kind, "error");
	});
	it("returns the assignment when the role exists", () => {
		const d = lockRole(full, "redteam");
		assert.equal(d.kind, "switch");
		if (d.kind === "switch") assert.equal(d.target.thinking, "low");
	});
});

describe("resolveRole", () => {
	it("finds builtin and custom", () => {
		assert.equal(resolveRole(full, "default")?.model, "zai/glm-5.3");
		assert.equal(resolveRole(full, "redteam")?.model, "kimi-coding/k3");
		assert.equal(resolveRole(full, "missing"), undefined);
	});
});
