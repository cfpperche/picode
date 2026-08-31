/**
 * Pure logic for pi-inbox. No pi imports — node:test covers this.
 *
 * The extension file (extensions/inbox.ts) is I/O glue: it reads
 * server.json, POSTs what these functions build, and registers the
 * tools with pi.
 */

export const MAX_TITLE = 200;
export const MAX_BODY = 100_000;

/** Where PiCode's data dir lives: PICODE_DATA beats ~/.picode. */
export function resolveDataDir(env: Record<string, string | undefined>, homedir: string): string {
	const explicit = (env.PICODE_DATA || "").trim();
	if (explicit) return explicit;
	return homedir.replace(/\/+$/, "") + "/.picode";
}

export type ServerInfo = { ok: true; url: string } | { ok: false; error: string };

/**
 * Parse server.json ({url, scheme, host, port, pid, time}). The file is
 * re-read per tool call — the port can rebind while a pi process lives.
 */
export function parseServerJson(text: string): ServerInfo {
	let data: unknown;
	try {
		data = JSON.parse(text);
	} catch {
		return { ok: false, error: "server.json is not valid JSON" };
	}
	const url = (data as { url?: unknown })?.url;
	if (typeof url !== "string" || !/^https?:\/\//.test(url)) {
		return { ok: false, error: "server.json has no usable url" };
	}
	return { ok: true, url: url.replace(/\/+$/, "") };
}

/** The identity PiCode stamps on managed and TUI spawns (ADR-0037). */
export function agentIdentity(env: Record<string, string | undefined>): {
	sourceKind: "agent" | "system";
	sourceId: string;
} {
	const id = (env.PICODE_AGENT_ID || "").trim();
	if (id) return { sourceKind: "agent", sourceId: id };
	// A raw terminal `pi` has no agent identity; items still carry honest
	// provenance so the human knows where they came from.
	return { sourceKind: "system", sourceId: "pi (unmanaged)" };
}

function clip(s: string, max: number): string {
	return s.length > max ? s.slice(0, max) : s;
}

export type NotifyArgs = { title: string; body?: string; reason?: string };
export type AskArgs = { question: string; context?: string };

export type InboxPayload = {
	kind: "fyi" | "question";
	sourceKind: "agent" | "system";
	sourceId: string;
	reason: string;
	title: string;
	body: string;
	blocking: boolean;
};

export function buildNotifyPayload(args: NotifyArgs, env: Record<string, string | undefined>): InboxPayload {
	const title = clip((args.title || "").trim(), MAX_TITLE);
	if (!title) throw new Error("notify_human needs a title");
	const who = agentIdentity(env);
	return {
		kind: "fyi",
		sourceKind: who.sourceKind,
		sourceId: who.sourceId,
		reason: clip((args.reason || "").trim() || "agent notification", MAX_TITLE),
		title,
		body: clip((args.body || "").trim(), MAX_BODY),
		blocking: false,
	};
}

export function buildAskPayload(args: AskArgs, env: Record<string, string | undefined>): InboxPayload {
	const question = (args.question || "").trim();
	if (!question) throw new Error("ask_human needs a question");
	const who = agentIdentity(env);
	let body = question;
	const context = (args.context || "").trim();
	if (context) body = question + "\n\n" + context;
	return {
		kind: "question",
		sourceKind: who.sourceKind,
		sourceId: who.sourceId,
		reason: "agent needs your input",
		title: clip(question, MAX_TITLE),
		body: clip(body, MAX_BODY),
		blocking: true,
	};
}
