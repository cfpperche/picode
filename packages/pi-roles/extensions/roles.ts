/**
 * Opt-in model roles for pi.
 *
 * Reads <cwd>/.pi/roles.json. With PI_ROLES_AGENT=<id>, also reads/writes
 * <cwd>/.pi/roles/<id>.json (overlay). Missing files = dormant routing.
 * Default is the switch-back target, not a startup override — PiCode already
 * passes --model/--thinking per agent.
 */
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import {
	addCustom,
	BUILTIN_ROLES,
	decideOnInput,
	emptyConfig,
	lockRole,
	mergeConfigs,
	overlayRel,
	parseAgentKey,
	parseConfig,
	parseModelId,
	pickAnswer,
	pickStart,
	removeCustom,
	serializeConfig,
	upsertRole,
	type Assignment,
	type Mode,
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
	 * Cascading provider → model → thinking selects. Every select that has a
	 * previous field offers an explicit BACK option; cancel (Esc / Cancel)
	 * aborts the whole flow. `hasPrior` adds BACK on the first field so the
	 * caller can return to its own preceding question (role select / name).
	 */
	async function pickAssignment(
		ctx: ExtensionContext,
		title: string,
		hasPrior: boolean,
	): Promise<Assignment | "back" | null> {
		const models = listModels(ctx);
		if (models.length === 0) {
			ctx.ui.notify("No models available. Sign in a provider first.", "error");
			return null;
		}
		let out = pickStart(models, hasPrior);
		while (out.kind === "ask") {
			const label =
				out.state.stage === "thinking" ? "Thinking level" : `${title} — ${out.state.stage}`;
			const choice = await ctx.ui.select(label, out.options);
			if (!choice) return null;
			out = pickAnswer(out.state, choice);
		}
		if (out.kind === "done") return out.assignment;
		if (out.kind === "back") return "back";
		ctx.ui.notify("No models available. Sign in a provider first.", "error");
		return null;
	}

	pi.on("session_start", async (_event, ctx) => {
		loaded = reload(ctx.cwd);
		warnedInvalid = false;
		mode = { kind: "auto" };
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
		const r = await apply(ctx, d.target, d.why);
		// Already on that model: still confirm — a silent lock looks stuck.
		if (r === "noop") ctx.ui.notify(assignmentLine(d.target, d.why), "info");
	}

	pi.registerCommand("auto", {
		description: "Auto: image → vision role, text → default role",
		handler: async (_args, ctx) => {
			mode = { kind: "auto" };
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
				ctx.ui.notify("Auto: image → vision, text → default", "info");
				return;
			}
			await lock(ctx, name);
		},
	});

	pi.registerCommand("roles", {
		description: "Pick a role, or: edit / add / remove",
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
			ctx.ui.notify("Use /roles, /roles edit, /roles add, or /roles remove", "error");
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
		const options: string[] = ["auto"];
		if (loaded.config.builtin.default) options.push("default");
		if (loaded.config.builtin.vision) options.push("vision");
		if (loaded.config.builtin.plan) options.push("plan");
		for (const c of loaded.config.custom) options.push(c.name);
		const current = mode.kind === "auto" ? "auto" : mode.role;
		const choice = await ctx.ui.select(`Roles (current: ${current})`, options);
		if (!choice) return;
		if (choice === "auto") {
			mode = { kind: "auto" };
			ctx.ui.notify("Auto: image → vision, text → default", "info");
			return;
		}
		await lock(ctx, choice);
	}

	function savedNote(overlay: boolean): string {
		return overlay ? " (this agent)" : "";
	}

	async function editFlow(ctx: ExtensionContext, named: string) {
		const cur = writable(ctx);
		if (!cur) return;
		while (true) {
			const role = named || (await ctx.ui.select("Edit which role?", [
				...BUILTIN_ROLES,
				...cur.effective.custom.map((c) => c.name),
			]));
			if (!role) return;
			const assignment = await pickAssignment(ctx, `Model for ${role}`, !named);
			if (assignment === "back") {
				if (named) return;
				continue; // back to the role select
			}
			if (!assignment) return; // cancel aborts the whole flow
			const result = upsertRole(cur.config, role, assignment);
			if (!result.ok) {
				ctx.ui.notify(result.error, "error");
				return;
			}
			if (!save(ctx, result.config, cur.raw, cur.writeRel)) return;
			ctx.ui.notify(`Saved ${role} → ${assignment.model}${savedNote(cur.overlay)}`, "info");
			if (mode.kind === "lock" && mode.role === role) {
				await apply(ctx, assignment, `lock /${role}`);
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
			const assignment = await pickAssignment(ctx, `Model for ${trimmed}`, !named);
			if (assignment === "back") {
				if (named) return;
				continue; // back to the name input
			}
			if (!assignment) return; // cancel aborts the whole flow
			const result = addCustom(cur.config, trimmed, assignment);
			if (!result.ok) {
				ctx.ui.notify(result.error, "error");
				return;
			}
			if (!save(ctx, result.config, cur.raw, cur.writeRel)) return;
			ctx.ui.notify(`Added ${trimmed} → ${assignment.model}${savedNote(cur.overlay)}`, "info");
			return;
		}
	}

	async function removeFlow(ctx: ExtensionContext, named: string) {
		const cur = writable(ctx);
		if (!cur) return;
		if (cur.config.custom.length === 0) {
			ctx.ui.notify(
				cur.overlay && cur.effective.custom.length > 0
					? "No custom roles on this agent. Workspace presets stay in .pi/roles.json."
					: "No custom roles.",
				"warning",
			);
			return;
		}
		const name =
			named || (await ctx.ui.select("Remove which preset?", cur.config.custom.map((c) => c.name)));
		if (!name) return;
		if (cur.overlay && !cur.config.custom.some((c) => c.name === name)) {
			ctx.ui.notify(`"${name}" is in .pi/roles.json (workspace). This agent can /roles edit it.`, "error");
			return;
		}
		const ok = await ctx.ui.confirm("Remove this preset?", name);
		if (!ok) return;
		const result = removeCustom(cur.config, name);
		if (!result.ok) {
			ctx.ui.notify(result.error, "error");
			return;
		}
		if (!save(ctx, result.config, cur.raw, cur.writeRel)) return;
		ctx.ui.notify(`Removed ${name}${savedNote(cur.overlay)}`, "info");
		if (mode.kind === "lock" && mode.role === name) {
			mode = { kind: "auto" };
			ctx.ui.notify("Auto: image → vision, text → default", "info");
		}
	}
}

function unique(items: string[]): string[] {
	return [...new Set(items)];
}
