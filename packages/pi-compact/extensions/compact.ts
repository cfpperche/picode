/**
 * Opt-in compaction policy for pi (ADR-0060).
 *
 * Early-triggers compact at 100k or 50% of the window, summarizes with a
 * cheap model (thinking off), and overlays /compact. Missing config file
 * still applies defaults. With PI_COMPACT_AGENT=<id>, also reads/writes
 * <cwd>/.pi/compact/<id>.json (overlay).
 */
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { randomUUID } from "node:crypto";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import { convertToLlm, serializeConversation } from "@earendil-works/pi-coding-agent";
import {
	applyWhenPreset,
	BACK,
	combineInstructions,
	DEFAULT_CONFIG,
	effectiveConfig,
	idsForProvider,
	overlayRel,
	parseAgentKey,
	parseCompactArgs,
	parseConfig,
	parseModelId,
	parsePercentInput,
	parseTokenInput,
	pickSummarizer,
	providersOf,
	SCOPE_AGENT,
	SCOPE_WORKSPACE,
	serializeLayer,
	shouldTrigger,
	SKIP_FALLBACK,
	statusLine,
	summarizerCandidates,
	thresholdTokens,
	WHEN_PRESETS,
	type CompactConfig,
	type PickScope,
	type ThinkingLevel,
	THINKING_LEVELS,
	whenPresetId,
} from "../src/logic.ts";

const WORKSPACE_REL = join(".pi", "compact.json");

type FileRead =
	| { status: "missing" }
	| { status: "invalid"; error: string }
	| { status: "ok"; layer: Partial<CompactConfig>; raw: Record<string, unknown> };

type Loaded =
	| { status: "invalid"; error: string }
	| {
			status: "ok";
			config: CompactConfig;
			layer: Partial<CompactConfig>;
			raw: Record<string, unknown>;
			writeRel: string;
			overlay: boolean;
	  };

function unique(xs: string[]): string[] {
	const out: string[] = [];
	for (const x of xs) if (!out.includes(x)) out.push(x);
	return out;
}

export default function piCompact(pi: ExtensionAPI) {
	let loaded: Loaded = {
		status: "ok",
		config: DEFAULT_CONFIG,
		layer: {},
		raw: {},
		writeRel: WORKSPACE_REL,
		overlay: false,
	};
	let sessionEnabled = true;
	let turnsSinceCompact = DEFAULT_CONFIG.cooldownTurns;
	let compacting = false;
	let warnedInvalid = false;

	function readFileAt(abs: string, rel: string): FileRead {
		let text: string;
		try {
			text = readFileSync(abs, "utf8");
		} catch (err) {
			const code = (err as NodeJS.ErrnoException).code;
			if (code === "ENOENT") return { status: "missing" };
			return { status: "invalid", error: `Cannot read ${rel}: ${(err as Error).message}` };
		}
		let raw: unknown;
		try {
			raw = JSON.parse(text);
		} catch (err) {
			return { status: "invalid", error: `${rel} is not valid JSON: ${(err as Error).message}` };
		}
		const parsed = parseConfig(raw);
		if (!parsed.ok) return { status: "invalid", error: parsed.error };
		const rawObj = raw !== null && typeof raw === "object" && !Array.isArray(raw)
			? (raw as Record<string, unknown>)
			: {};
		return { status: "ok", layer: parsed.layer, raw: rawObj };
	}

	function writeTarget(): { rel: string; overlay: boolean } {
		const key = parseAgentKey(process.env.PI_COMPACT_AGENT);
		if (key) return { rel: overlayRel(key), overlay: true };
		return { rel: WORKSPACE_REL, overlay: false };
	}

	function statePath(): string {
		const key = parseAgentKey(process.env.PI_COMPACT_AGENT);
		if (!key) return "";
		return join(homedir(), ".pi", "agent", "compact-state", `${key}.json`);
	}

	function reload(cwd: string): Loaded {
		const target = writeTarget();
		const ws = readFileAt(join(cwd, WORKSPACE_REL), WORKSPACE_REL);
		if (ws.status === "invalid") return ws;
		if (!target.overlay) {
			const layer = ws.status === "ok" ? ws.layer : {};
			const raw = ws.status === "ok" ? ws.raw : {};
			return {
				status: "ok",
				config: effectiveConfig(layer),
				layer,
				raw,
				writeRel: target.rel,
				overlay: false,
			};
		}
		const ov = readFileAt(join(cwd, target.rel), target.rel);
		if (ov.status === "invalid") return ov;
		const wsLayer = ws.status === "ok" ? ws.layer : {};
		const ovLayer = ov.status === "ok" ? ov.layer : {};
		const raw = ov.status === "ok" ? ov.raw : {};
		return {
			status: "ok",
			config: effectiveConfig(wsLayer, ovLayer),
			layer: ovLayer,
			raw,
			writeRel: target.rel,
			overlay: true,
		};
	}

	function configOrDefault(): CompactConfig {
		return loaded.status === "ok" ? loaded.config : DEFAULT_CONFIG;
	}

	function writable(ctx: ExtensionContext): {
		config: CompactConfig;
		layer: Partial<CompactConfig>;
		raw: Record<string, unknown>;
		writeRel: string;
		overlay: boolean;
	} | null {
		loaded = reload(ctx.cwd);
		if (loaded.status === "invalid") {
			ctx.ui.notify(loaded.error, "error");
			return null;
		}
		return loaded;
	}

	function save(
		ctx: ExtensionContext,
		layer: Partial<CompactConfig>,
		raw: Record<string, unknown>,
		writeRel: string,
	): boolean {
		const path = join(ctx.cwd, writeRel);
		try {
			mkdirSync(dirname(path), { recursive: true });
			writeFileSync(path, JSON.stringify(serializeLayer(layer, raw), null, 2) + "\n", "utf8");
		} catch (err) {
			ctx.ui.notify(`Could not write ${writeRel}: ${(err as Error).message}`, "error");
			return false;
		}
		loaded = reload(ctx.cwd);
		return loaded.status === "ok";
	}

	function sessionModelId(ctx: ExtensionContext): string | null {
		const m = ctx.model;
		if (!m) return null;
		return `${m.provider}/${m.id}`;
	}

	function listModels(ctx: ExtensionContext): string[] {
		const scoped = ctx.scopedModels;
		if (Array.isArray(scoped) && scoped.length > 0) {
			return unique(scoped.map((e) => `${e.model.provider}/${e.model.id}`));
		}
		return unique(ctx.modelRegistry.getAvailable().map((m) => `${m.provider}/${m.id}`));
	}

	function usable(ctx: ExtensionContext, id: string): boolean {
		const parsed = parseModelId(id);
		if (!parsed) return false;
		const found = ctx.modelRegistry.find(parsed.provider, parsed.id);
		if (!found) return false;
		return ctx.modelRegistry.hasConfiguredAuth(found);
	}

	function currentSummarizer(ctx: ExtensionContext): string | null {
		return pickSummarizer(summarizerCandidates(configOrDefault(), sessionModelId(ctx)), (id) =>
			usable(ctx, id),
		);
	}

	function publishState(ctx: ExtensionContext) {
		const path = statePath();
		if (!path) return;
		const usage = ctx.getContextUsage();
		const cfg = configOrDefault();
		const window = usage?.contextWindow ?? ctx.model?.contextWindow ?? 0;
		const body = {
			v: 1,
			enabled: sessionEnabled && cfg.enabled,
			tokens: usage?.tokens ?? null,
			contextWindow: window || null,
			threshold: thresholdTokens(cfg, window),
			model: currentSummarizer(ctx),
		};
		try {
			mkdirSync(dirname(path), { recursive: true });
			writeFileSync(path, JSON.stringify(body) + "\n", "utf8");
		} catch {
			/* best-effort */
		}
	}

	function paintStatus(ctx: ExtensionContext) {
		if (!ctx.hasUI) return;
		const usage = ctx.getContextUsage();
		const cfg = configOrDefault();
		const window = usage?.contextWindow ?? ctx.model?.contextWindow ?? 0;
		ctx.ui.setStatus(
			"pi-compact",
			statusLine({
				enabled: sessionEnabled && cfg.enabled,
				tokens: usage?.tokens ?? null,
				contextWindow: window,
				threshold: thresholdTokens(cfg, window),
				model: currentSummarizer(ctx),
			}),
		);
		publishState(ctx);
	}

	function triggerCompact(ctx: ExtensionContext, instructions?: string) {
		if (compacting) {
			if (ctx.hasUI) ctx.ui.notify("Compaction already running", "warning");
			return;
		}
		compacting = true;
		if (ctx.hasUI) ctx.ui.notify("Compaction started", "info");
		ctx.compact({
			customInstructions: combineInstructions(configOrDefault(), instructions),
			onComplete: () => {
				compacting = false;
				turnsSinceCompact = 0;
				if (ctx.hasUI) ctx.ui.notify("Compaction completed", "info");
				paintStatus(ctx);
			},
			onError: (error) => {
				compacting = false;
				if (ctx.hasUI) ctx.ui.notify(`Compaction failed: ${error.message}`, "error");
			},
		});
	}

	function layerFor(
		ctx: ExtensionContext,
		cur: NonNullable<ReturnType<typeof writable>>,
		scope: PickScope,
	): { layer: Partial<CompactConfig>; raw: Record<string, unknown>; rel: string } | null {
		if (scope === "agent" || !cur.overlay) {
			return { layer: cur.layer, raw: cur.raw, rel: cur.writeRel };
		}
		const ws = readFileAt(join(ctx.cwd, WORKSPACE_REL), WORKSPACE_REL);
		if (ws.status === "invalid") {
			ctx.ui.notify(ws.error, "error");
			return null;
		}
		if (ws.status === "missing") return { layer: {}, raw: {}, rel: WORKSPACE_REL };
		return { layer: ws.layer, raw: ws.raw, rel: WORKSPACE_REL };
	}

	async function pickModel(
		ctx: ExtensionContext,
		title: string,
		hasPrior: boolean,
	): Promise<string | "back" | null> {
		const models = listModels(ctx);
		if (models.length === 0) {
			ctx.ui.notify("No models available. Sign in a provider first.", "error");
			return null;
		}
		const providers = providersOf(models);
		if (providers.length === 0) return null;
		const providerOpts = hasPrior ? [BACK, ...providers] : providers;
		const provider = await ctx.ui.select(`${title} — provider`, providerOpts);
		if (!provider) return null;
		if (provider === BACK) return "back";
		const ids = idsForProvider(models, provider);
		const idOpts = [BACK, ...ids];
		const id = await ctx.ui.select(`${title} — model`, idOpts);
		if (!id) return null;
		if (id === BACK) return pickModel(ctx, title, hasPrior);
		return `${provider}/${id}`;
	}

	async function pickThinking(ctx: ExtensionContext, current: ThinkingLevel): Promise<ThinkingLevel | "back" | null> {
		const opts = [BACK, ...THINKING_LEVELS.map((t) => (t === current ? `${t} (current)` : t))];
		const choice = await ctx.ui.select("Summarizer thinking", opts);
		if (!choice) return null;
		if (choice === BACK) return "back";
		const t = choice.replace(" (current)", "") as ThinkingLevel;
		return t;
	}

	async function maybeScope(ctx: ExtensionContext, overlay: boolean): Promise<PickScope | "back" | null> {
		if (!overlay) return "workspace";
		const choice = await ctx.ui.select("Save to", [BACK, SCOPE_AGENT, SCOPE_WORKSPACE]);
		if (!choice) return null;
		if (choice === BACK) return "back";
		return choice === SCOPE_AGENT ? "agent" : "workspace";
	}

	function savedNote(overlay: boolean, scope: PickScope): string {
		if (!overlay) return "";
		return scope === "workspace" ? " (workspace)" : " (this agent)";
	}

	async function editFlow(ctx: ExtensionContext) {
		const cur = writable(ctx);
		if (!cur) return;
		let draft: CompactConfig = { ...cur.config };

		while (true) {
			const presetOpts = [
				BACK,
				...WHEN_PRESETS.map((p) => (p.id === whenPresetId(draft) ? `${p.label} (current)` : p.label)),
			];
			const when = await ctx.ui.select("When to compact early", presetOpts);
			if (!when) return;
			if (when === BACK) return;
			const preset = WHEN_PRESETS.find((p) => when.startsWith(p.label));
			if (!preset) return;
			if (preset.id === "custom") {
				const tok = await ctx.ui.input("atTokens (positive int, or none)", String(draft.atTokens ?? "none"));
				if (!tok) return;
				const parsedTok = parseTokenInput(tok, true);
				if (!parsedTok.ok) {
					ctx.ui.notify(parsedTok.error, "error");
					continue;
				}
				const pct = await ctx.ui.input("atPercent (0.5 or 50, or none)", String(draft.atPercent ?? "none"));
				if (!pct) return;
				const parsedPct = parsePercentInput(pct, true);
				if (!parsedPct.ok) {
					ctx.ui.notify(parsedPct.error, "error");
					continue;
				}
				draft = { ...draft, atTokens: parsedTok.value, atPercent: parsedPct.value };
			} else {
				draft = applyWhenPreset(draft, preset.id);
			}

			const model = await pickModel(ctx, "Summarizer", true);
			if (model === "back") continue;
			if (!model) return;
			draft = { ...draft, model };

			const fbChoice = await ctx.ui.select("Fallback summarizer", [BACK, SKIP_FALLBACK, "pick a model"]);
			if (!fbChoice) return;
			if (fbChoice === BACK) continue;
			if (fbChoice === SKIP_FALLBACK) {
				draft = { ...draft, fallback: [] };
			} else {
				const fb = await pickModel(ctx, "Fallback", true);
				if (fb === "back") continue;
				if (!fb) return;
				draft = { ...draft, fallback: fb === model ? [] : [fb] };
			}

			const thinking = await pickThinking(ctx, draft.thinking);
			if (thinking === "back") continue;
			if (!thinking) return;
			draft = { ...draft, thinking };

			const scope = await maybeScope(ctx, cur.overlay);
			if (scope === "back") continue;
			if (!scope) return;

			const target = layerFor(ctx, cur, scope);
			if (!target) return;
			if (!save(ctx, draft, target.raw, target.rel)) return;
			sessionEnabled = draft.enabled;
			ctx.ui.notify(
				`Saved compact policy → ${draft.model ?? "auto"}${savedNote(cur.overlay, scope)}`,
				"info",
			);
			paintStatus(ctx);
			return;
		}
	}

	async function modelFlow(ctx: ExtensionContext) {
		const cur = writable(ctx);
		if (!cur) return;
		let draft: CompactConfig = { ...cur.config };
		while (true) {
			const model = await pickModel(ctx, "Summarizer", false);
			if (model === "back") return;
			if (!model) return;
			draft = { ...draft, model };
			const fbChoice = await ctx.ui.select("Fallback summarizer", [BACK, SKIP_FALLBACK, "pick a model"]);
			if (!fbChoice) return;
			if (fbChoice === BACK) continue;
			if (fbChoice === SKIP_FALLBACK) draft = { ...draft, fallback: [] };
			else {
				const fb = await pickModel(ctx, "Fallback", true);
				if (fb === "back") continue;
				if (!fb) return;
				draft = { ...draft, fallback: fb === model ? [] : [fb] };
			}
			const scope = await maybeScope(ctx, cur.overlay);
			if (scope === "back") continue;
			if (!scope) return;
			const target = layerFor(ctx, cur, scope);
			if (!target) return;
			if (!save(ctx, draft, target.raw, target.rel)) return;
			ctx.ui.notify(
				`Summarizer ${draft.model}${draft.fallback[0] ? ` → ${draft.fallback[0]}` : ""}${savedNote(cur.overlay, scope)}`,
				"info",
			);
			paintStatus(ctx);
			return;
		}
	}

	pi.on("session_start", async (_event, ctx) => {
		loaded = reload(ctx.cwd);
		warnedInvalid = false;
		compacting = false;
		if (loaded.status === "invalid") {
			sessionEnabled = true;
			turnsSinceCompact = DEFAULT_CONFIG.cooldownTurns;
			if (!warnedInvalid) {
				warnedInvalid = true;
				ctx.ui.notify(`pi-compact: ${loaded.error}`, "error");
			}
		} else {
			sessionEnabled = loaded.config.enabled;
			turnsSinceCompact = loaded.config.cooldownTurns;
		}
		paintStatus(ctx);
	});

	pi.on("turn_end", async (_event, ctx) => {
		if (loaded.status !== "invalid") loaded = reload(ctx.cwd);
		turnsSinceCompact += 1;
		paintStatus(ctx);
		if (compacting) return;
		const cfg = configOrDefault();
		const usage = ctx.getContextUsage();
		const decision = shouldTrigger({
			config: cfg,
			sessionEnabled,
			tokens: usage?.tokens ?? null,
			contextWindow: usage?.contextWindow ?? ctx.model?.contextWindow ?? 0,
			turnsSinceCompact,
		});
		if (!decision.trigger) return;
		triggerCompact(ctx);
	});

	pi.on("session_compact", async (_event, ctx) => {
		compacting = false;
		turnsSinceCompact = 0;
		paintStatus(ctx);
	});

	pi.on("session_compact_failed", async (_event, ctx) => {
		compacting = false;
		paintStatus(ctx);
	});

	pi.on("session_before_compact", async (event, ctx) => {
		const cfg = configOrDefault();
		const chosen = pickSummarizer(summarizerCandidates(cfg, sessionModelId(ctx)), (id) => usable(ctx, id));
		if (!chosen) return;
		const parsed = parseModelId(chosen);
		if (!parsed) return;
		const model = ctx.modelRegistry.find(parsed.provider, parsed.id);
		if (!model) return;

		const { preparation, customInstructions, signal } = event;
		const allMessages = [...preparation.messagesToSummarize, ...preparation.turnPrefixMessages];
		if (allMessages.length === 0) return;

		const conversationText = serializeConversation(convertToLlm(allMessages));
		const previousContext = preparation.previousSummary
			? `\n\nPrevious session summary for context:\n${preparation.previousSummary}`
			: "";
		const extra = combineInstructions(cfg, customInstructions);
		const extraBlock = extra ? `\n\nAdditional focus:\n${extra}` : "";

		const summaryMessages = [
			{
				role: "user" as const,
				content: [
					{
						type: "text" as const,
						text: `You are a conversation summarizer. Create a comprehensive summary of this conversation that captures:${previousContext}${extraBlock}

1. The main goals and objectives discussed
2. Key decisions made and their rationale
3. Important code changes, file modifications, or technical details
4. Current state of any ongoing work
5. Any blockers, issues, or open questions
6. Next steps that were planned or suggested

Be thorough but concise. The summary replaces older conversation turns; recent messages are kept separately. Include all information needed to continue the work effectively.

Format the summary as structured markdown with clear sections.

<conversation>
${conversationText}
</conversation>`,
					},
				],
				timestamp: Date.now(),
			},
		];

		if (ctx.hasUI) {
			ctx.ui.notify(`Summarizing with ${chosen} (${cfg.thinking})…`, "info");
		}

		try {
			const response = await ctx.modelRegistry.complete(
				model,
				{ messages: summaryMessages },
				{
					maxTokens: 8192,
					signal,
					cacheRetention: "none",
					sessionId: randomUUID(),
					reasoning: cfg.thinking,
				},
			);
			if (signal.aborted) return;
			const summary = response.content
				.filter((c): c is { type: "text"; text: string } => c.type === "text")
				.map((c) => c.text)
				.join("\n");
			if (!summary.trim()) {
				if (ctx.hasUI) ctx.ui.notify("Summarizer returned empty text; using Pi default", "warning");
				return;
			}
			return {
				compaction: {
					summary,
					firstKeptEntryId: preparation.firstKeptEntryId,
					tokensBefore: preparation.tokensBefore,
					usage: response.usage,
					details: { summarizer: chosen, thinking: cfg.thinking, from: "pi-compact" },
				},
			};
		} catch (error) {
			const message = error instanceof Error ? error.message : String(error);
			if (ctx.hasUI) ctx.ui.notify(`pi-compact summarizer failed (${chosen}): ${message}`, "warning");
			return;
		}
	});

	pi.registerCommand("compact", {
		description: "Compact now. Subcommands: edit, model, on, off",
		handler: async (args, ctx) => {
			const cmd = parseCompactArgs(args ?? "");
			if (cmd.kind === "edit") {
				await editFlow(ctx);
				return;
			}
			if (cmd.kind === "model") {
				await modelFlow(ctx);
				return;
			}
			if (cmd.kind === "on") {
				sessionEnabled = true;
				ctx.ui.notify("Early compact on for this session", "info");
				paintStatus(ctx);
				return;
			}
			if (cmd.kind === "off") {
				sessionEnabled = false;
				ctx.ui.notify("Early compact off for this session (overflow still runs)", "info");
				paintStatus(ctx);
				return;
			}
			await ctx.waitForIdle();
			triggerCompact(ctx, cmd.instructions);
		},
	});
}
