/**
 * Opt-in model roles for pi.
 *
 * Reads <cwd>/.pi/roles.json. Missing file = dormant (no routing).
 * Default is the switch-back target, not a startup override — PiCode
 * already passes --model/--thinking per agent (ADR-0009).
 */
import { readFileSync } from "node:fs";
import { join } from "node:path";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import {
	decideOnInput,
	lockRole,
	parseConfig,
	parseModelId,
	type Assignment,
	type Mode,
	type RolesConfig,
} from "../src/logic.ts";

const CONFIG_REL = join(".pi", "roles.json");
const PLAN_PROMPT =
	"You are in plan mode. Propose a plan. Do not edit files or run mutating commands until the user approves the plan.";

type Loaded =
	| { status: "missing" }
	| { status: "invalid"; error: string }
	| { status: "ok"; config: RolesConfig };

export default function piRoles(pi: ExtensionAPI) {
	let mode: Mode = { kind: "auto" };
	let loaded: Loaded = { status: "missing" };
	let warnedInvalid = false;

	function reload(cwd: string): Loaded {
		const path = join(cwd, CONFIG_REL);
		let text: string;
		try {
			text = readFileSync(path, "utf8");
		} catch (err) {
			const code = (err as NodeJS.ErrnoException).code;
			if (code === "ENOENT") return { status: "missing" };
			return { status: "invalid", error: `Cannot read ${CONFIG_REL}: ${(err as Error).message}` };
		}
		let raw: unknown;
		try {
			raw = JSON.parse(text);
		} catch (err) {
			return { status: "invalid", error: `${CONFIG_REL} is not valid JSON: ${(err as Error).message}` };
		}
		const parsed = parseConfig(raw);
		if (!parsed.ok) return { status: "invalid", error: parsed.error };
		return { status: "ok", config: parsed.config };
	}

	function configOrNull(): RolesConfig | null {
		return loaded.status === "ok" ? loaded.config : null;
	}

	async function apply(ctx: ExtensionContext, target: Assignment, why: string): Promise<void> {
		const parsed = parseModelId(target.model);
		if (!parsed) {
			ctx.ui.notify(`Invalid model "${target.model}"`, "error");
			return;
		}
		const current = ctx.model;
		const sameModel = current?.provider === parsed.provider && current.id === parsed.id;
		const thinkingOk =
			target.thinking === undefined || pi.getThinkingLevel() === target.thinking;
		if (sameModel && thinkingOk) return;

		if (!sameModel) {
			const found = ctx.modelRegistry.find(parsed.provider, parsed.id);
			if (!found) {
				ctx.ui.notify(`Model ${target.model} not found`, "error");
				return;
			}
			const ok = await pi.setModel(found);
			if (!ok) {
				ctx.ui.notify(`No auth for ${target.model}`, "error");
				return;
			}
		}
		if (target.thinking) pi.setThinkingLevel(target.thinking);
		ctx.ui.notify(`${target.model} · ${why}`, "info");
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
		await apply(ctx, d.target, d.why);
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
		description: "Show roles and pick one",
		handler: async (_args, ctx) => {
			await pickRole(ctx);
		},
	});

	async function pickRole(ctx: ExtensionContext) {
		loaded = reload(ctx.cwd);
		if (loaded.status === "missing") {
			ctx.ui.notify("No .pi/roles.json in this workspace", "warning");
			return;
		}
		if (loaded.status === "invalid") {
			ctx.ui.notify(loaded.error, "error");
			return;
		}
		const options: string[] = ["auto"];
		if (loaded.config.builtin.vision) options.push("vision");
		if (loaded.config.builtin.plan) options.push("plan");
		for (const c of loaded.config.custom) options.push(c.name);
		const current =
			mode.kind === "auto" ? "auto" : mode.role;
		const choice = await ctx.ui.select(`Roles (current: ${current})`, options);
		if (!choice || choice === current) return;
		if (choice === "auto") {
			mode = { kind: "auto" };
			ctx.ui.notify("Auto: image → vision, text → default", "info");
			return;
		}
		await lock(ctx, choice);
	}
}
