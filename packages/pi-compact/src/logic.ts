/** Pure compaction-policy logic. No pi imports — node:test covers this. */

export type ThinkingLevel =
	| "off"
	| "minimal"
	| "low"
	| "medium"
	| "high"
	| "xhigh"
	| "max";

export type CompactConfig = {
	enabled: boolean;
	/** Absolute token trigger. null = do not use this knob. */
	atTokens: number | null;
	/** Fraction of the context window (0–1). null = do not use this knob. */
	atPercent: number | null;
	floorTokens: number;
	/** Primary summarizer `provider/id`, or null to use the auto chain. */
	model: string | null;
	fallback: string[];
	thinking: ThinkingLevel;
	instructions: string;
	cooldownTurns: number;
};

export const THINKING_LEVELS: ThinkingLevel[] = [
	"off",
	"minimal",
	"low",
	"medium",
	"high",
	"xhigh",
	"max",
];

const THINKING = new Set<string>(THINKING_LEVELS);

export const DEFAULT_CONFIG: CompactConfig = {
	enabled: true,
	atTokens: 100_000,
	atPercent: 0.5,
	floorTokens: 32_000,
	model: null,
	fallback: [],
	thinking: "off",
	instructions: "",
	cooldownTurns: 2,
};

/** Cheap summarizers tried after the configured model/fallback. */
export const AUTO_SUMMARIZERS = [
	"google/gemini-2.5-flash",
	"google/gemini-2.5-flash-lite",
	"anthropic/claude-haiku-4-5",
] as const;

export const BACK = "‹ back";
export const SCOPE_AGENT = "this agent";
export const SCOPE_WORKSPACE = "workspace";
export const SKIP_FALLBACK = "no fallback";

export type PickScope = "agent" | "workspace";

export type ParseResult =
	| { ok: true; layer: Partial<CompactConfig> }
	| { ok: false; error: string };

export type CompactCommand =
	| { kind: "trigger"; instructions?: string }
	| { kind: "edit" }
	| { kind: "on" }
	| { kind: "off" }
	| { kind: "model" };

const RESERVED = new Set(["edit", "on", "off", "model"]);

export const WHEN_PRESETS = [
	{
		id: "recommended",
		label: "100k or 50% of window (recommended)",
		atTokens: 100_000 as number | null,
		atPercent: 0.5 as number | null,
	},
	{
		id: "tokens",
		label: "100,000 tokens",
		atTokens: 100_000 as number | null,
		atPercent: null as number | null,
	},
	{
		id: "percent",
		label: "50% of the context window",
		atTokens: null as number | null,
		atPercent: 0.5 as number | null,
	},
	{ id: "custom", label: "custom…" },
] as const;

export type WhenPresetId = (typeof WHEN_PRESETS)[number]["id"];

/**
 * Why a summarizer response must not become the compaction checkpoint,
 * or null when it is safe. Mirrors Pi's own summarizer rule: a length
 * stop holds partial text; empty text carries nothing.
 */
export function summaryBlocked(response: { stopReason?: string }, text: string): string | null {
	if (response.stopReason === "length") return "length";
	if (!text.trim()) return "empty";
	return null;
}

export function parseAgentKey(raw: string | undefined): string | null {
	if (!raw) return null;
	const s = raw.trim();
	if (!/^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$/.test(s)) return null;
	return s;
}

export function overlayRel(key: string): string {
	return `.pi/compact/${key}.json`;
}

export function parseModelId(model: string): { provider: string; id: string } | null {
	const i = model.indexOf("/");
	if (i <= 0 || i === model.length - 1) return null;
	return { provider: model.slice(0, i), id: model.slice(i + 1) };
}

export function groupModels(models: string[]): Map<string, string[]> {
	const map = new Map<string, string[]>();
	for (const m of models) {
		const parsed = parseModelId(m);
		if (!parsed) continue;
		const list = map.get(parsed.provider) ?? [];
		if (!list.includes(parsed.id)) list.push(parsed.id);
		map.set(parsed.provider, list);
	}
	return map;
}

export function providersOf(models: string[]): string[] {
	return [...groupModels(models).keys()].sort();
}

export function idsForProvider(models: string[], provider: string): string[] {
	return groupModels(models).get(provider) ?? [];
}

function isObject(v: unknown): v is Record<string, unknown> {
	return v !== null && typeof v === "object" && !Array.isArray(v);
}

function parseThinking(v: unknown, path: string): ThinkingLevel | string {
	if (typeof v !== "string" || !THINKING.has(v)) {
		return `${path} must be a thinking level`;
	}
	return v as ThinkingLevel;
}

function parseOptionalModel(v: unknown, path: string): { ok: true; value: string | null } | { ok: false; error: string } {
	if (v === null || v === "") return { ok: true, value: null };
	if (typeof v !== "string") return { ok: false, error: `${path} must be provider/id or null` };
	if (!parseModelId(v)) return { ok: false, error: `${path} must be provider/id` };
	return { ok: true, value: v };
}

function parseIntLike(v: unknown, path: string, opts: { min: number; allowNull: true }): number | null | string;
function parseIntLike(v: unknown, path: string, opts: { min: number; allowNull?: false }): number | string;
function parseIntLike(v: unknown, path: string, opts: { min: number; allowNull?: boolean }): number | null | string {
	if (opts.allowNull && v === null) return null;
	if (typeof v !== "number" || !Number.isFinite(v) || !Number.isInteger(v) || v < opts.min) {
		return `${path} must be an integer ≥ ${opts.min}`;
	}
	return v;
}

function parsePercent(v: unknown, path: string): number | null | string {
	if (v === null) return null;
	if (typeof v !== "number" || !Number.isFinite(v) || v <= 0 || v > 1) {
		return `${path} must be a number in (0, 1] or null`;
	}
	return v;
}

/** Parse a compact.json object. Unknown keys are ignored. Missing keys stay unset. */
export function parseConfig(raw: unknown): ParseResult {
	if (!isObject(raw)) return { ok: false, error: "config must be a JSON object" };
	const layer: Partial<CompactConfig> = {};

	if ("enabled" in raw) {
		if (typeof raw.enabled !== "boolean") return { ok: false, error: "enabled must be a boolean" };
		layer.enabled = raw.enabled;
	}
	if ("atTokens" in raw) {
		const v = parseIntLike(raw.atTokens, "atTokens", { min: 1, allowNull: true });
		if (typeof v === "string") return { ok: false, error: v };
		layer.atTokens = v;
	}
	if ("atPercent" in raw) {
		const v = parsePercent(raw.atPercent, "atPercent");
		if (typeof v === "string") return { ok: false, error: v };
		layer.atPercent = v;
	}
	if ("floorTokens" in raw) {
		const v = parseIntLike(raw.floorTokens, "floorTokens", { min: 1, allowNull: false });
		if (typeof v === "string") return { ok: false, error: v };
		layer.floorTokens = v;
	}
	if ("model" in raw) {
		const v = parseOptionalModel(raw.model, "model");
		if (!v.ok) return v;
		layer.model = v.value;
	}
	if ("fallback" in raw) {
		if (!Array.isArray(raw.fallback)) return { ok: false, error: "fallback must be an array of provider/id" };
		const fallback: string[] = [];
		for (const item of raw.fallback) {
			if (typeof item !== "string" || !parseModelId(item)) {
				return { ok: false, error: "fallback entries must be provider/id" };
			}
			if (!fallback.includes(item)) fallback.push(item);
		}
		layer.fallback = fallback;
	}
	if ("thinking" in raw) {
		const v = parseThinking(raw.thinking, "thinking");
		if (typeof v === "string" && !THINKING.has(v)) return { ok: false, error: v };
		layer.thinking = v as ThinkingLevel;
	}
	if ("instructions" in raw) {
		if (typeof raw.instructions !== "string") return { ok: false, error: "instructions must be a string" };
		layer.instructions = raw.instructions;
	}
	if ("cooldownTurns" in raw) {
		const v = parseIntLike(raw.cooldownTurns, "cooldownTurns", { min: 0, allowNull: false });
		if (typeof v === "string") return { ok: false, error: v };
		layer.cooldownTurns = v;
	}
	return { ok: true, layer };
}

export function applyLayer(base: CompactConfig, layer: Partial<CompactConfig>): CompactConfig {
	return {
		enabled: layer.enabled ?? base.enabled,
		atTokens: layer.atTokens !== undefined ? layer.atTokens : base.atTokens,
		atPercent: layer.atPercent !== undefined ? layer.atPercent : base.atPercent,
		floorTokens: layer.floorTokens ?? base.floorTokens,
		model: layer.model !== undefined ? layer.model : base.model,
		fallback: layer.fallback ?? base.fallback,
		thinking: layer.thinking ?? base.thinking,
		instructions: layer.instructions ?? base.instructions,
		cooldownTurns: layer.cooldownTurns ?? base.cooldownTurns,
	};
}

export function effectiveConfig(workspace: Partial<CompactConfig>, overlay?: Partial<CompactConfig>): CompactConfig {
	const mid = applyLayer(DEFAULT_CONFIG, workspace);
	return overlay ? applyLayer(mid, overlay) : mid;
}

/** Merge the layer back onto original JSON so unknown keys survive. */
export function serializeLayer(
	layer: Partial<CompactConfig>,
	raw?: Record<string, unknown>,
): Record<string, unknown> {
	const out: Record<string, unknown> = { ...(raw ?? {}) };
	if (layer.enabled !== undefined) out.enabled = layer.enabled;
	if (layer.atTokens !== undefined) out.atTokens = layer.atTokens;
	if (layer.atPercent !== undefined) out.atPercent = layer.atPercent;
	if (layer.floorTokens !== undefined) out.floorTokens = layer.floorTokens;
	if (layer.model !== undefined) out.model = layer.model;
	if (layer.fallback !== undefined) out.fallback = layer.fallback;
	if (layer.thinking !== undefined) out.thinking = layer.thinking;
	if (layer.instructions !== undefined) out.instructions = layer.instructions;
	if (layer.cooldownTurns !== undefined) out.cooldownTurns = layer.cooldownTurns;
	return out;
}

/**
 * `/compact` with no args triggers. Sole reserved words are subcommands.
 * Anything with more text is instructions, even if it starts with `edit`.
 */
export function parseCompactArgs(args: string): CompactCommand {
	const trimmed = (args ?? "").trim();
	if (!trimmed) return { kind: "trigger" };
	const space = trimmed.indexOf(" ");
	const verb = (space < 0 ? trimmed : trimmed.slice(0, space)).toLowerCase();
	const rest = space < 0 ? "" : trimmed.slice(space + 1).trim();
	if (rest === "" && RESERVED.has(verb)) {
		return { kind: verb as "edit" | "on" | "off" | "model" };
	}
	return { kind: "trigger", instructions: trimmed };
}

export type TriggerDecision =
	| { trigger: false; reason: "disabled" | "unknown-tokens" | "below-floor" | "cooldown" | "under-threshold" }
	| { trigger: true; reason: "tokens" | "percent" | "tokens+percent" };

export function shouldTrigger(opts: {
	config: CompactConfig;
	sessionEnabled: boolean;
	tokens: number | null;
	contextWindow: number;
	turnsSinceCompact: number;
}): TriggerDecision {
	if (!opts.sessionEnabled) return { trigger: false, reason: "disabled" };
	if (opts.tokens === null) return { trigger: false, reason: "unknown-tokens" };
	if (opts.tokens < opts.config.floorTokens) return { trigger: false, reason: "below-floor" };
	if (opts.config.cooldownTurns > 0 && opts.turnsSinceCompact < opts.config.cooldownTurns) {
		return { trigger: false, reason: "cooldown" };
	}
	const hitTokens = opts.config.atTokens !== null && opts.tokens >= opts.config.atTokens;
	const hitPercent =
		opts.config.atPercent !== null &&
		opts.contextWindow > 0 &&
		opts.tokens >= opts.config.atPercent * opts.contextWindow;
	if (hitTokens && hitPercent) return { trigger: true, reason: "tokens+percent" };
	if (hitTokens) return { trigger: true, reason: "tokens" };
	if (hitPercent) return { trigger: true, reason: "percent" };
	return { trigger: false, reason: "under-threshold" };
}

/** Earliest fire point, for status text. null = no early trigger configured. */
export function thresholdTokens(config: CompactConfig, contextWindow: number): number | null {
	const parts: number[] = [];
	if (config.atTokens !== null) parts.push(config.atTokens);
	if (config.atPercent !== null && contextWindow > 0) {
		parts.push(Math.floor(config.atPercent * contextWindow));
	}
	if (parts.length === 0) return null;
	const earliest = Math.min(...parts);
	return Math.max(config.floorTokens, earliest);
}

export function summarizerCandidates(config: CompactConfig, sessionModel: string | null): string[] {
	const out: string[] = [];
	const add = (m: string | null | undefined) => {
		if (!m) return;
		if (!parseModelId(m)) return;
		if (!out.includes(m)) out.push(m);
	};
	add(config.model);
	for (const f of config.fallback) add(f);
	for (const a of AUTO_SUMMARIZERS) add(a);
	add(sessionModel);
	return out;
}

export function pickSummarizer(candidates: string[], usable: (model: string) => boolean): string | null {
	for (const c of candidates) {
		if (usable(c)) return c;
	}
	return null;
}

export function whenPresetId(config: CompactConfig): WhenPresetId {
	for (const p of WHEN_PRESETS) {
		if (p.id === "custom") continue;
		if (p.atTokens === config.atTokens && p.atPercent === config.atPercent) return p.id;
	}
	return "custom";
}

export function applyWhenPreset(config: CompactConfig, id: Exclude<WhenPresetId, "custom">): CompactConfig {
	const p = WHEN_PRESETS.find((x) => x.id === id);
	if (!p || p.id === "custom") return config;
	return { ...config, atTokens: p.atTokens, atPercent: p.atPercent };
}

export function parseTokenInput(raw: string, allowNone: boolean): { ok: true; value: number | null } | { ok: false; error: string } {
	const s = raw.trim().toLowerCase();
	if (allowNone && (s === "" || s === "none" || s === "off")) return { ok: true, value: null };
	const n = Number(s.replace(/_/g, "").replace(/,/g, ""));
	if (!Number.isInteger(n) || n < 1) return { ok: false, error: "Enter a positive integer, or none" };
	return { ok: true, value: n };
}

export function parsePercentInput(raw: string, allowNone: boolean): { ok: true; value: number | null } | { ok: false; error: string } {
	const s = raw.trim().toLowerCase();
	if (allowNone && (s === "" || s === "none" || s === "off")) return { ok: true, value: null };
	const n = Number(s.replace(/%/g, ""));
	if (!Number.isFinite(n)) return { ok: false, error: "Enter a fraction like 0.5, or none" };
	const frac = n > 1 ? n / 100 : n;
	if (frac <= 0 || frac > 1) return { ok: false, error: "Percent must be in (0, 1] or 1–100" };
	return { ok: true, value: frac };
}

export function statusLine(opts: {
	enabled: boolean;
	tokens: number | null;
	contextWindow: number;
	threshold: number | null;
	model: string | null;
}): string {
	const tok = opts.tokens === null ? "?" : opts.tokens.toLocaleString("en");
	const lim = opts.threshold === null ? "overflow only" : opts.threshold.toLocaleString("en");
	const on = opts.enabled ? "on" : "off";
	const model = opts.model ? ` · ${opts.model}` : "";
	return `compact ${on} · ${tok} / ${lim}${model}`;
}

export function combineInstructions(config: CompactConfig, extra?: string): string | undefined {
	const a = config.instructions.trim();
	const b = (extra ?? "").trim();
	if (!a && !b) return undefined;
	if (a && b) return `${a}\n${b}`;
	return a || b;
}
