/**
 * Pure logic for pi-delivery. No pi imports — node:test covers this.
 *
 * The extension file (extensions/delivery.ts) is I/O glue: it reads
 * server.json, POSTs what these functions build and registers the tool
 * with pi. Identity, Git checks and persistence stay in the daemon
 * (ADR-0171); this module shapes one request and reads one answer.
 */

import { createHash } from "node:crypto";

/** The actions the daemon serves; queues and deploy are not among them. */
export const ACTIONS = ["capabilities", "register", "update", "request-review", "withdraw-review", "show", "list"] as const;
export type Action = (typeof ACTIONS)[number];

/** Mutations carry a retry key and answer with a version. */
export const MUTATIONS: readonly string[] = ["register", "update", "request-review", "withdraw-review"];

/**
 * What each action may carry (`store.ValidateDeliveryMutation`). The daemon
 * refuses a field the action does not take — a review action is answered
 * `review actions take only id, expectedVersion and requestId` — so the
 * shape lives here too and a wrong field is named before a request is sent.
 */
export const FIELDS_BY_ACTION: Record<Action, readonly string[]> = {
	capabilities: [],
	register: ["title", "branch", "revision", "target"],
	update: ["id", "expectedVersion", "title", "branch", "revision", "target"],
	"request-review": ["id", "expectedVersion"],
	"withdraw-review": ["id", "expectedVersion"],
	show: ["id"],
	list: ["before"],
};

/** `requestId` length the daemon accepts. */
export const MAX_REQUEST_ID = 128;

export type Identity = { agent: string; term: string };

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

/** The install token beside server.json (ADR-0049); "" when absent. */
export function parseToken(text: string | null | undefined): string {
	const t = (text || "").trim();
	return /^[0-9a-f]{32,128}$/i.test(t) ? t : "";
}

/**
 * Where to post (ADR-0050): PICODE_URL names a PiCode on another
 * machine and wins; else server.json (text may be null when absent).
 */
export function resolveServerUrl(env: Record<string, string | undefined>, serverJson: string | null): ServerInfo {
	const explicit = (env.PICODE_URL || "").trim();
	if (explicit) {
		if (!/^https?:\/\/[^\s/]+\/?$/.test(explicit)) return { ok: false, error: "PICODE_URL must be an origin like https://box:8445" };
		return { ok: true, url: explicit.replace(/\/+$/, "") };
	}
	if (serverJson === null) return { ok: false, error: "no server.json" };
	return parseServerJson(serverJson);
}

/** The bearer: PICODE_TOKEN (remote) wins over the token file's text. */
export function resolveToken(env: Record<string, string | undefined>, fileText: string | null | undefined): string {
	return parseToken(env.PICODE_TOKEN) || parseToken(fileText);
}

/**
 * TLS policy for the one request: a self-signed / mkcert cert on
 * loopback is accepted; anything else must present a trusted chain.
 */
export function rejectUnauthorizedFor(url: string): boolean {
	try {
		const h = new URL(url).hostname.replace(/^\[|\]$/g, "");
		return !(h === "localhost" || h === "127.0.0.1" || h === "::1");
	} catch {
		return true;
	}
}

/**
 * The launch identity PiCode stamps on the process: a managed agent's id,
 * or the terminal a pi runs in (ADR-0089). Both are the launch, never a
 * credential the tool carries.
 */
export function identityFrom(env: Record<string, string | undefined>): Identity {
	return { agent: (env.PICODE_AGENT_ID || "").trim(), term: (env.PICODE_TERM_ID || "").trim() };
}

/**
 * The retry key for a mutation the model did not key itself: derived from
 * the session, the action and the payload, so repeating the same call
 * retries the same declaration instead of registering a second one. Pass
 * `requestId` explicitly to force a distinct key.
 */
export function stableRequestId(sessionId: string, action: string, fields: Record<string, unknown>): string {
	const material = JSON.stringify([sessionId, action, Object.keys(fields).sort().map((key) => [key, fields[key]])]);
	return createHash("sha256").update(material).digest("hex").slice(0, 40);
}

export type Built = { ok: true; body: string } | { ok: false; error: string };

/**
 * Shape one `POST /api/delivery/tool` request. A field the action does not
 * take is named, never dropped; empty values are left out so the daemon
 * sees the same payload the model meant.
 */
export function buildRequest(input: unknown, identity: Identity, sessionId: string): Built {
	const args = (input && typeof input === "object" ? input : {}) as Record<string, unknown>;
	const action = String(args.action ?? "").trim() as Action;
	if (!(ACTIONS as readonly string[]).includes(action)) {
		return { ok: false, error: `action must be one of ${ACTIONS.join(", ")}` };
	}
	const allowed = FIELDS_BY_ACTION[action];
	const mutation = MUTATIONS.includes(action);
	for (const key of Object.keys(args)) {
		if (key === "action") continue;
		if (key === "requestId" && mutation) continue;
		if (!allowed.includes(key)) {
			const takes = allowed.length ? ` — it takes ${allowed.join(", ")}` : " — it takes no other fields";
			return { ok: false, error: `${action} does not take ${key}${takes}` };
		}
	}
	if (!identity.agent && !identity.term) {
		return { ok: false, error: "no PiCode identity: open this agent through PiCode, then call again" };
	}
	const payload: Record<string, unknown> = { action };
	for (const key of allowed) {
		const value = args[key];
		if (value === undefined || value === null || value === "") continue;
		payload[key] = value;
	}
	if (mutation) {
		const given = typeof args.requestId === "string" ? args.requestId.trim() : "";
		payload.requestId = given && given.length <= MAX_REQUEST_ID ? given : stableRequestId(sessionId, action, payload);
	}
	if (identity.agent) payload.agent = identity.agent;
	if (identity.term) payload.term = identity.term;
	return { ok: true, body: JSON.stringify(payload) };
}

/** One line for the tool's render, from the daemon's answer. */
export function summarize(raw: string): string {
	let data: unknown;
	try {
		data = JSON.parse(raw);
	} catch {
		return raw.trim().slice(0, 200) || "no answer";
	}
	const d = data as {
		error?: string;
		replayed?: boolean;
		actions?: unknown[];
		deliveries?: unknown[];
		delivery?: { id?: string; version?: number; branch?: string; target?: string; review?: string };
	};
	if (d?.error) return d.error;
	if (d?.delivery) {
		const parts = [`${d.replayed ? "Replayed " : ""}delivery ${d.delivery.id || "?"} v${d.delivery.version ?? "?"}`];
		if (d.delivery.branch) parts.push(d.delivery.target ? `${d.delivery.branch} → ${d.delivery.target}` : d.delivery.branch);
		parts.push(d.delivery.review === "requested" ? "review requested" : "no review request");
		return parts.join(" · ");
	}
	if (Array.isArray(d?.deliveries)) return `${d.deliveries.length} delivery record${d.deliveries.length === 1 ? "" : "s"}`;
	if (Array.isArray(d?.actions)) return "no deliveries here yet — register one first";
	return raw.trim().slice(0, 200) || "no answer";
}
