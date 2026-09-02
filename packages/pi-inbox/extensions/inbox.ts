/**
 * Opt-in inbox tools for pi (ADR-0037).
 *
 * notify_human files an FYI into PiCode's inbox; ask_human files a
 * blocking question and ends the turn (terminate: true) — the human's
 * reply arrives later as a follow-up message through PiCode's queue.
 * With no reachable PiCode the tools fail softly: the model gets an
 * explanatory text result, never a thrown error to retry against.
 */

import { readFileSync } from "node:fs";
import { request as httpRequest } from "node:http";
import { request as httpsRequest } from "node:https";
import { homedir } from "node:os";
import { join } from "node:path";
import { defineTool, type ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import {
	buildAskPayload,
	buildNotifyPayload,
	rejectUnauthorizedFor,
	resolveDataDir,
	resolveServerUrl,
	resolveToken,
} from "../src/logic.ts";

const UNREACHABLE =
	"PiCode is not reachable (no server.json or connection refused) — could not file to the inbox. " +
	"Proceed without human input or surface this in your final message.";

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
	return new Promise((resolve, reject) => {
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
	});
}

async function post(payload: InboxPayload): Promise<{ ok: boolean; error?: string }> {
	const base = serverUrl();
	if (!base) return { ok: false };
	try {
		const res = await postJSON(new URL(base + "/api/inbox"), JSON.stringify(payload));
		if (res.status < 200 || res.status >= 300) {
			let msg = `HTTP ${res.status}`;
			try {
				const parsed = JSON.parse(res.text) as { error?: string };
				if (parsed.error) msg = parsed.error;
			} catch {
				/* keep the status */
			}
			return { ok: false, error: msg };
		}
		return { ok: true };
	} catch {
		return { ok: false };
	}
}

const notifyTool = defineTool({
	name: "notify_human",
	label: "Notify human",
	description:
		"File a non-blocking note into the human's PiCode inbox. Use it only for things the human must know about; it never interrupts them.",
	promptSnippet: "File a non-blocking note into the human's inbox",
	promptGuidelines: [
		"Use notify_human only for things the human must know; silence is a valid outcome.",
		"Send one consolidated notify_human per state change, never one per event.",
		"Your final answer belongs in your reply to the user, not in notify_human.",
	],
	parameters: Type.Object({
		title: Type.String({ description: "Short headline (what happened)" }),
		body: Type.Optional(Type.String({ description: "Details, markdown" })),
		reason: Type.Optional(Type.String({ description: "Why the human is seeing this" })),
	}),
	async execute(_toolCallId, params) {
		let payload: InboxPayload;
		try {
			payload = buildNotifyPayload(params, process.env);
		} catch (err) {
			throw new Error((err as Error).message); // caller bug → real error
		}
		const res = await post(payload);
		if (!res.ok) {
			return { content: [{ type: "text", text: res.error ? `Inbox refused the item: ${res.error}` : UNREACHABLE }] };
		}
		return { content: [{ type: "text", text: "Filed to the human's inbox." }] };
	},
});

const askTool = defineTool({
	name: "ask_human",
	label: "Ask human",
	description:
		"File a blocking question into the human's PiCode inbox and end the turn. The human's reply arrives later as a follow-up message.",
	promptSnippet: "Ask the human a question via their inbox; the reply arrives as a follow-up",
	promptGuidelines: [
		"Use ask_human when you are genuinely blocked on a decision only the human can make.",
		"After ask_human, stop — the turn ends and the reply arrives as a follow-up message.",
		"Ask one consolidated question; do not file several ask_human items in a row.",
	],
	parameters: Type.Object({
		question: Type.String({ description: "The question, one sentence if possible" }),
		context: Type.Optional(Type.String({ description: "What the human needs to answer well, markdown" })),
	}),
	async execute(_toolCallId, params) {
		let payload: InboxPayload;
		try {
			payload = buildAskPayload(params, process.env);
		} catch (err) {
			throw new Error((err as Error).message);
		}
		const res = await post(payload);
		if (!res.ok) {
			return { content: [{ type: "text", text: res.error ? `Inbox refused the question: ${res.error}` : UNREACHABLE }] };
		}
		return {
			content: [
				{ type: "text", text: "Question filed to the human's inbox; their reply will arrive as a follow-up message." },
			],
			terminate: true, // park-and-wake: the turn ends here
		};
	},
});

export default function piInbox(pi: ExtensionAPI) {
	pi.registerTool(notifyTool);
	pi.registerTool(askTool);
}
