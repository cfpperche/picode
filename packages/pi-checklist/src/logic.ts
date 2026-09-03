/**
 * Pure logic for pi-checklist (ADR-0055). No pi imports — node:test
 * covers this. The extension file (extensions/checklist.ts) is I/O glue:
 * it registers the tool, the gate and the reminder with pi, and POSTs
 * what these functions build to PiCode.
 */

export const STATUSES = ["pending", "in-progress", "completed"] as const;
export type Status = (typeof STATUSES)[number];

export interface Item {
	readonly text: string;
	readonly status: Status;
}

/** The obligation level PiCode passes per agent (PICODE_CHECKLIST). */
export const LEVELS = ["changes", "always", "never"] as const;
export type Level = (typeof LEVELS)[number];

export const MAX_ITEMS = 30;
export const MAX_TEXT = 200;

/**
 * `changes` (the default, also for a plain pi outside PiCode): a checklist
 * is required before the first change of a task; a read-only answer needs
 * none. `always`: every task, changes or not. `never`: the tool stays
 * available, nothing is required.
 */
export function parseLevel(env: Record<string, string | undefined>): Level {
	const raw = (env.PICODE_CHECKLIST || "").trim().toLowerCase();
	return (LEVELS as readonly string[]).includes(raw) ? (raw as Level) : "changes";
}

/**
 * The tools that change the world. A closed allowlist, on purpose: a tool
 * this set does not know passes, so an unknown tool can only widen the
 * door, never turn a read into a refusal (Tachyon's lock 1).
 */
export const MUTATING_TOOLS: ReadonlySet<string> = new Set(["bash", "powershell", "edit", "write", "multiedit"]);

export const REFUSAL =
	"[pi-checklist] A checklist is required before the first change of this task. " +
	"Call the checklist tool with 2–8 concrete steps (the first one in-progress), then retry this call.";

export const REMINDER =
	"[pi-checklist] This task requires a checklist and the turn ended without one. " +
	"Call the checklist tool with your steps, then continue.";

/** How many reminders `always` sends for one task before it stops nagging. */
export const MAX_REMINDERS = 3;

/**
 * The gate (Tachyon's lock 2 and 3, per task rather than per session):
 * with a level that requires a plan, a mutating tool call before this
 * task's checklist is refused. `planned` is whether the checklist tool
 * ran since the task started. Never `terminate`: the model must be able
 * to write the checklist and retry.
 */
export function decideGate(input: { level: Level; toolName: string; planned: boolean }): { block: true; reason: string } | { block: false } {
	if (input.level === "never") return { block: false };
	if (input.planned) return { block: false };
	if (!MUTATING_TOOLS.has(input.toolName)) return { block: false };
	return { block: true, reason: REFUSAL };
}

/** Whether a turn that ended without a checklist earns a reminder. */
export function decideReminder(input: { level: Level; planned: boolean; sent: number }): boolean {
	if (input.level !== "always" || input.planned) return false;
	return input.sent < MAX_REMINDERS;
}

/** The contract the model reads, appended to the system prompt. */
export function contractPrompt(level: Level): string {
	if (level === "never") return "";
	const when =
		level === "always"
			? "Every task, before anything else,"
			: "Before your first change of a task (edit, write, bash),";
	return (
		"# Checklist (pi-checklist)\n\n" +
		`${when} write your plan with the \`checklist\` tool: 2–8 concrete steps, the first one \`in-progress\`. ` +
		"Send the whole list on every call; mark a step `completed` when it is done and the next one `in-progress` in the same call, " +
		"so the human can follow your work from the sidebar without opening this session. " +
		"Add steps you discover; never delete history. " +
		(level === "always"
			? "A question or a read-only answer still gets a short checklist."
			: "A question or a read-only answer needs no checklist. ") +
		"Changes are refused until the checklist exists."
	);
}

/** Validate and normalize the tool's `items`; throws a message the model can act on. */
export function normalizeItems(raw: unknown): Item[] {
	if (!Array.isArray(raw)) throw new Error("items must be an array of {text, status}");
	if (raw.length === 0) throw new Error("items must hold at least one step");
	if (raw.length > MAX_ITEMS) throw new Error(`items must hold at most ${MAX_ITEMS} steps`);
	return raw.map((it, i) => {
		const o = (it && typeof it === "object" ? it : {}) as { text?: unknown; status?: unknown };
		const text = typeof o.text === "string" ? o.text.replace(/\s+/g, " ").trim() : "";
		if (!text) throw new Error(`items[${i}].text is required`);
		const status = o.status === undefined ? "pending" : o.status;
		if (!(STATUSES as readonly unknown[]).includes(status)) throw new Error(`items[${i}].status must be one of ${STATUSES.join(", ")}`);
		return { text: text.slice(0, MAX_TEXT), status: status as Status };
	});
}

export function countDone(items: readonly Item[]): number {
	return items.filter((it) => it.status === "completed").length;
}

/** The step the human sees: the in-progress one, else the first pending, else the last when all are done. */
export function currentStep(items: readonly Item[]): { text: string; position: number; total: number } | undefined {
	if (items.length === 0) return undefined;
	let i = items.findIndex((it) => it.status === "in-progress");
	if (i < 0) i = items.findIndex((it) => it.status === "pending");
	if (i < 0) i = items.length - 1;
	return { text: items[i]!.text, position: i + 1, total: items.length };
}

/** What the model reads back after a call. */
export function summarize(items: readonly Item[]): string {
	const step = currentStep(items);
	const done = countDone(items);
	const head = `Checklist saved: ${done}/${items.length} completed.`;
	if (!step || done === items.length) return head + " All steps completed.";
	return `${head} Current step (${step.position}/${step.total}): ${step.text}`;
}

export const GLYPH: Record<Status, string> = { pending: "☐", "in-progress": "◐", completed: "☑" };

/** The list as the TUI prints it, one line per step. */
export function renderLines(items: readonly Item[]): string[] {
	return items.map((it) => `${GLYPH[it.status]} ${it.text}`);
}

export interface Payload {
	sessionId?: string;
	items: Item[];
	absent?: boolean;
	blocked?: boolean;
}

/** The body PiCode receives; the agent id travels in the URL. */
export function buildPayload(items: readonly Item[], opts: { sessionId?: string; absent?: boolean; blocked?: boolean } = {}): Payload {
	const out: Payload = { items: items.map((it) => ({ text: it.text, status: it.status })) };
	if (opts.sessionId) out.sessionId = opts.sessionId;
	if (opts.absent) out.absent = true;
	if (opts.blocked) out.blocked = true;
	return out;
}

/** The agent this pi runs as, from the env PiCode stamps on every spawn; "" for a raw pi. */
export function agentId(env: Record<string, string | undefined>): string {
	const id = (env.PICODE_AGENT_ID || "").trim();
	return /^[A-Za-z0-9_.-]{1,128}$/.test(id) ? id : "";
}

/**
 * Rebuild the list from the session branch: the last toolResult of the
 * checklist tool carries `details.items` (pi's branching-safe state idiom).
 */
export function reconstruct(entries: readonly unknown[]): Item[] | null {
	let found: Item[] | null = null;
	for (const entry of entries) {
		const e = entry as { type?: string; message?: { role?: string; toolName?: string; details?: { items?: unknown } } };
		if (e.type !== "message" || !e.message || e.message.role !== "toolResult" || e.message.toolName !== "checklist") continue;
		try {
			found = normalizeItems(e.message.details?.items);
		} catch {
			/* a malformed row is not the whole history */
		}
	}
	return found;
}

// ---- PiCode reachability (the pi-inbox idiom, copied so this package has no deps) ----

export function resolveDataDir(env: Record<string, string | undefined>, homedir: string): string {
	const explicit = (env.PICODE_DATA || "").trim();
	if (explicit) return explicit;
	return homedir.replace(/\/+$/, "") + "/.picode";
}

export type ServerInfo = { ok: true; url: string } | { ok: false; error: string };

export function parseServerJson(text: string): ServerInfo {
	let data: unknown;
	try {
		data = JSON.parse(text);
	} catch {
		return { ok: false, error: "server.json is not valid JSON" };
	}
	const url = (data as { url?: unknown })?.url;
	if (typeof url !== "string" || !/^https?:\/\//.test(url)) return { ok: false, error: "server.json has no usable url" };
	return { ok: true, url: url.replace(/\/+$/, "") };
}

export function parseToken(text: string | null | undefined): string {
	const t = (text || "").trim();
	return /^[0-9a-f]{32,128}$/i.test(t) ? t : "";
}

export function resolveServerUrl(env: Record<string, string | undefined>, serverJson: string | null): ServerInfo {
	const explicit = (env.PICODE_URL || "").trim();
	if (explicit) {
		if (!/^https?:\/\/[^\s/]+\/?$/.test(explicit)) return { ok: false, error: "PICODE_URL must be an origin like https://box:8445" };
		return { ok: true, url: explicit.replace(/\/+$/, "") };
	}
	if (serverJson === null) return { ok: false, error: "no server.json" };
	return parseServerJson(serverJson);
}

export function resolveToken(env: Record<string, string | undefined>, fileText: string | null | undefined): string {
	return parseToken(env.PICODE_TOKEN) || parseToken(fileText);
}

export function rejectUnauthorizedFor(url: string): boolean {
	try {
		const h = new URL(url).hostname.replace(/^\[|\]$/g, "");
		return !(h === "localhost" || h === "127.0.0.1" || h === "::1");
	} catch {
		return true;
	}
}
