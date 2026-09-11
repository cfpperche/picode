/**
 * Live browser frames while any `agent_browser` tool call runs — as a
 * standalone extension (ADR-0080).
 *
 * This extension does not patch or wrap pi-agent-browser-native. It observes
 * pi's own tool lifecycle events (`tool_execution_start` / `tool_execution_end`
 * for the `agent_browser` tool), locates the session's existing upstream
 * agent-browser stream rendezvous (the same `<session>.stream` + `.pid` files
 * upstream 0.36 already writes beside its socket), and mirrors the latest
 * bounded frame into `<pi-session-file>.capture/` as atomic
 * `current.jpg` + `current.json` files. A host (PiCode's daemon) watches that
 * directory only while a browser tool runs and forwards frames to the UI feed.
 *
 * Consent model, identical in spirit to the superseded in-package emitter:
 * `/browser-captures on|off|status` or the `--browser-captures` flag. The
 * choice is branch-persisted with `pi.appendEntry` and restored on session
 * start/tree. When enabled, each completed capture call also appends a
 * `browser-capture-final` entry so hosts can replay the last frame after a
 * reload. Frames may contain page secrets — the command copy says so.
 *
 * The upstream session name is re-derived with the same algorithm
 * pi-agent-browser-native 0.6.6 uses (`piab-<slug>-<sessionHash>-<cwdHash>`,
 * plus `-fresh-*` rotations matched by prefix). Reading is strictly local:
 * loopback WebSocket, uid/mode checks, O_NOFOLLOW, pid liveness. This
 * extension never launches a browser, sends input events, or enables a server.
 */

import { createHash } from "node:crypto";
import { constants } from "node:fs";
import { lstat, mkdir, open, readdir, readFile, realpath, rename, rm, writeFile } from "node:fs/promises";
import { basename, join, resolve } from "node:path";
import { setTimeout as delay } from "node:timers/promises";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";

const CONSENT_ENTRY = "browser-capture-consent";
const FINAL_ENTRY = "browser-capture-final";
const CAPTURE_MAX_BYTES = 200 * 1024;
const READY_TIMEOUT_MS = 8000;
const FINISH_DRAIN_MS = 1200;

// --- Upstream session naming (mirrors pi-agent-browser-native 0.6.6 runtime.ts) ---

const MANAGED_SESSION_NAME_PREFIX = "piab-";
const MAX_PROJECT_SLUG_LENGTH = 24;
const SESSION_NAME_CWD_HASH_LENGTH = 8;
const SESSION_NAME_SESSION_ID_LENGTH = 12;

function createCwdHash(cwd: string): string {
	return createHash("sha256").update(`cwd:${cwd}`).digest("hex").slice(0, SESSION_NAME_CWD_HASH_LENGTH);
}

export function createImplicitSessionName(sessionId: string, cwd: string, platform: NodeJS.Platform = process.platform): string {
	const normalizedSessionId = sessionId.replaceAll("-", "").toLowerCase();
	const slug =
		basename(cwd)
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, "-")
			.replace(/^-+|-+$/g, "")
			.slice(0, MAX_PROJECT_SLUG_LENGTH) || "project";
	const cwdHash = createCwdHash(cwd);
	const stableSessionId = createHash("sha256")
		.update(`session:${normalizedSessionId}`)
		.digest("hex")
		.slice(0, SESSION_NAME_SESSION_ID_LENGTH);
	if (platform === "android") {
		const identity = `session:${normalizedSessionId}:cwd:${cwd}`;
		const digest = createHash("sha256").update(identity).digest("hex").slice(0, SESSION_NAME_SESSION_ID_LENGTH);
		return `${MANAGED_SESSION_NAME_PREFIX}${digest}`;
	}
	return `${MANAGED_SESSION_NAME_PREFIX}${slug}-${stableSessionId}-${cwdHash}`;
}

// --- Upstream socket rendezvous (mirrors the patched capture-channel checks) ---

function socketRoot(env: NodeJS.ProcessEnv = process.env): string | undefined {
	const override = env.PI_AGENT_BROWSER_SOCKET_DIR;
	if (override) return override;
	if (process.platform === "win32") return undefined;
	if (process.platform === "android") return undefined;
	const prefix = process.platform === "darwin" ? "/private/tmp/piab" : "/tmp/piab";
	const uid = typeof process.getuid === "function" ? process.getuid() : undefined;
	return `${prefix}${uid !== undefined ? `-${uid}` : ""}`;
}

const safeSegment = (value: string) => value !== "." && value !== ".." && basename(value) === value && !value.includes("\\");

export async function findCapturePort(sessionBase: string, namespace = "", env: NodeJS.ProcessEnv = process.env): Promise<number | undefined> {
	const root = socketRoot(env);
	if (!root || !safeSegment(sessionBase) || (namespace && !safeSegment(namespace))) return;
	try {
		if (await realpath(root) !== resolve(root)) return;
		const st = await lstat(root);
		const uid = typeof process.getuid === "function" ? process.getuid() : undefined;
		if (!st.isDirectory() || st.isSymbolicLink() || (uid !== undefined && (st.uid !== uid || (st.mode & 0o777) !== 0o700))) return;
		const dir = namespace ? join(root, "namespaces", namespace, "run") : root;
		if (await realpath(dir) !== resolve(dir)) return;
		// The base name plus its `-fresh-` rotations (fresh names embed a
		// per-process seed we cannot recompute from outside the wrapper).
		const entries = await readdir(dir);
		const candidates = entries
			.filter(name => name.endsWith(".stream"))
			.map(name => name.slice(0, -".stream".length))
			.filter(name => name === sessionBase || name.startsWith(`${sessionBase}-fresh-`));
		const stats = await Promise.all(candidates.map(async name => ({ name, st: await lstat(join(dir, `${name}.stream`)).catch(() => undefined) })));
		stats.sort((a, b) => (b.st?.mtimeMs ?? 0) - (a.st?.mtimeMs ?? 0));
		for (const { name } of stats) {
			try {
				const numberFile = async (suffix: string) => {
					const file = await open(join(dir, `${name}.${suffix}`), constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
					try {
						const info = await file.stat();
						if (!info.isFile() || info.size > 16 || (uid !== undefined && info.uid !== uid)) return;
						const data = await file.readFile("utf8");
						if (!/^\d+\s*$/.test(data)) return;
						return Number(data.trim());
					} finally { await file.close(); }
				};
				const pid = await numberFile("pid");
				const port = await numberFile("stream");
				if (!pid || !port || !Number.isInteger(port) || port > 65535) continue;
				process.kill(pid, 0);
				return port;
			} catch { /* Try the next candidate. */ }
		}
	} catch { /* The rendezvous is optional; absence is normal before first launch. */ }
	return;
}

// --- Bounded frames (mirrors the superseded emitter's image policy) ---

export const caption = (value: unknown, max = 120): string =>
	typeof value === "string" ? value.replace(/[\u0000-\u001f\u007f]/g, " ").trim().slice(0, max) : "";

export function captionUrl(value: unknown): string | undefined {
	try {
		const u = new URL(String(value));
		return /^https?:$/.test(u.protocol) ? (u.origin + u.pathname).slice(0, 320) : undefined;
	} catch { return undefined; }
}

export function boundedJpeg(data: unknown): string | undefined {
	if (typeof data !== "string" || data.length > Math.ceil(CAPTURE_MAX_BYTES / 3) * 4 || !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(data)) return;
	const bytes = Buffer.from(data, "base64");
	if (bytes.length > CAPTURE_MAX_BYTES || bytes.length < 12 || bytes[0] !== 255 || bytes[1] !== 216 || bytes.at(-2) !== 255 || bytes.at(-1) !== 217) return;
	for (let pos = 2; pos + 4 <= bytes.length;) {
		if (bytes[pos++] !== 255) return;
		while (bytes[pos] === 255) pos++;
		const marker = bytes[pos++];
		if (marker === undefined || marker === 218 || marker === 217 || pos + 2 > bytes.length) return;
		const size = bytes.readUInt16BE(pos);
		if (size < 2 || pos + size > bytes.length) return;
		if ([192, 193, 194].includes(marker)) {
			if (size < 8) return;
			const height = bytes.readUInt16BE(pos + 3);
			const width = bytes.readUInt16BE(pos + 5);
			if (width && height && width <= 1600 && height <= 1600 && width * height <= 1_600_000) return `data:image/jpeg;base64,${data}`;
			return;
		}
		pos += size;
	}
}

const isRecord = (value: unknown): value is Record<string, unknown> => typeof value === "object" && value !== null && !Array.isArray(value);

// --- Frame store: atomic latest-wins files beside the pi session file ---

export class FrameStore {
	#dir: string;
	#toolCallId = "";
	#seq = 0;

	constructor(sessionFile: string) { this.#dir = `${sessionFile}.capture`; }

	get dir() { return this.#dir; }

	async begin(toolCallId: string): Promise<void> {
		this.#toolCallId = toolCallId;
		this.#seq = 0;
		try {
			await mkdir(this.#dir, { recursive: true, mode: 0o700 });
			await rm(join(this.#dir, "current.json"), { force: true });
			await this.#writeJson("state.json", { capturing: true, toolCallId, startedAt: Date.now() });
		} catch { /* A read-only session dir disables capture; never fail the tool. */ }
	}

	async accept(frame: { image: string; ts: number; url?: string; title?: string; source?: string }): Promise<void> {
		if (!this.#toolCallId) return;
		try {
			const jpg = join(this.#dir, "current.jpg.tmp");
			await rm(jpg, { force: true });
			await writeFile(jpg, Buffer.from(frame.image.split(",")[1] ?? "", "base64"), { mode: 0o600 });
			const image = join(this.#dir, "current.jpg");
			await rename(jpg, image);
			await this.#writeJson("current.json", { toolCallId: this.#toolCallId, seq: ++this.#seq, ts: frame.ts, final: false, bytes: Buffer.byteLength(frame.image), url: frame.url, title: frame.title, source: frame.source });
		} catch { /* Never fail the tool for a capture write. */ }
	}

	async finish(): Promise<void> {
		try {
			// Claim a final frame only when this capture actually took one; a
			// frameless call (e.g. `close`) must not describe the previous
			// call's image as its own.
			if (this.#seq > 0) {
				await this.#writeJson("current.json", { toolCallId: this.#toolCallId, seq: this.#seq, ts: Date.now(), final: true, bytes: 0 });
			} else {
				await rm(join(this.#dir, "current.json"), { force: true });
			}
			await rm(join(this.#dir, "state.json"), { force: true });
		} catch { /* Leave whatever landed; the host times out stale state. */ }
		this.#toolCallId = "";
	}

	async #writeJson(name: string, value: unknown): Promise<void> {
		const tmp = join(this.#dir, `${name}.tmp`);
		await writeFile(tmp, JSON.stringify(value), { mode: 0o600 });
		await rename(tmp, join(this.#dir, name));
	}
}

// --- Stream observer (mirrors the superseded emitter's observer) ---

interface ActiveCapture {
	toolCallId: string;
	store: FrameStore;
	stop: () => void;
	finish: () => Promise<{ image?: string; ts?: number; url?: string; title?: string }>;
}

function observeCaptures(options: { sessionBase: string; namespace?: string; store: FrameStore; signal: AbortSignal }): ActiveCapture {
	const controller = new AbortController();
	let socket: WebSocket | undefined;
	let stopped = false, finishing = false, sawFrame = false;
	let url: string | undefined, title: string | undefined, tab = "";
	let last: { image?: string; ts?: number; url?: string; title?: string } = {};
	const stop = () => {
		if (stopped) return;
		stopped = true;
		controller.abort();
		options.signal.removeEventListener("abort", stop);
		try { socket?.close(); } catch { /* Already closed. */ }
	};
	options.signal.addEventListener("abort", stop, { once: true });
	if (options.signal.aborted) stop();
	const send = (value: unknown) => { if (!stopped && socket?.readyState === WebSocket.OPEN) socket.send(JSON.stringify(value)); };
	// Push pacing with a per-client frame cap (URL query covers the opening
	// frame). Ack pacing deadlocks a client that attaches before the browser
	// exists: the opening frame counts as in-flight and every later frame
	// replaces it until an ack arrives, but acks can only echo a frame we
	// received. Loopback push with maxFps bounds the rate without that hazard.
	const setFps = (fps: number) => send({ type: "config", maxFps: fps });
	const ready = (async () => {
		const deadline = Date.now() + READY_TIMEOUT_MS;
		while (!stopped && Date.now() < deadline) {
			const port = await findCapturePort(options.sessionBase, options.namespace);
			if (port) {
				if (stopped) return;
				socket = new WebSocket(`ws://127.0.0.1:${port}/?maxFps=1`);
				socket.onopen = () => setFps(1);
				socket.onmessage = event => {
					if (stopped) return;
					if (typeof event.data !== "string" || event.data.length > Math.ceil(CAPTURE_MAX_BYTES / 3) * 4 + 16_384) return;
					try {
						const message: unknown = JSON.parse(event.data);
						if (!isRecord(message)) return;
						if (message.type === "tabs" && Array.isArray(message.tabs)) {
							const active = message.tabs.find((t: unknown) => isRecord(t) && t.active === true);
							if (isRecord(active)) { url = captionUrl(active.url); title = caption(active.title); tab = caption(active.tabId, 24); }
						} else if (message.type === "url") { url = captionUrl(message.url); title = undefined; }
						else if (message.type === "frame") {
							const image = boundedJpeg(message.data);
							if (image) {
								sawFrame = true;
								last = { image, ts: Date.now(), url, title };
								void options.store.accept({ image, ts: last.ts, url, title, source: caption(`agent-browser / ${options.sessionBase} / ${tab}`) });
							}
						}
					} catch { /* Skip malformed messages. */ }
				};
				return;
			}
			await delay(250, undefined, { signal: controller.signal }).catch(() => {});
		}
	})().catch(() => {});
	const finish = async () => {
		if (!stopped) {
			finishing = true;
			setFps(10);
			const deadline = Date.now() + FINISH_DRAIN_MS;
			try {
				do { await delay(100, undefined, { signal: controller.signal }); } while (!sawFrame && !stopped && Date.now() < deadline);
				await delay(250, undefined, { signal: controller.signal });
			} catch { /* Aborted. */ }
		}
		stop();
		await ready;
		return last;
	};
	return { stop, finish };
}

// --- Extension wiring ---

export default function (pi: ExtensionAPI) {
	let enabled = false;
	let active: (ActiveCapture & { toolCallId: string }) | undefined;
	let controller = new AbortController();

	function reset(value: boolean) {
		controller.abort();
		controller = new AbortController();
		enabled = value;
	}

	function restore(ctx: ExtensionContext) {
		let value = pi.getFlag("browser-captures") === true;
		for (const entry of ctx.sessionManager.getBranch()) {
			if (entry.type === "custom" && entry.customType === CONSENT_ENTRY && isRecord(entry.data) && typeof entry.data.enabled === "boolean") value = entry.data.enabled;
		}
		reset(value);
	}

	pi.registerFlag("browser-captures", { type: "boolean", default: false, description: "Opt in to live browser frames streamed to the PiCode server and a final frame saved in session history" });
	pi.registerCommand("browser-captures", {
		description: "Browser capture status, on, or off (frames stream to the server; final frame persists in session history)",
		handler: async (args, ctx) => {
			const action = args.trim();
			if (action === "on" || action === "off") {
				reset(action === "on");
				pi.appendEntry(CONSENT_ENTRY, { enabled });
			} else if (action && action !== "status") {
				ctx.ui.notify("Use /browser-captures on, off, or status.", "warning");
				return;
			}
			ctx.ui.notify(enabled
				? "Browser captures on. Frames may contain secrets: they stream to the PiCode server during browser calls and the final frame is saved in session history."
				: "Browser captures off. Existing saved captures are not deleted.", enabled ? "warning" : "info");
		},
	});

	pi.on("session_start", (_event, ctx) => restore(ctx));
	pi.on("session_tree", (_event, ctx) => restore(ctx));
	pi.on("session_shutdown", () => { enabled = false; active?.stop(); active = undefined; controller.abort(); controller = new AbortController(); });

	pi.on("tool_execution_start", async (event, ctx) => {
		if (!enabled || event.toolName !== "agent_browser" || active) return;
		const sessionFile = ctx.sessionManager.getSessionFile();
		if (!sessionFile) return; // Ephemeral sessions have nowhere to persist frames.
		const sessionId = ctx.sessionManager.getSessionId();
		if (!sessionId) return;
		const store = new FrameStore(sessionFile);
		await store.begin(event.toolCallId);
		const observed = observeCaptures({ sessionBase: createImplicitSessionName(sessionId, ctx.cwd), namespace: process.env.AGENT_BROWSER_NAMESPACE ?? "", store, signal: controller.signal });
		active = { toolCallId: event.toolCallId, store, ...observed };
	});

	pi.on("tool_execution_end", async event => {
		if (!active || event.toolCallId !== active.toolCallId) return;
		const running = active;
		active = undefined;
		const last = await running.finish();
		await running.store.finish();
		if (last.image) pi.appendEntry(FINAL_ENTRY, { toolCallId: event.toolCallId, image: last.image, ts: last.ts, url: last.url, title: last.title, savedAt: Date.now() });
	});
}
