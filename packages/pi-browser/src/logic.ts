/**
 * Pure half of the `browser` tool: how to find the daemon, and how to turn its
 * answers into lines a model can use. No network, no filesystem — the
 * extension does the I/O, the tests exercise every branch here.
 *
 * The connection helpers mirror pi-checklist/pi-inbox (each package is
 * standalone; there is no shared runtime module).
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

// --- what the daemon answers -------------------------------------------------

export type ToolAnswer = { verb: string; output: unknown };

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
	const verb = (data as { verb?: unknown })?.verb;
	return { ok: true, answer: { verb: typeof verb === "string" ? verb : "unknown", output: (data as { output?: unknown })?.output } };
}

// --- rendering --------------------------------------------------------------

// MAX_LINES keeps a big page from filling the model's context; the count of
// what was dropped is reported instead of silently truncating.
export const MAX_LINES = 200;

/**
 * summarizeAx renders an accessibility tree as `role "name"` lines. It is the
 * read verb's whole value: roles and names, no script, no page text that is not
 * exposed to assistive tech.
 */
export function summarizeAx(output: unknown, max = MAX_LINES): { lines: string[]; dropped: number } {
	const nodes = (output as { nodes?: unknown })?.nodes;
	if (!Array.isArray(nodes)) return { lines: [], dropped: 0 };
	const lines: string[] = [];
	let dropped = 0;
	for (const node of nodes) {
		const role = (node as { role?: { value?: unknown } })?.role?.value;
		if (typeof role !== "string" || role === "" || role === "none" || role === "generic" || role === "InlineTextBox") continue;
		const name = String((node as { name?: { value?: unknown } })?.name?.value ?? "").replace(/\s+/g, " ").trim();
		if (name === "" && !["image", "textbox", "heading", "paragraph", "list", "table"].includes(role)) continue;
		if (lines.length >= max) {
			dropped += 1;
			continue;
		}
		lines.push(name ? `${role} "${name}"` : role);
	}
	return { lines, dropped };
}

/** ImageBlock is pi's image content: what the model sees instead of a path. */
export type ImageBlock = { type: "image"; data: string; mimeType: string };

/**
 * imageBlock turns the daemon's screenshot answer ({data: base64 PNG}) into
 * the image block pi hands the model, or null when nothing was captured.
 */
export function imageBlock(output: unknown): ImageBlock | null {
	const data = (output as { data?: unknown } | null)?.data;
	if (typeof data !== "string" || data === "") return null;
	return { type: "image", data, mimeType: "image/png" };
}

/** summarizeEvents renders the tab's recorded ring for the events verb. */
export function summarizeEvents(output: unknown, max = 40): string[] {
	const events = (output as { events?: unknown })?.events;
	if (!Array.isArray(events)) return [];
	const lines: string[] = [];
	for (const ev of events.slice(-max)) {
		const name = (ev as { event?: unknown })?.event;
		if (typeof name !== "string") continue;
		const params = JSON.stringify((ev as { params?: unknown })?.params ?? {});
		lines.push(`${name} ${params.length > 160 ? params.slice(0, 160) + "…" : params}`);
	}
	const last = (output as { last?: unknown })?.last;
	if (typeof last === "number" && lines.length === 0) lines.push(`nothing since the cursor (last #${last})`);
	return lines;
}
// --- the act verbs' answers --------------------------------------------------

// Found by using the tool with a full grant (2026-09-17): `evaluate`,
// `navigate` and `cdp` all ran, and all three were then rendered by the
// accessibility-tree formatter — the agent got "The page has no accessible
// content" for a call that worked. Each verb answers with its own payload, so
// each gets its own line.

/** summarizeEvaluate renders Runtime.evaluate's answer: the value, or the page's own exception. */
export function summarizeEvaluate(output: unknown): string {
	const result = (output as { result?: unknown })?.result as { value?: unknown; description?: unknown; type?: unknown } | undefined;
	const exception = (output as { exceptionDetails?: unknown })?.exceptionDetails as
		| { text?: unknown; exception?: { description?: unknown } }
		| undefined;
	if (exception) {
		// The page threw: that is an answer about the page, not a tool failure,
		// so it reads as one line instead of an error.
		const detail = exception.exception?.description ?? exception.text;
		return `the page threw: ${typeof detail === "string" ? detail.split("\n")[0] : "an exception"}`;
	}
	if (!result) return "evaluate returned nothing";
	if (result.type === "undefined") return "undefined";
	if (result.value !== undefined) return typeof result.value === "string" ? result.value : JSON.stringify(result.value);
	if (typeof result.description === "string") return result.description;
	return JSON.stringify(result);
}

/** summarizeNavigate renders Page.navigate's answer: where it went, or why it did not. */
export function summarizeNavigate(output: unknown, url = ""): string {
	const error = (output as { errorText?: unknown })?.errorText;
	if (typeof error === "string" && error !== "") return `navigate failed: ${error}`;
	return url ? `navigated to ${url}` : "navigated";
}

/** summarizeCdp renders a raw method's answer as JSON, capped so a big payload cannot flood the context. */
export function summarizeCdp(output: unknown, max = 4000): string {
	let text: string;
	try {
		text = JSON.stringify(output) ?? String(output);
	} catch {
		text = String(output);
	}
	return text.length > max ? `${text.slice(0, max)}… (${text.length - max} more chars)` : text;
}

/** summarizeHistory renders the visits the daemon returned: newest first, one line each. */
export function summarizeHistory(output: unknown, max = 60): string {
	const visits = (output as { visits?: { url?: string; title?: string; visitedAt?: string }[] })?.visits;
	if (!Array.isArray(visits) || visits.length === 0) return "No visits match.";
	const shown = visits.slice(0, max);
	const lines = shown.map((v) => {
		const when = v.visitedAt ? new Date(v.visitedAt).toISOString().replace("T", " ").slice(0, 16) : "";
		const title = v.title && v.title !== v.url ? v.title + " — " : "";
		return `${when}  ${title}${v.url || ""}`.trim();
	});
	const extra = visits.length > shown.length ? `, ${visits.length - shown.length} more` : "";
	return lines.join("\n") + `\n(${visits.length} visit${visits.length === 1 ? "" : "s"}${extra})`;
}
