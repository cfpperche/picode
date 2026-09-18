/**
 * Pure half of the `computer` tool (ADR-0148): how to find the daemon, and
 * how to turn its answers into what a model can use — an image block for a
 * capture, lines for a snapshot, one line for everything else. No network,
 * no filesystem; the extension does the I/O and the tests cover every branch.
 *
 * The connection helpers mirror pi-browser (each package is standalone;
 * there is no shared runtime module).
 */

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
		if (!/^https?:\/\/[^/\s@]+\/?$/.test(explicit)) return { ok: false, error: "PICODE_URL must be an origin like https://box:8445" };
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

// --- the catalog --------------------------------------------------------------

/** The 23 actions, the daemon's catalog and the shell's (ADR-0148). */
export const ACTIONS = [
	"screenshot", "zoom", "snapshot", "cursor_position", "wait", "windows", "focus",
	"left_click", "right_click", "middle_click", "double_click", "triple_click",
	"left_click_drag", "mouse_move", "left_mouse_down", "left_mouse_up", "scroll",
	"type", "key", "hold_key", "clipboard_read", "clipboard_write", "open",
] as const;

export type Action = (typeof ACTIONS)[number];

// --- what the daemon answers -------------------------------------------------

export type ToolAnswer = { action: string; output: unknown };

/** parseAnswer reads the daemon's reply body; a non-2xx carries its message. */
export function parseAnswer(status: number, body: string): { ok: true; answer: ToolAnswer } | { ok: false; error: string } {
	let data: unknown;
	try {
		data = JSON.parse(body);
	} catch {
		return { ok: false, error: status >= 400 ? `PiCode answered ${status}` : "PiCode answered with something that is not JSON" };
	}
	if (status >= 400) {
		const message = (data as { error?: unknown })?.error;
		return { ok: false, error: typeof message === "string" && message.trim() !== "" ? message : `PiCode answered ${status}` };
	}
	const action = (data as { action?: unknown })?.action;
	return { ok: true, answer: { action: typeof action === "string" ? action : "unknown", output: (data as { output?: unknown })?.output } };
}

// --- rendering --------------------------------------------------------------

/** ImageBlock is pi's image content: what the model sees. */
export type ImageBlock = { type: "image"; data: string; mimeType: string };

/** imageBlock turns a capture answer ({image, mime}) into the block, or null. */
export function imageBlock(output: unknown): ImageBlock | null {
	const o = output as { image?: unknown; mime?: unknown } | null;
	const data = o?.image;
	if (typeof data !== "string" || data === "") return null;
	const mime = typeof o?.mime === "string" && o.mime ? o.mime : "image/png";
	return { type: "image", data, mimeType: mime };
}

type Meta = {
	display?: number;
	window?: { exe?: string; title?: string; id?: number } | null;
	width?: number;
	height?: number;
	scale?: number;
	seq?: number;
	cursor?: [number, number] | null;
	preview?: string;
	ms?: number;
};

function metaOf(output: unknown): Meta {
	const m = (output as { meta?: unknown })?.meta;
	return m && typeof m === "object" ? (m as Meta) : {};
}

/** captureLine names what the image shows and the space its pixels live in. */
export function captureLine(output: unknown): string {
	const m = metaOf(output);
	const w = m.window;
	const what = w && w.exe ? `window ${w.exe}${w.title ? ` "${w.title}"` : ""}` : `display ${m.display ?? 1}`;
	const size = m.width && m.height ? ` (${m.width}×${m.height} image, scale ${m.scale ?? 1})` : "";
	const cursor = Array.isArray(m.cursor) ? `; cursor at [${m.cursor[0]}, ${m.cursor[1]}]` : "";
	return `Screenshot of ${what}${size} — coordinates you send are pixels of this image${cursor}.`;
}

/** previewDetails is the small still the PiCode UI shows as "Last capture". */
export function previewDetails(output: unknown, title?: string): { preview: { image: string; title: string; seq: number; final: boolean }; window?: unknown } | undefined {
	const m = metaOf(output);
	if (typeof m.preview !== "string" || !m.preview) return undefined;
	const w = m.window;
	return {
		preview: { image: m.preview, title: title || (w && w.title ? w.title : `display ${m.display ?? 1}`), seq: m.seq ?? 0, final: true },
		window: w ?? undefined,
	};
}

export const MAX_LINES = 400;

/** summarizeSnapshot renders the shell's accessibility lines, capped. */
export function summarizeSnapshot(output: unknown, max = MAX_LINES): string {
	const o = output as { lines?: unknown; dropped?: unknown; window?: { exe?: string; title?: string } } | null;
	const lines = Array.isArray(o?.lines) ? (o!.lines as unknown[]).filter((l) => typeof l === "string") as string[] : [];
	const w = o?.window;
	const head = w && (w.exe || w.title) ? `Accessibility tree of ${w.exe ?? "window"}${w.title ? ` "${w.title}"` : ""}` : "Accessibility tree";
	if (lines.length === 0) return head + ": nothing is exposed (the window may have no accessibility support).";
	const shown = lines.slice(0, max);
	let dropped = lines.length - shown.length;
	if (typeof o?.dropped === "number") dropped += o.dropped;
	return [head + " (refs, roles, names, centres in the last image):", ...shown, ...(dropped ? [`… ${dropped} more node(s) not shown`] : [])].join("\n");
}

/** summarizeWindows lists the windows the way the model should name them. */
export function summarizeWindows(output: unknown): string {
	const o = output as { windows?: unknown; displays?: unknown } | null;
	const ws = Array.isArray(o?.windows) ? (o!.windows as Array<Record<string, unknown>>) : [];
	const ds = Array.isArray(o?.displays) ? (o!.displays as Array<Record<string, unknown>>) : [];
	const lines: string[] = [];
	for (const d of ds) lines.push(`display ${d.index}${d.primary ? " (primary)" : ""}: ${Number(d.right) - Number(d.left)}×${Number(d.bottom) - Number(d.top)} at ${d.left},${d.top}`);
	for (const w of ws) {
		const flags = [w.foreground ? "front" : "", w.minimized ? "minimized" : ""].filter(Boolean).join(", ");
		lines.push(`window ${w.id} ${w.exe} "${String(w.title ?? "").slice(0, 80)}" display=${w.display}${flags ? ` [${flags}]` : ""}`);
	}
	return lines.length ? lines.join("\n") : "No windows are open.";
}

/** summarizeAct is the one-line answer for actions that return no image. */
export function summarizeAct(action: string, output: unknown): string {
	const o = (output ?? {}) as Record<string, unknown>;
	switch (action) {
		case "cursor_position":
			return Array.isArray(o.coordinate)
				? `Cursor at [${o.coordinate[0]}, ${o.coordinate[1]}] in the last image.`
				: `Cursor is outside the last image (screen ${JSON.stringify(o.screen ?? null)}).`;
		case "clipboard_read": {
			const text = typeof o.text === "string" ? o.text : "";
			const dropped = typeof o.dropped === "number" && o.dropped > 0 ? `\n… ${o.dropped} more character(s) not shown` : "";
			return text ? `Clipboard text:\n${text}${dropped}` : "The clipboard has no text.";
		}
		case "clipboard_write":
			return `Clipboard set (${o.chars ?? 0} characters).`;
		case "focus": {
			const w = o.window as { exe?: string; title?: string } | undefined;
			return w ? `Focused ${w.exe ?? "window"}${w.title ? ` "${w.title}"` : ""}.` : "Focused.";
		}
		case "open":
			return `Asked Windows to open ${String(o.target ?? "")}.`;
		case "mouse_move":
			return "Moved the pointer.";
		case "left_mouse_down":
			return "Left button held down.";
		case "left_mouse_up":
			return "Left button released.";
		default:
			return o.ok === true ? `${action}: ok` : `${action}: done`;
	}
}
