import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
	parseConfig,
	parseModelId,
	providersOf,
	idsForProvider,
	wantsVision,
	decideOnInput,
	lockRole,
	resolveRole,
	editRole,
	addCustom,
	removeCustom,
	upsertRole,
	mergeConfigs,
	parseAgentKey,
	overlayRel,
	serializeConfig,
	emptyConfig,
	BACK,
	pickStart,
	pickAnswer,
	type PickOutcome,
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

describe("groupModels", () => {
	const models = [
		"xai/grok-4.6",
		"xai/grok-4.5",
		"anthropic/claude-sonnet-4-5",
		"not-a-model",
		"xai/grok-4.6",
	];
	it("lists providers sorted", () => {
		assert.deepEqual(providersOf(models), ["anthropic", "xai"]);
	});
	it("lists ids for one provider, first-seen, no dupes", () => {
		assert.deepEqual(idsForProvider(models, "xai"), ["grok-4.6", "grok-4.5"]);
	});
	it("empty for an unknown provider", () => {
		assert.deepEqual(idsForProvider(models, "zai"), []);
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

describe("editRole", () => {
	it("fills an unset builtin", () => {
		const r = editRole(emptyConfig(), "vision", { model: "xai/grok-4.6", thinking: "high" });
		assert.equal(r.ok, true);
		if (r.ok) assert.equal(r.config.builtin.vision?.model, "xai/grok-4.6");
	});
	it("replaces a custom assignment", () => {
		const r = editRole(full, "redteam", { model: "xai/grok-4.5" });
		assert.equal(r.ok, true);
		if (r.ok) {
			assert.equal(r.config.custom[0].model, "xai/grok-4.5");
			assert.equal(r.config.custom[0].thinking, undefined);
		}
	});
	it("errors on a missing custom role", () => {
		assert.equal(editRole(full, "nope", { model: "xai/grok-4.6" }).ok, false);
	});
	it("errors on a model without provider/id", () => {
		assert.equal(editRole(full, "default", { model: "glm-5.3" }).ok, false);
	});
});

describe("addCustom / removeCustom", () => {
	it("adds a preset", () => {
		const r = addCustom(emptyConfig(), "fast", { model: "xai/grok-build-0.1", thinking: "low" });
		assert.equal(r.ok, true);
		if (r.ok) assert.equal(r.config.custom[0].name, "fast");
	});
	it("rejects reserved, duplicate, and bad names", () => {
		assert.equal(addCustom(full, "vision", { model: "xai/grok-4.6" }).ok, false);
		assert.equal(addCustom(full, "redteam", { model: "xai/grok-4.6" }).ok, false);
		assert.equal(addCustom(full, "red team", { model: "xai/grok-4.6" }).ok, false);
	});
	it("removes a custom role", () => {
		const r = removeCustom(full, "redteam");
		assert.equal(r.ok, true);
		if (r.ok) assert.equal(r.config.custom.length, 0);
	});
	it("cannot remove a builtin or a missing name", () => {
		assert.equal(removeCustom(full, "plan").ok, false);
		assert.equal(removeCustom(full, "nope").ok, false);
	});
});

describe("parseAgentKey", () => {
	it("accepts a slug", () => {
		assert.equal(parseAgentKey("picode-agent-a1b2c3"), "picode-agent-a1b2c3");
	});
	it("rejects empty, path, and overlong values", () => {
		assert.equal(parseAgentKey(undefined), null);
		assert.equal(parseAgentKey(""), null);
		assert.equal(parseAgentKey("../etc"), null);
		assert.equal(parseAgentKey("a/b"), null);
		assert.equal(parseAgentKey("a".repeat(65)), null);
	});
	it("builds the overlay path", () => {
		assert.equal(overlayRel("writer"), ".pi/roles/writer.json");
	});
});

describe("mergeConfigs", () => {
	it("overlay wins builtin slots; workspace slots stay", () => {
		const base = cfg({
			builtin: {
				default: { model: "zai/glm-5.3" },
				vision: { model: "xai/grok-4.6" },
			},
		});
		const overlay = cfg({
			builtin: { default: { model: "anthropic/claude-sonnet-4-5", thinking: "high" } },
		});
		const m = mergeConfigs(base, overlay);
		assert.deepEqual(m.builtin.default, {
			model: "anthropic/claude-sonnet-4-5",
			thinking: "high",
		});
		assert.deepEqual(m.builtin.vision, { model: "xai/grok-4.6" });
		assert.equal(m.builtin.plan, undefined);
	});
	it("overlay replaces a custom by name and appends new ones", () => {
		const base = cfg({
			custom: [
				{ name: "redteam", model: "kimi-coding/k3" },
				{ name: "fast", model: "zai/glm-5.3" },
			],
		});
		const overlay = cfg({
			custom: [
				{ name: "redteam", model: "xai/grok-4.6", thinking: "low" },
				{ name: "writer", model: "anthropic/claude-sonnet-4-5" },
			],
		});
		const m = mergeConfigs(base, overlay);
		assert.deepEqual(
			m.custom.map((c) => c.name),
			["redteam", "fast", "writer"],
		);
		assert.deepEqual(m.custom[0], { name: "redteam", model: "xai/grok-4.6", thinking: "low" });
	});
	it("empty overlay is a no-op", () => {
		assert.deepEqual(mergeConfigs(full, emptyConfig()), full);
	});
});

describe("upsertRole", () => {
	it("adds a custom that this layer does not have yet", () => {
		const r = upsertRole(emptyConfig(), "redteam", { model: "xai/grok-4.6" });
		assert.equal(r.ok, true);
		if (r.ok) assert.equal(r.config.custom[0].name, "redteam");
	});
	it("edits a builtin on an empty layer", () => {
		const r = upsertRole(emptyConfig(), "default", { model: "zai/glm-5.3" });
		assert.equal(r.ok, true);
		if (r.ok) assert.deepEqual(r.config.builtin.default, { model: "zai/glm-5.3" });
	});
});

describe("serializeConfig", () => {
	it("keeps unknown root keys", () => {
		const raw = { extra: true, builtin: { default: { model: "xai/grok-4.6" } } };
		const parsed = cfg(raw);
		const edited = editRole(parsed, "default", { model: "xai/grok-4.5", thinking: "low" });
		assert.equal(edited.ok, true);
		if (!edited.ok) return;
		const out = serializeConfig(edited.config, raw);
		assert.equal(out.extra, true);
		assert.deepEqual((out.builtin as { default: unknown }).default, {
			model: "xai/grok-4.5",
			thinking: "low",
		});
	});
});

describe("pickStart / pickAnswer", () => {
	const models = ["xai/grok-4.6", "xai/grok-4.5", "anthropic/opus", "anthropic/sonnet"];
	const oneProvider = ["xai/grok-4.6", "xai/grok-4.5"];
	const oneOfEach = ["xai/grok-4.6"];

	function ask(out: PickOutcome): Extract<PickOutcome, { kind: "ask" }> {
		assert.equal(out.kind, "ask");
		return out as Extract<PickOutcome, { kind: "ask" }>;
	}

	it("walks provider → model → thinking → done", () => {
		let out = ask(pickStart(models, false));
		assert.equal(out.state.stage, "provider");
		assert.deepEqual(out.options, ["anthropic", "xai"]);
		out = ask(pickAnswer(out.state, "xai"));
		assert.equal(out.state.stage, "model");
		assert.deepEqual(out.options, ["grok-4.6", "grok-4.5", BACK]);
		out = ask(pickAnswer(out.state, "grok-4.5"));
		assert.equal(out.state.stage, "thinking");
		assert.ok(out.options.includes(BACK));
		const done = pickAnswer(out.state, "medium");
		assert.deepEqual(done, {
			kind: "done",
			assignment: { model: "xai/grok-4.5", thinking: "medium" },
		});
	});

	it("thinking 'none' omits the level", () => {
		let out = ask(pickStart(oneOfEach, false));
		assert.equal(out.state.stage, "thinking");
		const done = pickAnswer(out.state, "none");
		assert.deepEqual(done, { kind: "done", assignment: { model: "xai/grok-4.6" } });
	});

	it("single provider and model skip straight to thinking without BACK", () => {
		const out = ask(pickStart(oneOfEach, false));
		assert.equal(out.state.stage, "thinking");
		assert.equal(out.options.includes(BACK), false);
	});

	it("row 4: BACK from thinking reaches model, then provider", () => {
		let out = ask(pickStart(models, false));
		out = ask(pickAnswer(out.state, "xai"));
		out = ask(pickAnswer(out.state, "grok-4.6"));
		assert.equal(out.state.stage, "thinking");
		out = ask(pickAnswer(out.state, BACK));
		assert.equal(out.state.stage, "model");
		out = ask(pickAnswer(out.state, BACK));
		assert.equal(out.state.stage, "provider");
	});

	it("row 5: BACK from thinking skips an unasked model select", () => {
		let out = ask(pickStart(["xai/grok-4.6", "anthropic/opus"], false));
		out = ask(pickAnswer(out.state, "xai"));
		assert.equal(out.state.stage, "thinking"); // only one xai model
		out = ask(pickAnswer(out.state, BACK));
		assert.equal(out.state.stage, "provider");
	});

	it("BACK past the first field returns 'back' only with a prior field", () => {
		let out = ask(pickStart(models, true));
		assert.deepEqual(out.options, ["anthropic", "xai", BACK]);
		assert.deepEqual(pickAnswer(out.state, BACK), { kind: "back" });
		out = ask(pickStart(models, false));
		assert.equal(out.options.includes(BACK), false);
	});

	it("hasPrior offers BACK even when the provider select is skipped", () => {
		let out = ask(pickStart(oneProvider, true));
		assert.equal(out.state.stage, "model");
		assert.ok(out.options.includes(BACK));
		out = ask(pickAnswer(out.state, "grok-4.5"));
		assert.equal(out.state.stage, "thinking");
		// BACK from thinking lands on the model select again…
		out = ask(pickAnswer(out.state, BACK));
		assert.equal(out.state.stage, "model");
		// …and BACK from there (no provider select) steps out to the prior field.
		assert.deepEqual(pickAnswer(out.state, BACK), { kind: "back" });
	});

	it("unknown answers re-ask the same stage", () => {
		const out = ask(pickStart(models, false));
		const again = ask(pickAnswer(out.state, "nope"));
		assert.equal(again.state.stage, "provider");
	});

	it("no models yields none", () => {
		assert.deepEqual(pickStart([], false), { kind: "none" });
	});
});
