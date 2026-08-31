/**
 * Opt-in model roles for pi.
 *
 * Reads <cwd>/.pi/roles.json. With PI_ROLES_AGENT=<id>, also reads/writes
 * <cwd>/.pi/roles/<id>.json (overlay). Missing files = dormant routing.
 * Default is the switch-back target, not a startup override — PiCode already
 * passes --model/--thinking per agent.
 */
import { mkdirSync, readFileSync, unlinkSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import {
	addCustom,
	BACK,
	BUILTIN_ROLES,
	decideOnInput,
	emptyConfig,
	lockRole,
	mergeConfigs,
	overlayRel,
	parseAgentKey,
	parseConfig,
	parseModelId,
	parseState,
	pickAnswer,
	pickStart,
	removeCustom,
	removeScopes,
	resolveRole,
	roleEntries,
	roleFromChoice,
	roleOption,
	SCOPE_AGENT,
	SCOPE_WORKSPACE,
	stateJson,
	serializeConfig,
	upsertRole,
	type Assignment,
	type Mode,
	type PickScope,
	type RolesConfig,
} from "../src/logic.ts";

const WORKSPACE_REL = join(".pi", "roles.json");
const PLAN_PROMPT =
	"You are in plan mode. Propose a plan. Do not edit files or run mutating commands until the user approves the plan.";

type FileRead =
	| { status: "missing" }
	| { status: "invalid"; error: string }
	| { status: "ok"; config: RolesConfig; raw: Record<string, unknown> };

type Loaded =
	| { status: "missing"; writeRel: string; overlay: boolean }
	| { status: "invalid"; error: string }
	| {
			status: "ok";
			config: RolesConfig;
			layer: RolesConfig;
			raw: Record<string, unknown>;
			writeRel: string;
			overlay: boolean;
	  };

export default function piRoles(pi: ExtensionAPI) {
	let mode: Mode = { kind: "auto" };
	let loaded: Loaded = { status: "missing", writeRel: WORKSPACE_REL, overlay: false };
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
		const rawObj =
			raw !== null && typeof raw === "object" && !Array.isArray(raw)
				? (raw as Record<string, unknown>)
				: {};
		return { status: "ok", config: parsed.config, raw: rawObj };
	}

	function writeTarget(): { rel: string; overlay: boolean } {
		const key = parseAgentKey(process.env.PI_ROLES_AGENT);
		if (key) return { rel: overlayRel(key), overlay: true };
		return { rel: WORKSPACE_REL, overlay: false };
	}

	/** ~/.pi/agent/roles-state/<agent>.json, or "" without PI_ROLES_AGENT. */
	function statePath(): string {
		const key = parseAgentKey(process.env.PI_ROLES_AGENT);
		if (!key) return "";
		return join(homedir(), ".pi", "agent", "roles-state", `${key}.json`);
	}

	/**
	 * Publish the active-role state (ADR-0033 amendment #2) — the contract
	 * PiCode's composer chip reads. Called on every mode change and whenever
	 * the effective role list changes; never on per-input routing.
	 */
	function publishState() {
		const path = statePath();
		if (!path) return;
		try {
			mkdirSync(dirname(path), { recursive: true });
			writeFileSync(path, JSON.stringify(stateJson(mode, configOrNull()), null, 2) + "\n", "utf8");
		} catch { /* state is best-effort; routing must not fail on it */ }
	}

	/** Restore a locked mode from the state file (survives restarts). */
	function restoreState() {
		const path = statePath();
		if (!path) return;
		let raw: unknown;
		try {
			raw = JSON.parse(readFileSync(path, "utf8"));
		} catch {
			return;
		}
		const st = parseState(raw);
		if (st && st.mode === "lock" && st.role && resolveRole(configOrNull() ?? emptyConfig(), st.role)) {
			mode = { kind: "lock", role: st.role };
		}
	}

	function reload(cwd: string): Loaded {
		const target = writeTarget();
		const ws = readFileAt(join(cwd, WORKSPACE_REL), WORKSPACE_REL);
		if (ws.status === "invalid") return ws;
		if (!target.overlay) {
			if (ws.status === "missing") return { status: "missing", writeRel: target.rel, overlay: false };
			return {
				status: "ok",
				config: ws.config,
				layer: ws.config,
				raw: ws.raw,
				writeRel: target.rel,
				overlay: false,
			};
		}
		const ov = readFileAt(join(cwd, target.rel), target.rel);
		if (ov.status === "invalid") return ov;
		if (ws.status === "missing" && ov.status === "missing") {
			return { status: "missing", writeRel: target.rel, overlay: true };
		}
		const base = ws.status === "ok" ? ws.config : emptyConfig();
		const over = ov.status === "ok" ? ov.config : emptyConfig();
		const raw = ov.status === "ok" ? ov.raw : {};
		return {
			status: "ok",
			config: mergeConfigs(base, over),
			layer: over,
			raw,
			writeRel: target.rel,
			overlay: true,
		};
	}

	function configOrNull(): RolesConfig | null {
		return loaded.status === "ok" ? loaded.config : null;
	}

	function writable(
		ctx: ExtensionContext,
	): {
		config: RolesConfig;
		raw: Record<string, unknown>;
		writeRel: string;
		overlay: boolean;
		effective: RolesConfig;
	} | null {
		loaded = reload(ctx.cwd);
		if (loaded.status === "invalid") {
			ctx.ui.notify(loaded.error, "error");
			return null;
		}
		if (loaded.status === "missing") {
			return {
				config: emptyConfig(),
				raw: {},
				writeRel: loaded.writeRel,
				overlay: loaded.overlay,
				effective: emptyConfig(),
			};
		}
		return {
			config: loaded.layer,
			raw: loaded.raw,
			writeRel: loaded.writeRel,
			overlay: loaded.overlay,
			effective: loaded.config,
		};
	}

	function save(
		ctx: ExtensionContext,
		config: RolesConfig,
		raw: Record<string, unknown>,
		writeRel: string,
	): boolean {
		const path = join(ctx.cwd, writeRel);
		try {
			mkdirSync(dirname(path), { recursive: true });
			writeFileSync(path, JSON.stringify(serializeConfig(config, raw), null, 2) + "\n", "utf8");
		} catch (err) {
			ctx.ui.notify(`Could not write ${writeRel}: ${(err as Error).message}`, "error");
			return false;
		}
		loaded = reload(ctx.cwd);
		publishState();
		return loaded.status === "ok";
	}

	function assignmentLine(target: Assignment, why: string): string {
		const t = target.thinking ? ` · ${target.thinking}` : "";
		return `${target.model}${t} · ${why}`;
	}

	/** Returns "applied" | "noop" | "failed". Notifies on change and on failure. */
	async function apply(
		ctx: ExtensionContext,
		target: Assignment,
		why: string,
	): Promise<"applied" | "noop" | "failed"> {
		const parsed = parseModelId(target.model);
		if (!parsed) {
			ctx.ui.notify(`Invalid model "${target.model}"`, "error");
			return "failed";
		}
		const current = ctx.model;
		const sameModel = current?.provider === parsed.provider && current.id === parsed.id;
		const thinkingOk =
			target.thinking === undefined || pi.getThinkingLevel() === target.thinking;
		if (sameModel && thinkingOk) return "noop";

		if (!sameModel) {
			const found = ctx.modelRegistry.find(parsed.provider, parsed.id);
			if (!found) {
				ctx.ui.notify(`Model ${target.model} not found`, "error");
				return "failed";
			}
			const ok = await pi.setModel(found);
			if (!ok) {
				ctx.ui.notify(`No auth for ${target.model}`, "error");
				return "failed";
			}
		}
		if (target.thinking) pi.setThinkingLevel(target.thinking);
		ctx.ui.notify(assignmentLine(target, why), "info");
		return "applied";
	}

	async function applyDecision(
		ctx: ExtensionContext,
		d: ReturnType<typeof decideOnInput>,
	): Promise<void> {
		if (d.kind === "noop") return;
		if (d.kind === "error") {
			ctx.ui.notify(d.message, "error");
			return;
		}
		await apply(ctx, d.target, d.why);
	}

	function listModels(ctx: ExtensionContext): string[] {
		const scoped = ctx.scopedModels;
		if (Array.isArray(scoped) && scoped.length > 0) {
			return unique(scoped.map((e) => `${e.model.provider}/${e.model.id}`));
		}
		const all = ctx.modelRegistry.getAvailable();
		return unique(all.map((m) => `${m.provider}/${m.id}`));
	}

	/**
	 * Cascading provider → model → thinking (→ save-to) selects. Every select
	 * that has a previous field offers an explicit BACK option; cancel
	 * (Esc / Cancel) aborts the whole flow. `hasPrior` adds BACK on the first
	 * field so the caller can return to its own preceding question (role
	 * select / name). `askScope` appends the "Save to" select — offered when
	 * the process has an agent overlay (PI_ROLES_AGENT).
	 */
	async function pickAssignment(
		ctx: ExtensionContext,
		title: string,
		hasPrior: boolean,
		askScope: boolean,
	): Promise<{ assignment: Assignment; scope?: PickScope } | "back" | null> {
		const models = listModels(ctx);
		if (models.length === 0) {
			ctx.ui.notify("No models available. Sign in a provider first.", "error");
			return null;
		}
		let out = pickStart(models, { hasPrior, askScope });
		while (out.kind === "ask") {
			const label =
				out.state.stage === "thinking" ? "Thinking level"
				: out.state.stage === "scope" ? "Save to"
				: `${title} — ${out.state.stage}`;
			const choice = await ctx.ui.select(label, out.options);
			if (!choice) return null;
			out = pickAnswer(out.state, choice);
		}
		if (out.kind === "done") return { assignment: out.assignment, scope: out.scope };
		if (out.kind === "back") return "back";
		ctx.ui.notify("No models available. Sign in a provider first.", "error");
		return null;
	}

	/** The config layer + file a save should land on for the chosen scope. */
	function layerFor(
		ctx: ExtensionContext,
		cur: NonNullable<ReturnType<typeof writable>>,
		scope: PickScope,
	): { config: RolesConfig; raw: Record<string, unknown>; rel: string } | null {
		if (scope === "agent" || !cur.overlay) {
			return { config: cur.config, raw: cur.raw, rel: cur.writeRel };
		}
		const ws = readFileAt(join(ctx.cwd, WORKSPACE_REL), WORKSPACE_REL);
		if (ws.status === "invalid") {
			ctx.ui.notify(ws.error, "error");
			return null;
		}
		if (ws.status === "missing") return { config: emptyConfig(), raw: {}, rel: WORKSPACE_REL };
		return { config: ws.config, raw: ws.raw, rel: WORKSPACE_REL };
	}

	pi.on("session_start", async (_event, ctx) => {
		loaded = reload(ctx.cwd);
		warnedInvalid = false;
		mode = { kind: "auto" };
		// A lock outlives the process: restore it when its role still
		// resolves — the next input applies the model (never at startup,
		// which would fight --model / ADR-0009).
		restoreState();
		publishState();
		if (loaded.status === "invalid" && !warnedInvalid) {
			warnedInvalid = true;
			ctx.ui.notify(`pi-roles: ${loaded.error}`, "error");
		}
	});

	pi.on("input", async (event, ctx) => {
		if (event.source === "extension") return { action: "continue" as const };
		loaded = reload(ctx.cwd);
		const d = decideOnInput({
			config: configOrNull(),
			mode,
			text: event.text ?? "",
			images: event.images,
			source: event.source,
		});
		await applyDecision(ctx, d);
		return { action: "continue" as const };
	});

	pi.on("before_agent_start", async (event) => {
		if (mode.kind === "lock" && mode.role === "plan") {
			return { systemPrompt: `${event.systemPrompt}\n\n${PLAN_PROMPT}` };
		}
		return;
	});

	async function lock(ctx: ExtensionContext, role: string) {
		loaded = reload(ctx.cwd);
		const d = lockRole(configOrNull(), role);
		if (d.kind === "error") {
			ctx.ui.notify(d.message, "error");
			return;
		}
		mode = { kind: "lock", role };
		publishState();
		const r = await apply(ctx, d.target, d.why);
		// Already on that model: still confirm — a silent lock looks stuck.
		if (r === "noop") ctx.ui.notify(assignmentLine(d.target, d.why), "info");
	}

	pi.registerCommand("auto", {
		description: "Auto: image → vision role, text → default role",
		handler: async (_args, ctx) => {
			mode = { kind: "auto" };
			publishState();
			ctx.ui.notify("Auto: image → vision, text → default", "info");
		},
	});

	pi.registerCommand("vision", {
		description: "Lock the vision role until /auto",
		handler: async (_args, ctx) => {
			await lock(ctx, "vision");
		},
	});

	pi.registerCommand("plan", {
		description: "Lock the plan role (and inject plan-mode instructions) until /auto",
		handler: async (_args, ctx) => {
			await lock(ctx, "plan");
		},
	});

	pi.registerCommand("role", {
		description: "Lock a named preset from .pi/roles.json. No args: pick one.",
		handler: async (args, ctx) => {
			const name = (args ?? "").trim();
			if (!name) {
				await pickRole(ctx);
				return;
			}
			if (name === "auto") {
				mode = { kind: "auto" };
				publishState();
				ctx.ui.notify("Auto: image → vision, text → default", "info");
				return;
			}
			await lock(ctx, name);
		},
	});

	pi.registerCommand("roles", {
		description: "Pick a role, or: edit / add / remove / clear",
		handler: async (args, ctx) => {
			const trimmed = (args ?? "").trim();
			if (!trimmed) {
				await pickRole(ctx);
				return;
			}
			const space = trimmed.indexOf(" ");
			const verb = (space < 0 ? trimmed : trimmed.slice(0, space)).toLowerCase();
			const rest = space < 0 ? "" : trimmed.slice(space + 1).trim();
			if (verb === "edit") {
				await editFlow(ctx, rest);
				return;
			}
			if (verb === "add") {
				await addFlow(ctx, rest);
				return;
			}
			if (verb === "remove") {
				await removeFlow(ctx, rest);
				return;
			}
			if (verb === "clear") {
				await clearFlow(ctx, rest);
				return;
			}
			if (verb === "auto") {
				// Alias for /auto — people type it here first.
				mode = { kind: "auto" };
				publishState();
				ctx.ui.notify("Auto: image → vision, text → default", "info");
				return;
			}
			ctx.ui.notify("Use /roles, /roles edit, /roles add, /roles remove, or /roles clear", "error");
		},
	});

	async function pickRole(ctx: ExtensionContext) {
		loaded = reload(ctx.cwd);
		if (loaded.status === "missing") {
			ctx.ui.notify("No roles yet. /roles add creates one.", "warning");
			return;
		}
		if (loaded.status === "invalid") {
			ctx.ui.notify(loaded.error, "error");
			return;
		}
		const options: string[] = [roleOption("auto")];
		for (const e of roleEntries(loaded.config)) options.push(roleOption(e.name, e.assignment));
		const current = mode.kind === "auto" ? "auto" : mode.role;
		const rawChoice = await ctx.ui.select(`Roles (current: ${current})`, options);
		if (!rawChoice) return;
		const choice = roleFromChoice(rawChoice);
		if (choice === "auto") {
			mode = { kind: "auto" };
			ctx.ui.notify("Auto: image → vision, text → default", "info");
			return;
		}
		await lock(ctx, choice);
	}

	function savedNote(overlay: boolean, scope: PickScope): string {
		if (!overlay) return "";
		return scope === "workspace" ? " (workspace)" : " (this agent)";
	}

	async function editFlow(ctx: ExtensionContext, named: string) {
		const cur = writable(ctx);
		if (!cur) return;
		while (true) {
			const roleChoice = named || (await ctx.ui.select("Edit which role?", [
				...BUILTIN_ROLES.map((name) => roleOption(name, cur.effective.builtin[name])),
				...cur.effective.custom.map((c) =>
					roleOption(c.name, { model: c.model, ...(c.thinking ? { thinking: c.thinking } : {}) })),
			]));
			if (!roleChoice) return;
			const role = roleFromChoice(roleChoice);
			const picked = await pickAssignment(ctx, `Model for ${role}`, !named, cur.overlay);
			if (picked === "back") {
				if (named) return;
				continue; // back to the role select
			}
			if (!picked) return; // cancel aborts the whole flow
			const scope: PickScope = picked.scope ?? (cur.overlay ? "agent" : "workspace");
			const target = layerFor(ctx, cur, scope);
			if (!target) return;
			const result = upsertRole(target.config, role, picked.assignment);
			if (!result.ok) {
				ctx.ui.notify(result.error, "error");
				return;
			}
			if (!save(ctx, result.config, target.raw, target.rel)) return;
			ctx.ui.notify(
				`Saved ${role} → ${picked.assignment.model}${savedNote(cur.overlay, scope)}`,
				"info",
			);
			if (mode.kind === "lock" && mode.role === role) {
				await apply(ctx, picked.assignment, `lock /${role}`);
			}
			return;
		}
	}

	async function addFlow(ctx: ExtensionContext, named: string) {
		const cur = writable(ctx);
		if (!cur) return;
		while (true) {
			const name = named || (await ctx.ui.input("Preset name", "fast"));
			if (!name) return;
			const trimmed = name.trim();
			if (cur.effective.custom.some((c) => c.name === trimmed)) {
				ctx.ui.notify(`"${trimmed}" already exists. Use /roles edit.`, "error");
				return;
			}
			const picked = await pickAssignment(ctx, `Model for ${trimmed}`, !named, cur.overlay);
			if (picked === "back") {
				if (named) return;
				continue; // back to the name input
			}
			if (!picked) return; // cancel aborts the whole flow
			const scope: PickScope = picked.scope ?? (cur.overlay ? "agent" : "workspace");
			const target = layerFor(ctx, cur, scope);
			if (!target) return;
			const result = addCustom(target.config, trimmed, picked.assignment);
			if (!result.ok) {
				ctx.ui.notify(result.error, "error");
				return;
			}
			if (!save(ctx, result.config, target.raw, target.rel)) return;
			ctx.ui.notify(
				`Added ${trimmed} → ${picked.assignment.model}${savedNote(cur.overlay, scope)}`,
				"info",
			);
			return;
		}
	}

	async function removeFlow(ctx: ExtensionContext, named: string) {
		const cur = writable(ctx);
		if (!cur) return;
		if (cur.effective.custom.length === 0) {
			ctx.ui.notify("No custom presets yet — /roles add creates one.", "warning");
			return;
		}
		const ws = readFileAt(join(ctx.cwd, WORKSPACE_REL), WORKSPACE_REL);
		if (ws.status === "invalid") {
			ctx.ui.notify(ws.error, "error");
			return;
		}
		const wsConfig = ws.status === "ok" ? ws.config : emptyConfig();
		const overlayConfig = cur.overlay ? cur.config : emptyConfig();
		while (true) {
			const choice = named || (await ctx.ui.select(
				"Remove which preset?",
				cur.effective.custom.map((c) =>
					roleOption(c.name, { model: c.model, ...(c.thinking ? { thinking: c.thinking } : {}) })),
			));
			if (!choice) return;
			const name = roleFromChoice(choice);
			const scopes = removeScopes(wsConfig, overlayConfig, name);
			if (scopes.length === 0) {
				ctx.ui.notify(`Role "${name}" is not configured — /roles edit ${name} creates it.`, "error");
				return;
			}
			// One layer holds it: no question. Both: ask which copy goes.
			let scope: PickScope = scopes[0];
			if (scopes.length > 1) {
				const from = await ctx.ui.select("Remove from", [SCOPE_AGENT, SCOPE_WORKSPACE, BACK]);
				if (!from) return;
				if (from === BACK) {
					if (named) return;
					continue;
				}
				scope = from === SCOPE_AGENT ? "agent" : "workspace";
			}
			const target = layerFor(ctx, cur, scope);
			if (!target) return;
			const ok = await ctx.ui.confirm(
				"Remove this preset?",
				`${name}${savedNote(cur.overlay, scope)}`,
			);
			if (!ok) return;
			const result = removeCustom(target.config, name);
			if (!result.ok) {
				ctx.ui.notify(result.error, "error");
				return;
			}
			if (!save(ctx, result.config, target.raw, target.rel)) return;
			ctx.ui.notify(`Removed ${name}${savedNote(cur.overlay, scope)}`, "info");
			if (mode.kind === "lock" && mode.role === name
				&& !resolveRole(configOrNull() ?? emptyConfig(), name)) {
				mode = { kind: "auto" };
				publishState();
				ctx.ui.notify("Auto: image → vision, text → default", "info");
			}
			return;
		}
	}

	/** /roles clear [agent|workspace] — delete a whole roles file. */
	async function clearFlow(ctx: ExtensionContext, named: string) {
		const target = writeTarget();
		let scope = named.trim().toLowerCase();
		if (scope && scope !== "agent" && scope !== "workspace") {
			ctx.ui.notify("Use /roles clear, /roles clear agent, or /roles clear workspace", "error");
			return;
		}
		if (scope === "agent" && !target.overlay) {
			ctx.ui.notify("No agent overlay here — this pi has no PI_ROLES_AGENT.", "error");
			return;
		}
		if (!scope) {
			if (target.overlay) {
				const c = await ctx.ui.select("Clear which config?", [SCOPE_AGENT, SCOPE_WORKSPACE]);
				if (!c) return;
				scope = c === SCOPE_AGENT ? "agent" : "workspace";
			} else {
				scope = "workspace";
			}
		}
		const rel = scope === "agent" ? target.rel : WORKSPACE_REL;
		const read = readFileAt(join(ctx.cwd, rel), rel);
		if (read.status === "missing") {
			ctx.ui.notify(`Nothing to clear — ${rel} does not exist.`, "warning");
			return;
		}
		const ok = await ctx.ui.confirm("Delete this roles file?", rel);
		if (!ok) {
			// A "No" is a decision too — without this the chat line degrades
			// to the raw answers ("this agent · No").
			ctx.ui.notify(`Kept ${rel}`, "info");
			return;
		}
		try {
			unlinkSync(join(ctx.cwd, rel));
		} catch (err) {
			ctx.ui.notify(`Could not delete ${rel}: ${(err as Error).message}`, "error");
			return;
		}
		loaded = reload(ctx.cwd);
		ctx.ui.notify(`Cleared ${rel}`, "info");
		// A lock whose role no longer resolves would error on every input.
		if (mode.kind === "lock" && !resolveRole(configOrNull() ?? emptyConfig(), mode.role)) {
			mode = { kind: "auto" };
			ctx.ui.notify("Auto: image → vision, text → default", "info");
		}
		publishState();
	}
}

function unique(items: string[]): string[] {
	return [...new Set(items)];
}
