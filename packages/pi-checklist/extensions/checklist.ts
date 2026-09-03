/**
 * Opt-in internal checklist for pi (ADR-0055).
 *
 * The model writes its plan with the `checklist` tool before the first
 * change of a task and keeps it updated; a mutating tool call before the
 * checklist is refused with the way out; under `always` a turn that ends
 * without one gets a reminder. Every change is POSTed to PiCode, which
 * shows the current step on the sidebar. With no reachable PiCode the
 * tool still works — the plan lives in the session either way.
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
	agentId,
	buildPayload,
	contractPrompt,
	decideGate,
	decideReminder,
	GLYPH,
	normalizeItems,
	parseLevel,
	reconstruct,
	rejectUnauthorizedFor,
	REMINDER,
	renderLines,
	resolveDataDir,
	resolveServerUrl,
	resolveToken,
	STATUSES,
	summarize,
	type Item,
	type Payload,
} from "../src/logic.ts";

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

function postJSON(url: URL, body: string): Promise<number> {
	return new Promise((resolve, reject) => {
		const fn = url.protocol === "https:" ? httpsRequest : httpRequest;
		const headers: Record<string, string | number> = { "content-type": "application/json", "content-length": Buffer.byteLength(body) };
		const token = installToken();
		if (token) headers.authorization = "Bearer " + token;
		const req = fn(url, { method: "POST", headers, rejectUnauthorized: rejectUnauthorizedFor(url.toString()), timeout: 5000 }, (res) => {
			res.resume();
			res.on("end", () => resolve(res.statusCode || 0));
		});
		req.on("timeout", () => req.destroy(new Error("timeout")));
		req.on("error", reject);
		req.end(body);
	});
}

/** Best-effort, never awaited by the model's turn: PiCode absent is not an error here. */
function publish(payload: Payload): void {
	const id = agentId(process.env);
	const base = serverUrl();
	if (!id || !base) return;
	postJSON(new URL(`${base}/api/agents/${encodeURIComponent(id)}/checklist`), JSON.stringify(payload)).catch(() => {});
}

export default function piChecklist(pi: ExtensionAPI) {
	const level = parseLevel(process.env);
	let items: Item[] | null = null; // the session's list
	let planned = false; // the checklist tool ran since this task started
	let reminders = 0; // sent for this task
	let sessionId = "";

	const tool = defineTool({
		name: "checklist",
		label: "Checklist",
		description:
			"Write or update your internal checklist for the current task: the whole list of concrete steps with their status. " +
			"Call it before your first change, and again whenever a step starts or finishes.",
		promptSnippet: "Write or update the task checklist (plan) the human follows from the sidebar",
		promptGuidelines: [
			"Call checklist before the first edit, write or bash of a task; changes are refused until then.",
			"Send the whole list every time: mark the finished step completed and the next one in-progress in one call.",
			"Keep steps concrete and short (one line each); 2–8 steps is the usual size.",
		],
		parameters: Type.Object({
			items: Type.Array(
				Type.Object({
					text: Type.String({ description: "One concrete step, one line" }),
					status: Type.Optional(Type.Union(STATUSES.map((s) => Type.Literal(s)), { description: "pending (default), in-progress, completed" })),
				}),
				{ description: "The whole checklist, in order" },
			),
		}),
		async execute(_toolCallId, params) {
			let next: Item[];
			try {
				next = normalizeItems(params.items);
			} catch (err) {
				return { content: [{ type: "text", text: `checklist refused: ${(err as Error).message}` }], details: { items: items || [] }, isError: true };
			}
			items = next;
			planned = true;
			publish(buildPayload(items, { sessionId }));
			return { content: [{ type: "text", text: summarize(items) }], details: { items } };
		},
		renderCall(args, theme) {
			const list = Array.isArray((args as { items?: unknown }).items) ? ((args as { items: unknown[] }).items as { text?: string; status?: string }[]) : [];
			const done = list.filter((it) => it && it.status === "completed").length;
			return new Text(theme.fg("toolTitle", theme.bold("checklist ")) + theme.fg("muted", `${done}/${list.length} completed`), 0, 0);
		},
		renderResult(result, _opts, theme) {
			const list = ((result.details as { items?: Item[] } | undefined)?.items || []) as Item[];
			if (!list.length) return new Text(theme.fg("warning", (result.content[0] as { text?: string } | undefined)?.text || "no checklist"), 0, 0);
			const lines = renderLines(list).map((line, i) => {
				const st = list[i]!.status;
				return st === "in-progress" ? theme.fg("accent", line) : st === "completed" ? theme.fg("dim", line) : line;
			});
			return new Text(lines.join("\n"), 0, 0);
		},
	});
	pi.registerTool(tool);

	pi.on("session_start", async (_event, ctx) => {
		planned = false;
		reminders = 0;
		try {
			sessionId = ctx.sessionManager.getSessionId?.() || "";
		} catch {
			sessionId = "";
		}
		let entries: unknown[] = [];
		try {
			entries = ctx.sessionManager.getBranch();
		} catch {
			entries = [];
		}
		items = reconstruct(entries);
		if (items) publish(buildPayload(items, { sessionId }));
	});

	pi.on("before_agent_start", async (event) => {
		// A new task: the plan must be written (or updated) again before a change.
		planned = false;
		reminders = 0;
		const contract = contractPrompt(level);
		if (!contract) return undefined;
		return { systemPrompt: `${event.systemPrompt}\n\n${contract}` };
	});

	pi.on("tool_call", async (event) => {
		try {
			const d = decideGate({ level, toolName: event.toolName, planned });
			if (!d.block) return undefined;
			publish(buildPayload(items || [], { sessionId, blocked: true }));
			return { block: true, reason: d.reason };
		} catch {
			return undefined; // fail open: a broken gate must not stop the agent
		}
	});

	pi.on("agent_end", async () => {
		if (!decideReminder({ level, planned, sent: reminders })) {
			if (level === "always" && !planned) publish(buildPayload(items || [], { sessionId, absent: true }));
			return;
		}
		reminders += 1;
		publish(buildPayload(items || [], { sessionId, absent: true }));
		pi.sendMessage({ customType: "pi-checklist", content: REMINDER, display: true }, { deliverAs: "followUp", triggerTurn: true });
	});
}

// Keep the glyph table reachable for renderers that import this file.
export { GLYPH };
