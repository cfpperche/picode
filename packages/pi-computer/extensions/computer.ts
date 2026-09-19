/**
 * The `computer` tool (ADR-0148): an agent uses the Windows desktop through
 * PiCode's desktop app — screenshots, mouse, keyboard, windows, clipboard,
 * programs — behind one grant the human switches on per agent or terminal
 * (Settings ▸ Computer). The daemon checks the grant, the shell checks its
 * mirrored copy and runs the action on its desk thread; nothing here decides.
 *
 * Desktop-only by nature: with no shell on the line the daemon answers "the
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
	ACTIONS,
	captureLine,
	imageBlock,
	parseAnswer,
	previewDetails,
	rejectUnauthorizedFor,
	resolveDataDir,
	resolveServerUrl,
	resolveToken,
	summarizeAct,
	summarizeSnapshot,
	summarizeWindows,
} from "../src/logic.ts";

async function readText(path: string): Promise<string | null> {
	try {
		return await readFile(path, "utf8");
	} catch {
		return null;
	}
}

let token = "";

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

const ACTIONS_WITH_IMAGE = new Set(["screenshot", "zoom", "wait", "left_click", "right_click", "middle_click", "double_click", "triple_click", "left_click_drag", "scroll", "type", "key", "hold_key"]);

export default function piComputer(pi: ExtensionAPI) {
	const tool = defineTool({
		name: "computer",
		label: "Computer",
		description:
			"Use the Windows desktop through PiCode's desktop app: see a monitor or a window (screenshot, zoom, snapshot of its accessibility tree), " +
			"list and focus windows, click, drag, scroll, type, press keys, read and write the clipboard, open a program or a file. " +
			"Needs the human's grant for this agent (Settings ▸ Computer); with it you act with their own permissions on their desktop.",
		promptSnippet: "Use the computer (screen, mouse, keyboard) through PiCode's desktop app",
		promptGuidelines: [
			"A screenshot is one whole monitor (display 1 by default) or one window when you pass `window`; every coordinate you send is a pixel of the LAST image you received — not a screen coordinate. Take a screenshot before the first click and after a window changes.",
			"Clicks, typing, scrolling and keys return a fresh image: read it before the next step. `windows` lists what is open; `focus` brings a window forward before you type into it.",
			"`snapshot` reads a window's accessibility tree with each element's centre in the last image — use it to find small controls and to read text back instead of guessing from pixels.",
			"You act with the human's permissions on their desktop. Before paying, sending a message, deleting or overwriting files, or typing a password, stop and confirm with the human.",
			"A refusal names what is missing (the grant, the desktop app, a stale window id, a window that is no longer in front: `foreground_changed` means the human moved — look again or `focus` the window, then act); say so and do not retry around it.",
		],
		parameters: Type.Object({
			action: Type.Union(
				ACTIONS.map((a) => Type.Literal(a)),
				{ description: "screenshot | zoom | snapshot | cursor_position | wait | windows | focus | left_click | right_click | middle_click | double_click | triple_click | left_click_drag | mouse_move | left_mouse_down | left_mouse_up | scroll | type | key | hold_key | clipboard_read | clipboard_write | open" },
			),
			coordinate: Type.Optional(Type.Array(Type.Integer(), { minItems: 2, maxItems: 2, description: "[x, y] in pixels of the last image (clicks, mouse_move, scroll, drag end)" })),
			start_coordinate: Type.Optional(Type.Array(Type.Integer(), { minItems: 2, maxItems: 2, description: "left_click_drag: where the drag starts" })),
			text: Type.Optional(Type.String({ description: "type: the text; key/hold_key: the chord (ctrl+s, Return, alt+F4); clicks: modifiers to hold (shift, ctrl+shift); clipboard_write: the text" })),
			scroll_direction: Type.Optional(Type.Union([Type.Literal("up"), Type.Literal("down"), Type.Literal("left"), Type.Literal("right")])),
			scroll_amount: Type.Optional(Type.Integer({ minimum: 1, maximum: 50, description: "scroll: wheel notches (default 3)" })),
			duration: Type.Optional(Type.Number({ minimum: 0, maximum: 5, description: "wait/hold_key: seconds, at most 5" })),
			repeat: Type.Optional(Type.Integer({ minimum: 1, maximum: 100, description: "key: how many times" })),
			region: Type.Optional(Type.Array(Type.Integer(), { minItems: 4, maxItems: 4, description: "zoom: [x0, y0, x1, y1] in the last image" })),
			display: Type.Optional(Type.Integer({ minimum: 1, description: "screenshot: which monitor (1 = primary)" })),
			window: Type.Optional(Type.Integer({ description: "screenshot/snapshot/focus: a window id from `windows`" })),
			depth: Type.Optional(Type.Integer({ minimum: 0, maximum: 24, description: "snapshot: how deep to walk (default 24)" })),
			target: Type.Optional(Type.String({ description: "open: a program name (notepad.exe), a file path or a URL" })),
		}),
		async execute(_toolCallId, params) {
			const dataDir = resolveDataDir(process.env, homedir());
			const serverJson = await readText(join(dataDir, "server.json"));
			const info = resolveServerUrl(process.env, serverJson);
			if (!info.ok) {
				return { content: [{ type: "text", text: `computer: PiCode is not reachable (${info.error}) — is the daemon running?` }], isError: true };
			}
			token = resolveToken(process.env, await readText(join(dataDir, "token")));
			const { action, ...rest } = params;
			const body = JSON.stringify({
				agent: (process.env.PICODE_AGENT_ID || "").trim(),
				// ADR-0143: the same identity tuple pi-browser and pi-inbox use.
				term: (process.env.PICODE_TERM_ID || "").trim(),
				call: _toolCallId,
				action,
				params: rest,
			});
			let answer: { status: number; text: string };
			try {
				answer = await postJSON(new URL(info.url + "/api/computer/tool"), body);
			} catch (err) {
				return { content: [{ type: "text", text: `computer: ${(err as Error).message}` }], isError: true };
			}
			const parsed = parseAnswer(answer.status, answer.text);
			if (!parsed.ok) {
				return { content: [{ type: "text", text: `computer: ${parsed.error}` }], isError: true };
			}
			const { output } = parsed.answer;
			if (ACTIONS_WITH_IMAGE.has(action)) {
				const image = imageBlock(output);
				if (!image) {
					return { content: [{ type: "text", text: `computer: ${action} ran but no image came back` }], isError: true };
				}
				const details = previewDetails(output);
				return { content: [{ type: "text", text: captureLine(output) }, image], ...(details ? { details } : {}) };
			}
			if (action === "snapshot") {
				return { content: [{ type: "text", text: summarizeSnapshot(output) }] };
			}
			if (action === "windows") {
				return { content: [{ type: "text", text: summarizeWindows(output) }] };
			}
			return { content: [{ type: "text", text: summarizeAct(action, output) }] };
		},
		renderCall(args, theme) {
			const a = args as { action?: string; coordinate?: number[]; text?: string; target?: string };
			const detail = a.coordinate ? ` [${a.coordinate.join(",")}]` : a.text ? ` ${JSON.stringify(a.text).slice(0, 40)}` : a.target ? ` ${a.target}` : "";
			return new Text(theme.fg("toolTitle", theme.bold("computer ")) + theme.fg("muted", String(a.action || "") + detail), 0, 0);
		},
		renderResult(result, _opts, theme) {
			const text = (result.content[0] as { text?: string } | undefined)?.text || "";
			const first = text.split("\n")[0] || "no answer";
			return new Text(theme.fg(result.isError ? "warning" : "dim", first.length > 120 ? first.slice(0, 120) + "…" : first), 0, 0);
		},
	});
	pi.registerTool(tool);
}
