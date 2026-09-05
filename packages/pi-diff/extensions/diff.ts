/**
 * Side diff panel for the pi TUI (ADR-0077).
 *
 * `/diff` opens a right-hand overlay: every file changed against HEAD
 * (tracked and untracked) with its +/- counts, then the hunks of the file
 * the agent touched last. The panel never takes keyboard focus — the
 * editor keeps it — and refreshes itself after each edit, write or shell
 * call and at the end of every turn. `/diff` again hides it. The footer
 * always carries the running total (`diff +12 -3`).
 *
 * The overlay is shown straight through the TUI (`tui.showOverlay`), not
 * through `ctx.ui.custom()`: a custom component is a UI prompt for pi's
 * lifecycle events, and a panel that stays open all session would report
 * "waiting for user" to every host watching those events (ADR-0056).
 */

import { execFile } from "node:child_process";
import { readFile } from "node:fs/promises";
import { isAbsolute, join, relative, resolve, sep } from "node:path";
import type { ExtensionAPI, ExtensionContext, Theme } from "@earendil-works/pi-coding-agent";
import { truncateToWidth, visibleWidth, type Component, type OverlayHandle, type TUI } from "@earendil-works/pi-tui";
import {
	MAX_COUNT_BYTES,
	clampScroll,
	countLines,
	cycleFocus,
	extractToolPath,
	isBinary,
	layout,
	mergeChanges,
	parseCommand,
	parseNumstat,
	parseUnifiedDiff,
	parseUntracked,
	pickFocus,
	renameParts,
	resolveFocusArg,
	shouldRefresh,
	statusText,
	untrackedChange,
	type DiffLine,
	type FileChange,
	type Model,
	type Paint,
} from "../src/logic.ts";

const KEY = "pi-diff";
const REFRESH_DELAY_MS = 150;
const SCROLL_STEP = 3;

type UI = ExtensionContext["ui"];

// ---- git ----

function git(cwd: string, args: string[], okExit: number[] = [0]): Promise<string> {
	return new Promise((resolve, reject) => {
		execFile("git", ["-c", "core.quotepath=false", ...args], { cwd, maxBuffer: 16 * 1024 * 1024, windowsHide: true }, (err, stdout) => {
			const code = (err as { code?: unknown } | null)?.code;
			if (!err || (typeof code === "number" && okExit.includes(code))) resolve(String(stdout));
			else reject(err);
		});
	});
}

async function repoRoot(cwd: string): Promise<string | null> {
	try {
		return (await git(cwd, ["rev-parse", "--show-toplevel"])).trim() || null;
	} catch {
		return null;
	}
}

async function hasHead(root: string): Promise<boolean> {
	try {
		await git(root, ["rev-parse", "--verify", "-q", "HEAD"]);
		return true;
	} catch {
		return false;
	}
}

async function untrackedInfo(root: string, path: string): Promise<{ lines: number; binary: boolean }> {
	try {
		const bytes = await readFile(join(root, path));
		if (isBinary(bytes)) return { lines: 0, binary: true };
		if (bytes.length > MAX_COUNT_BYTES) return { lines: 0, binary: false };
		return { lines: countLines(bytes.toString("utf8")), binary: false };
	} catch {
		return { lines: 0, binary: false };
	}
}

async function fileDiff(root: string, file: FileChange): Promise<DiffLine[]> {
	if (file.binary) return [];
	try {
		if (file.untracked) {
			// --no-index exits 1 when the sides differ, which is the point.
			return parseUnifiedDiff(await git(root, ["diff", "--no-index", "--no-color", "--", "/dev/null", file.path], [0, 1]));
		}
		const { paths } = renameParts(file.path);
		return parseUnifiedDiff(await git(root, ["diff", "HEAD", "--no-color", "-M", "--", ...paths]));
	} catch {
		return [];
	}
}

/** A tool's path (relative to the session cwd, or absolute) as git names it: relative to the root, forward slashes. */
function rootRelative(root: string, cwd: string, p: string | undefined): string | undefined {
	if (!p) return undefined;
	const abs = isAbsolute(p) ? p : resolve(cwd, p);
	const rel = relative(root, abs);
	if (!rel || rel.startsWith("..")) return undefined;
	return sep === "/" ? rel : rel.split(sep).join("/");
}

/** Everything the panel shows, in one pass; `unavailable` when git cannot answer. */
async function collect(cwd: string, preferredRaw: string | undefined, current: string | undefined): Promise<Model> {
	const root = await repoRoot(cwd);
	if (!root) return { files: [], focus: undefined, diff: [], unavailable: true };
	const preferred = rootRelative(root, cwd, preferredRaw);
	const head = await hasHead(root);
	const [numstat, status] = await Promise.all([
		head ? git(root, ["diff", "--numstat", "-M", "HEAD", "--"]).catch(() => "") : Promise.resolve(""),
		git(root, ["status", "--porcelain", "--untracked-files=all"]).catch(() => ""),
	]);
	const untrackedPaths = parseUntracked(status);
	const untracked = await Promise.all(untrackedPaths.map(async (p) => untrackedChange(p, await untrackedInfo(root, p))));
	const files = mergeChanges(parseNumstat(numstat), untracked);
	const focus = pickFocus(files, preferred, current);
	const focused = files.find((f) => f.path === focus);
	const diff = focused ? await fileDiff(root, focused) : [];
	return { files, focus, diff };
}

// ---- TUI ----

function painter(theme: Theme): Paint {
	return (kind, text) => {
		switch (kind) {
			case "title":
				return theme.bold(theme.fg("text", text));
			case "add":
				return theme.fg("toolDiffAdded", text);
			case "del":
				return theme.fg("toolDiffRemoved", text);
			case "ctx":
				return theme.fg("toolDiffContext", text);
			case "hunk":
				return theme.fg("accent", text);
			case "accent":
				return theme.fg("accent", text);
			case "path":
				return theme.fg("text", text);
			case "border":
				return theme.fg("borderMuted", text);
			case "meta":
				return theme.fg("warning", text);
			default:
				return theme.fg("muted", text);
		}
	};
}

class DiffPanel implements Component {
	maxScroll = 0;
	constructor(
		private readonly tui: TUI,
		private readonly paint: Paint,
		private readonly state: { model: Model | null; scroll: number },
	) {}

	render(width: number): string[] {
		const inner = Math.max(10, width - 2);
		// Leave the editor and footer visible under the panel.
		const height = Math.max(8, this.tui.terminal.rows - 7);
		const model = this.state.model ?? { files: [], focus: undefined, diff: [], unavailable: false };
		const out = layout({ model, scroll: this.state.scroll, width: inner, height }, this.paint, visibleWidth);
		this.maxScroll = out.maxScroll;
		this.state.scroll = clampScroll(this.state.scroll, out.maxScroll);
		const border = this.paint("border", "│");
		return out.lines.map((line) => `${border} ${truncateToWidth(line, inner)}`);
	}

	invalidate(): void {}
}

/** The widget pi owns for us: zero lines above the editor, and the hook that hides the overlay when pi removes it. */
class Anchor implements Component {
	constructor(private readonly onDispose: () => void) {}
	render(): string[] {
		return [];
	}
	invalidate(): void {}
	dispose(): void {
		this.onDispose();
	}
}

export default function piDiff(pi: ExtensionAPI) {
	const state = { model: null as Model | null, scroll: 0 };
	let ui: UI | null = null;
	let cwd = process.cwd();
	let open = false;
	let tui: TUI | null = null;
	let handle: OverlayHandle | null = null;
	let panel: DiffPanel | null = null;
	let preferred: string | undefined; // the file the agent touched last
	let timer: ReturnType<typeof setTimeout> | null = null;
	let running: Promise<void> | null = null;
	let again = false;

	const paintStatus = () => {
		if (!ui) return;
		const text = statusText(state.model);
		ui.setStatus(KEY, text === undefined ? undefined : open ? `${text} · /diff hides` : `${text} · /diff`);
	};

	const redraw = () => {
		paintStatus();
		if (open && tui) tui.requestRender();
	};

	async function refresh(): Promise<void> {
		if (running) {
			again = true;
			return running;
		}
		running = (async () => {
			try {
				state.model = await collect(cwd, preferred, state.model?.focus);
			} catch {
				state.model = { files: [], focus: undefined, diff: [], unavailable: true };
			}
			redraw();
		})();
		try {
			await running;
		} finally {
			running = null;
			if (again) {
				again = false;
				void refresh();
			}
		}
	}

	const schedule = () => {
		if (timer) clearTimeout(timer);
		timer = setTimeout(() => {
			timer = null;
			void refresh();
		}, REFRESH_DELAY_MS);
	};

	const show = (ctx: ExtensionContext) => {
		if (open) return;
		if (ctx.mode !== "tui") {
			ctx.ui.notify("pi-diff: the side panel needs the terminal TUI; the footer total still updates.", "warning");
			return;
		}
		ctx.ui.setWidget(KEY, (t, theme) => {
			tui = t;
			panel = new DiffPanel(t, painter(theme), state);
			handle = t.showOverlay(panel, {
				nonCapturing: true,
				anchor: "top-right",
				width: "45%",
				minWidth: 44,
				maxHeight: "100%",
				margin: { top: 1, right: 0 },
				visible: (termWidth) => termWidth >= 100,
			});
			return new Anchor(() => {
				handle?.hide();
				handle = null;
				panel = null;
			});
		});
		open = true;
		state.scroll = 0;
		void refresh();
	};

	const hide = (ctx: ExtensionContext) => {
		if (!open) return;
		ctx.ui.setWidget(KEY, undefined);
		open = false;
		paintStatus();
	};

	const scrollBy = (delta: number) => {
		if (!open || !panel) return;
		state.scroll = clampScroll(state.scroll + delta, panel.maxScroll);
		tui?.requestRender();
	};

	pi.registerCommand("diff", {
		description: "Side panel with the files changed against HEAD and the hunks of the focused file — /diff <path> | next | prev | top | off",
		getArgumentCompletions: (prefix) => {
			const files = state.model?.files ?? [];
			const words = ["next", "prev", "top", "off", ...files.map((f) => renameParts(f.path).display)];
			const items = words.filter((w) => w.startsWith(prefix)).map((w) => ({ value: w, label: w }));
			return items.length ? items : null;
		},
		handler: async (args, ctx) => {
			ui = ctx.ui;
			cwd = ctx.cwd || cwd;
			const cmd = parseCommand(args);
			switch (cmd.kind) {
				case "toggle":
					if (open) hide(ctx);
					else show(ctx);
					return;
				case "open":
					show(ctx);
					return;
				case "close":
					hide(ctx);
					return;
				case "top":
					state.scroll = 0;
					tui?.requestRender();
					return;
				case "next":
				case "prev": {
					const files = state.model?.files ?? [];
					const next = cycleFocus(files, state.model?.focus, cmd.kind === "next" ? 1 : -1);
					if (!next) {
						ctx.ui.notify("pi-diff: no changed files.", "info");
						return;
					}
					preferred = next;
					state.scroll = 0;
					if (!open) show(ctx);
					await refresh();
					return;
				}
				case "focus": {
					const files = state.model?.files ?? [];
					const target = resolveFocusArg(files, cmd.value);
					if (!target) {
						ctx.ui.notify(`pi-diff: no changed file matches "${cmd.value}".`, "warning");
						return;
					}
					preferred = target;
					state.scroll = 0;
					if (!open) show(ctx);
					await refresh();
					return;
				}
			}
		},
	});

	// alt+up/down and alt+j/k are pi's own (model reorder, editor cursor); these are free.
	pi.registerShortcut("alt+n", { description: "pi-diff: scroll the panel down", handler: () => scrollBy(SCROLL_STEP) });
	pi.registerShortcut("alt+u", { description: "pi-diff: scroll the panel up", handler: () => scrollBy(-SCROLL_STEP) });
	pi.registerShortcut("alt+pageDown", { description: "pi-diff: page the panel down", handler: () => scrollBy(SCROLL_STEP * 8) });
	pi.registerShortcut("alt+pageUp", { description: "pi-diff: page the panel up", handler: () => scrollBy(-SCROLL_STEP * 8) });

	pi.on("session_start", async (_event, ctx) => {
		ui = ctx.ui;
		cwd = ctx.cwd || cwd;
		preferred = undefined;
		schedule();
	});

	pi.on("tool_execution_start", async (event) => {
		const p = extractToolPath(event.toolName, event.args);
		if (p) preferred = p;
	});

	pi.on("tool_execution_end", async (event) => {
		if (shouldRefresh(event.toolName)) schedule();
	});

	pi.on("turn_end", async () => schedule());
	pi.on("agent_end", async () => schedule());
}
