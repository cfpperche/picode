/**
 * The `browser` tool (ADR-0132, ADR-0134): read the work-browser tab the human
 * has open in the PiCode desktop app.
 *
 * The tool names a **verb**, never a CDP method. The daemon maps the verb
 * (`browser.VerbFor`), resolves the agent's policy — read on the tab on screen
 * unless a grant says more — and the shell re-checks the method against its
 * tier catalog. Nothing here can widen that: an unknown verb is refused by the
 * daemon, and a method the catalog does not name is refused by the shell.
 *
 * Desktop-only by nature. With no shell on the line the daemon answers "the
 * desktop app is not connected", and that is what the model gets.
 */

import { readFile } from "node:fs/promises";
import { request as httpRequest } from "node:http";
import { request as httpsRequest } from "node:https";
import { homedir } from "node:os";
import { join } from "node:path";
import { defineTool, type ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Text } from "@earendil-works/pi-tui";
import { Type } from "typebox";
import {
	imageBlock,
	parseAnswer,
	rejectUnauthorizedFor,
	resolveDataDir,
	resolveServerUrl,
	resolveToken,
	summarizeAx,
	summarizeCdp,
	summarizeEvaluate,
	summarizeEvents,
	summarizeHistory,
	summarizeNavigate,
} from "../src/logic.ts";

async function readText(path: string): Promise<string | null> {
	try {
		return await readFile(path, "utf8");
	} catch {
		return null;
	}
}

function serverURL(serverJson: string | null): string | null {
	const info = resolveServerUrl(process.env, serverJson);
	return info.ok ? info.url : null;
}

function postJSON(url: URL, body: string): Promise<{ status: number; text: string }> {
	return new Promise((resolve, reject) => {
		const fn = url.protocol === "https:" ? httpsRequest : httpRequest;
		const headers: Record<string, string | number> = {
			"content-type": "application/json",
			"content-length": Buffer.byteLength(body),
		};
		if (token) headers.authorization = "Bearer " + token;
		const req = fn(url, { method: "POST", headers, rejectUnauthorized: rejectUnauthorizedFor(url.toString()), timeout: 60_000 }, (res) => {
			const chunks: Buffer[] = [];
			res.on("data", (c: Buffer) => chunks.push(c));
			res.on("end", () => resolve({ status: res.statusCode || 0, text: Buffer.concat(chunks).toString("utf8") }));
		});
		req.on("timeout", () => req.destroy(new Error("timeout")));
		req.on("error", reject);
		req.end(body);
	});
}

let token = "";

export default function piBrowser(pi: ExtensionAPI) {
	const tool = defineTool({
		name: "browser",
		label: "Browser",
		description:
			"Read the web page the human has open in PiCode's work browser (the desktop app). " +
			"Verbs: snapshot (the page as an accessibility tree — roles and names), screenshot (the page as an image you see), " +
			"events (what the tab recorded: navigation, console, network), cdp (one Chrome DevTools Protocol method by name; needs Developer mode), history (where the human has been; off until they allow it in Settings). " +
			"Read-only by default: it cannot click, type or navigate.",
		promptSnippet: "Read the page open in PiCode's work browser (desktop app)",
		promptGuidelines: [
			"Use snapshot to read the page's structure, screenshot when the visual matters, events for what happened since the last poll.",
			"This reads the tab the human has on screen; if they are looking elsewhere, say which page you read.",
			"It cannot act on the page: clicking and navigation need a per-agent grant (Settings ▸ Browser).",
			"cdp names one protocol method and needs two things the human controls: Developer mode on, and the Full tier for this agent. A refusal says which is missing — do not retry around it.",
		],
		parameters: Type.Object({
			verb: Type.Union(
				[
					Type.Literal("snapshot"),
					Type.Literal("screenshot"),
					Type.Literal("events"),
					Type.Literal("evaluate"),
					Type.Literal("navigate"),
					Type.Literal("cdp"),
					Type.Literal("history"),
				],
				{ description: "snapshot | screenshot | events | evaluate | navigate | cdp (evaluate/navigate need a grant; cdp needs Developer mode and the Full tier)" },
			),
			since: Type.Optional(Type.Number({ description: "events only: the last sequence number you saw" })),
			expression: Type.Optional(Type.String({ description: "evaluate only: the JavaScript expression to run" })),
			query: Type.Optional(Type.String({ description: "history only: a search over url and title" })),
			limit: Type.Optional(Type.Number({ description: "history only: how many visits to return (default 100, max 200)" })),
			url: Type.Optional(Type.String({ description: "navigate only: the destination; its origin must be in the grant" })),
			method: Type.Optional(Type.String({ description: "cdp only: the full protocol method name, e.g. Network.getAllCookies" })),
			params: Type.Optional(Type.String({ description: "cdp only: the method's parameters as a JSON object, e.g. {\"urls\":true}" })),
		}),
		async execute(_toolCallId, params) {
			const dataDir = resolveDataDir(process.env, homedir());
			const url = serverURL(await readText(join(dataDir, "server.json")));
			if (!url) {
				return {
					content: [{ type: "text", text: "browser: PiCode is not reachable (no server.json) — is the daemon running?" }],
					isError: true,
				};
			}
			token = resolveToken(process.env, await readText(join(dataDir, "token")));
			// cdp names a protocol method the daemon does not resolve from its
			// catalog (ADR-0144): the method and its JSON parameters travel as
			// the verb's params, and the daemon decides whether this caller may
			// have them at all.
			let cdpParams: unknown;
			if (params.params) {
				try {
					cdpParams = JSON.parse(params.params);
				} catch {
					return { content: [{ type: "text", text: "browser: params must be a JSON object" }], isError: true };
				}
			}
			const body = JSON.stringify({
				agent: (process.env.PICODE_AGENT_ID || "").trim(),
				// ADR-0143: the same identity tuple pi-inbox and pi-checklist use —
				// a CLI in a PiCode terminal has no agent id, only a terminal one.
				term: (process.env.PICODE_TERM_ID || "").trim(),
				verb: params.verb,
				params: { since: params.since, expression: params.expression, url: params.url, method: params.method, params: cdpParams, query: params.query, limit: params.limit },
			});
			let answer: { status: number; text: string };
			try {
				answer = await postJSON(new URL(url + "/api/browser/tool"), body);
			} catch (err) {
				return { content: [{ type: "text", text: `browser: ${(err as Error).message}` }], isError: true };
			}
			const parsed = parseAnswer(answer.status, answer.text);
			if (!parsed.ok) {
				return { content: [{ type: "text", text: `browser: ${parsed.error}` }], isError: true };
			}
			const { verb, output } = parsed.answer;

			if (verb === "screenshot") {
				// The capture goes to the model as an image block (pi resizes it
				// after the tool_result hook), not as a temp path it would then
				// have to read: one call to see the page.
				const image = imageBlock(output);
				if (!image) {
					return { content: [{ type: "text", text: "browser: the page could not be captured" }], isError: true };
				}
				return { content: [{ type: "text", text: "Screenshot of the page on screen (attached)." }, image] };
			}
			if (verb === "history") {
				return { content: [{ type: "text", text: summarizeHistory(output) }] };
			}
			if (verb === "events") {
				const lines = summarizeEvents(output);
				return { content: [{ type: "text", text: lines.length ? lines.join("\n") : "The tab recorded nothing." }] };
			}
			if (verb === "evaluate") {
				return { content: [{ type: "text", text: summarizeEvaluate(output) }] };
			}
			if (verb === "navigate") {
				return { content: [{ type: "text", text: summarizeNavigate(output, params.url) }] };
			}
			if (verb === "cdp") {
				return { content: [{ type: "text", text: summarizeCdp(output) }] };
			}
			const { lines, dropped } = summarizeAx(output);
			const text = lines.length
				? lines.join("\n") + (dropped ? `\n… ${dropped} more node(s) not shown` : "")
				: "The page has no accessible content (it may still be loading).";
			return { content: [{ type: "text", text }] };
		},
		renderCall(args, theme) {
			return new Text(theme.fg("toolTitle", theme.bold("browser ")) + theme.fg("muted", String((args as { verb?: string }).verb || "")), 0, 0);
		},
		renderResult(result, _opts, theme) {
			const text = (result.content[0] as { text?: string } | undefined)?.text || "";
			const first = text.split("\n")[0] || "no answer";
			return new Text(theme.fg(result.isError ? "warning" : "dim", first.length > 120 ? first.slice(0, 120) + "…" : first), 0, 0);
		},
	});
	pi.registerTool(tool);
}
