/**
 * Delivery declarations for pi (ADR-0171): the `delivery` tool registers
 * a change in the repository this agent was launched in, asks for review,
 * or reads what is already declared. The daemon derives the repository
 * and the principal from that launch, so no vendor credential is
 * involved.
 *
 * A declaration, never a merge, a queue or a deploy: a review request is
 * not an approval, and the tool says so. With no reachable PiCode the
 * call fails softly — the model gets an explanatory text result, never a
 * thrown error to retry against.
 */

import { readFileSync } from "node:fs";
import { request as httpRequest } from "node:http";
import { request as httpsRequest } from "node:https";
import { homedir } from "node:os";
import { join } from "node:path";
import { defineTool, type ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Text } from "@earendil-works/pi-tui";
import { Type } from "typebox";
import {
	ACTIONS,
	buildRequest,
	identityFrom,
	rejectUnauthorizedFor,
	resolveDataDir,
	resolveServerUrl,
	resolveToken,
	summarize,
} from "../src/logic.ts";

const UNREACHABLE =
	"PiCode is not reachable (no server.json or connection refused) — nothing was declared. " +
	"Put the delivery in your final message instead of retrying.";

/** Re-read server.json every call: the port can rebind at runtime. PICODE_URL wins (ADR-0050). */
function serverUrl(): string | null {
	const dir = resolveDataDir(process.env, homedir());
	let text: string | null = null;
	try {
		text = readFileSync(join(dir, "server.json"), "utf8");
	} catch {
		text = null;
	}
	const info = resolveServerUrl(process.env, text);
	return info.ok ? info.url : null;
}

/** The bearer (ADR-0049), re-read per call so a rotation lands; PICODE_TOKEN wins. */
function installToken(): string {
	const dir = resolveDataDir(process.env, homedir());
	let text: string | null = null;
	try {
		text = readFileSync(join(dir, "token"), "utf8");
	} catch {
		text = null;
	}
	return resolveToken(process.env, text);
}

// node:https directly (zero deps — an extension resolves modules from its
// own path, so nothing beyond built-ins is guaranteed). PiCode serves
// localhost with a self-signed/mkcert cert Node doesn't trust:
// rejectUnauthorized:false for this one loopback request only (the
// in-repo precedent), never NODE_TLS_REJECT_UNAUTHORIZED.
function postJSON(url: URL, body: string): Promise<{ status: number; text: string }> {
	const { promise, resolve, reject } = Promise.withResolvers<{ status: number; text: string }>();
	const fn = url.protocol === "https:" ? httpsRequest : httpRequest;
	const headers: Record<string, string | number> = { "content-type": "application/json", "content-length": Buffer.byteLength(body) };
	const token = installToken();
	if (token) headers.authorization = "Bearer " + token;
	const req = fn(
		url,
		{
			method: "POST",
			headers,
			rejectUnauthorized: rejectUnauthorizedFor(url.toString()),
			timeout: 5000,
		},
		(res) => {
			let out = "";
			res.setEncoding("utf8");
			res.on("data", (chunk) => (out += chunk));
			res.on("end", () => resolve({ status: res.statusCode || 0, text: out }));
		},
	);
	req.on("timeout", () => req.destroy(new Error("timeout")));
	req.on("error", reject);
	req.end(body);
	return promise;
}

export default function piDelivery(pi: ExtensionAPI) {
	// The session id keys a retry, so it is read once and never guessed.
	let sessionId = "";
	pi.on("session_start", async (_event, ctx) => {
		try {
			sessionId = ctx.sessionManager.getSessionId?.() || "";
		} catch {
			sessionId = "";
		}
	});

	const tool = defineTool({
		name: "delivery",
		label: "Delivery",
		description:
			"Register a change and request review in the Delivery view of the repository this agent was launched in, or read what is already declared. " +
			"Actions: " +
			ACTIONS.join(", ") +
			". A declaration: no merge, queue, deploy or human approval happens here.",
		promptSnippet: "Declare your change in the project's Delivery view and ask the human for review",
		promptGuidelines: [
			"Register once the change is committed: title, branch, the full commit id and the target branch (usually main).",
			"Keep the requestId from the result when retrying the same call — the daemon replays instead of registering twice.",
			"request-review says the human should look at it; it never means approved, checked or integrated.",
			"After changing the deliverable, update with the returned id and version — an update clears the review request.",
		],
		parameters: Type.Object({
			action: Type.Union(ACTIONS.map((a) => Type.Literal(a)), { description: "What to do" }),
			title: Type.Optional(Type.String({ description: "One line describing the change (register, update)" })),
			branch: Type.Optional(Type.String({ description: "Local source branch of the change (register, update)" })),
			revision: Type.Optional(Type.String({ description: "Full commit id that branch points at (register, update); the revision the delivery declares (request-integration)" })),
			target: Type.Optional(Type.String({ description: "Local target branch, usually main (register, update, request-integration)" })),
			id: Type.Optional(Type.String({ description: "Delivery id from a previous result (update, request-review, withdraw-review, request-integration, show); queue entry id from the request or from show (withdraw-integration)" })),
			expectedVersion: Type.Optional(Type.Integer({ minimum: 1, description: "Version from the last result (update, request-review, withdraw-review); the queue entry's version (withdraw-integration)" })),
			requestId: Type.Optional(Type.String({ description: "Retry key; reuse it verbatim to retry the same declaration" })),
			before: Type.Optional(Type.Integer({ minimum: 0, description: "Pagination cursor returned by list" })),
		}),
		async execute(_toolCallId, params) {
			const built = buildRequest(params, identityFrom(process.env), sessionId);
			if (!built.ok) {
				return { content: [{ type: "text", text: `delivery refused: ${built.error}` }], details: { summary: built.error, ok: false }, isError: true };
			}
			const base = serverUrl();
			if (!base) {
				return { content: [{ type: "text", text: UNREACHABLE }], details: { summary: UNREACHABLE, ok: false }, isError: true };
			}
			let res: { status: number; text: string };
			try {
				res = await postJSON(new URL(base + "/api/delivery/tool"), built.body);
			} catch {
				return { content: [{ type: "text", text: UNREACHABLE }], details: { summary: UNREACHABLE, ok: false }, isError: true };
			}
			const ok = res.status >= 200 && res.status < 300;
			const text = res.text.trim() || `HTTP ${res.status}`;
			return { content: [{ type: "text", text }], details: { summary: summarize(res.text), ok }, isError: !ok };
		},
		renderCall(args, theme) {
			const a = (args || {}) as { action?: string; title?: string; id?: string };
			const head = theme.fg("toolTitle", theme.bold("delivery ")) + theme.fg("muted", a.action || "");
			const subject = a.title || a.id || "";
			return new Text(subject ? head + theme.fg("muted", " · " + subject) : head, 0, 0);
		},
		renderResult(result, _opts, theme) {
			const details = (result.details || {}) as { summary?: string; ok?: boolean };
			const line = details.summary || (result.content[0] as { text?: string } | undefined)?.text || "delivery";
			return new Text(details.ok ? line : theme.fg("warning", line), 0, 0);
		},
	});
	pi.registerTool(tool);
}
