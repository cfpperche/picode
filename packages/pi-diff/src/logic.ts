/**
 * Pure logic for pi-diff (ADR-0076). No pi imports — node:test covers
 * this. The extension file (extensions/diff.ts) is I/O glue: it runs git,
 * owns the overlay and feeds these functions what they lay out.
 */

export interface FileChange {
	/** Path as git prints it (a rename keeps `old => new`). */
	readonly path: string;
	readonly added: number;
	readonly removed: number;
	readonly untracked: boolean;
	readonly binary: boolean;
}

export type DiffLineKind = "hunk" | "add" | "del" | "ctx" | "meta";

export interface DiffLine {
	readonly kind: DiffLineKind;
	readonly text: string;
	readonly oldNo?: number;
	readonly newNo?: number;
}

/** Tools whose end means the working tree may have changed. A closed allowlist: unknown tools never trigger git. */
export const MUTATING_TOOLS: ReadonlySet<string> = new Set(["bash", "powershell", "edit", "write", "multiedit"]);

export function shouldRefresh(toolName: string): boolean {
	return MUTATING_TOOLS.has(toolName);
}

/** The file an edit tool call names, from its args; undefined for shell tools. */
export function extractToolPath(toolName: string, args: unknown): string | undefined {
	if (toolName !== "edit" && toolName !== "write" && toolName !== "multiedit") return undefined;
	const p = (args && typeof args === "object" ? (args as { path?: unknown }).path : undefined) as unknown;
	return typeof p === "string" && p.trim() ? p.trim() : undefined;
}

/** Longest diff we lay out; past this the panel says so instead of rendering it. */
export const MAX_DIFF_LINES = 4000;
/** Untracked files larger than this are not counted line by line. */
export const MAX_COUNT_BYTES = 1 << 20;

// ---- git output ----

/** `git diff --numstat`: `added\tremoved\tpath`, binary as `-\t-\tpath`. */
export function parseNumstat(text: string): FileChange[] {
	const out: FileChange[] = [];
	for (const raw of text.split("\n")) {
		const line = raw.replace(/\r$/, "");
		if (!line) continue;
		const parts = line.split("\t");
		if (parts.length < 3) continue;
		const path = unquote(parts.slice(2).join("\t"));
		if (!path) continue;
		const binary = parts[0] === "-" || parts[1] === "-";
		out.push({
			path,
			added: binary ? 0 : toCount(parts[0]!),
			removed: binary ? 0 : toCount(parts[1]!),
			untracked: false,
			binary,
		});
	}
	return out;
}

/** `git status --porcelain --untracked-files=all`: the `??` rows. */
export function parseUntracked(porcelain: string): string[] {
	const out: string[] = [];
	for (const raw of porcelain.split("\n")) {
		const line = raw.replace(/\r$/, "");
		if (!line.startsWith("?? ")) continue;
		const path = unquote(line.slice(3));
		if (path) out.push(path);
	}
	return out;
}

/** Git quotes paths with special characters: `"a b"`, with C escapes inside. */
export function unquote(path: string): string {
	const t = path.trim();
	if (t.length < 2 || !t.startsWith('"') || !t.endsWith('"')) return t;
	// Escapes are bytes of the UTF-8 name, so decode them together.
	const bytes: number[] = [];
	const body = t.slice(1, -1);
	for (let i = 0; i < body.length; i++) {
		const ch = body[i]!;
		if (ch !== "\\" || i === body.length - 1) {
			bytes.push(...new TextEncoder().encode(ch));
			continue;
		}
		const next = body[i + 1]!;
		if (/[0-7]/.test(next) && /^[0-7]{3}$/.test(body.slice(i + 1, i + 4))) {
			bytes.push(parseInt(body.slice(i + 1, i + 4), 8));
			i += 3;
			continue;
		}
		const map: Record<string, string> = { t: "\t", n: "\n", r: "\r", '"': '"', "\\": "\\", a: "\x07", b: "\b", f: "\f", v: "\v" };
		bytes.push(...new TextEncoder().encode(map[next] ?? next));
		i += 1;
	}
	return new TextDecoder().decode(new Uint8Array(bytes));
}

function toCount(s: string): number {
	const n = Number.parseInt(s, 10);
	return Number.isFinite(n) && n >= 0 ? n : 0;
}

/** Lines in a text file, the way `git diff --numstat` counts them (a trailing newline adds none). */
export function countLines(text: string): number {
	if (!text) return 0;
	let n = 0;
	for (let i = 0; i < text.length; i++) if (text.charCodeAt(i) === 10) n++;
	return text.endsWith("\n") ? n : n + 1;
}

export function isBinary(bytes: Uint8Array): boolean {
	const n = Math.min(bytes.length, 8000);
	for (let i = 0; i < n; i++) if (bytes[i] === 0) return true;
	return false;
}

export function untrackedChange(path: string, info: { lines: number; binary: boolean }): FileChange {
	return { path, added: info.binary ? 0 : info.lines, removed: 0, untracked: true, binary: info.binary };
}

/** Tracked and untracked changes, one row per path, sorted. */
export function mergeChanges(tracked: readonly FileChange[], untracked: readonly FileChange[]): FileChange[] {
	const byPath = new Map<string, FileChange>();
	for (const f of tracked) byPath.set(f.path, f);
	for (const f of untracked) if (!byPath.has(f.path)) byPath.set(f.path, f);
	return [...byPath.values()].sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));
}

export interface Totals {
	files: number;
	added: number;
	removed: number;
}

export function totals(files: readonly FileChange[]): Totals {
	let added = 0;
	let removed = 0;
	for (const f of files) {
		added += f.added;
		removed += f.removed;
	}
	return { files: files.length, added, removed };
}

/** A rename as numstat prints it: `dir/{old => new}/f` or `old => new`. */
export function renameParts(path: string): { display: string; paths: string[] } {
	const braced = /^(.*)\{(.*) => (.*)\}(.*)$/.exec(path);
	if (braced) {
		const [, pre, oldMid, newMid, post] = braced;
		const oldPath = `${pre}${oldMid}${post}`;
		const newPath = `${pre}${newMid}${post}`;
		return { display: newPath, paths: [oldPath, newPath] };
	}
	const plain = /^(.+) => (.+)$/.exec(path);
	if (plain) return { display: plain[2]!, paths: [plain[1]!, plain[2]!] };
	return { display: path, paths: [path] };
}

// ---- focus ----

/** Which file the hunks show: the file the agent touched last, else the one already shown, else the first. */
export function pickFocus(files: readonly FileChange[], preferred: string | undefined, current: string | undefined): string | undefined {
	if (files.length === 0) return undefined;
	const has = (p: string | undefined) => p !== undefined && files.some((f) => f.path === p || renameParts(f.path).display === p);
	if (has(preferred)) return files.find((f) => f.path === preferred || renameParts(f.path).display === preferred)!.path;
	if (has(current)) return current;
	return files[0]!.path;
}

export function cycleFocus(files: readonly FileChange[], current: string | undefined, dir: 1 | -1): string | undefined {
	if (files.length === 0) return undefined;
	const i = files.findIndex((f) => f.path === current);
	if (i < 0) return dir === 1 ? files[0]!.path : files[files.length - 1]!.path;
	return files[(i + dir + files.length) % files.length]!.path;
}

/** `/diff <arg>`: an exact path, else the unique path ending with it, else the unique path containing it. */
export function resolveFocusArg(files: readonly FileChange[], arg: string): string | undefined {
	const q = arg.trim().replace(/^\.\//, "");
	if (!q) return undefined;
	const exact = files.find((f) => f.path === q || renameParts(f.path).display === q);
	if (exact) return exact.path;
	const tail = files.filter((f) => renameParts(f.path).display.endsWith("/" + q) || f.path.endsWith("/" + q));
	if (tail.length === 1) return tail[0]!.path;
	const part = files.filter((f) => f.path.includes(q));
	if (part.length === 1) return part[0]!.path;
	return undefined;
}

export type Command =
	| { kind: "toggle" }
	| { kind: "open" }
	| { kind: "close" }
	| { kind: "next" }
	| { kind: "prev" }
	| { kind: "top" }
	| { kind: "focus"; value: string };

/** The words after `/diff`. */
export function parseCommand(args: string): Command {
	const a = (args || "").trim();
	if (!a) return { kind: "toggle" };
	const word = a.toLowerCase();
	if (word === "on" || word === "open" || word === "show") return { kind: "open" };
	if (word === "off" || word === "close" || word === "hide") return { kind: "close" };
	if (word === "next" || word === "n") return { kind: "next" };
	if (word === "prev" || word === "previous" || word === "p") return { kind: "prev" };
	if (word === "top") return { kind: "top" };
	return { kind: "focus", value: a };
}

// ---- unified diff ----

/** One file's `git diff` as lines with numbers; the file header is dropped, hunk headers stay. */
export function parseUnifiedDiff(text: string): DiffLine[] {
	const out: DiffLine[] = [];
	let oldNo = 0;
	let newNo = 0;
	let inHunk = false;
	for (const raw of text.split("\n")) {
		const line = raw.replace(/\r$/, "");
		if (line.startsWith("@@")) {
			const m = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/.exec(line);
			oldNo = m ? Number.parseInt(m[1]!, 10) : 0;
			newNo = m ? Number.parseInt(m[2]!, 10) : 0;
			inHunk = true;
			out.push({ kind: "hunk", text: line });
			continue;
		}
		if (!inHunk) continue; // diff --git, index, ---, +++
		if (line.startsWith("\\")) {
			out.push({ kind: "meta", text: line.slice(1).trim() });
			continue;
		}
		if (line.startsWith("+")) {
			out.push({ kind: "add", text: line.slice(1), newNo });
			newNo++;
		} else if (line.startsWith("-")) {
			out.push({ kind: "del", text: line.slice(1), oldNo });
			oldNo++;
		} else if (line.startsWith(" ")) {
			// A blank context line is a single space; a bare "" is only the split's trailing element.
			out.push({ kind: "ctx", text: line.slice(1), oldNo, newNo });
			oldNo++;
			newNo++;
		}
	}
	if (out.length > MAX_DIFF_LINES) {
		const kept = out.slice(0, MAX_DIFF_LINES);
		kept.push({ kind: "meta", text: `… ${out.length - MAX_DIFF_LINES} more lines not shown` });
		return kept;
	}
	return out;
}

// ---- layout ----

export interface Model {
	readonly files: readonly FileChange[];
	readonly focus: string | undefined;
	readonly diff: readonly DiffLine[];
	/** True when git could not answer (not a repository, git missing). */
	readonly unavailable?: boolean;
}

export interface Paint {
	(kind: "title" | "add" | "del" | "ctx" | "hunk" | "muted" | "accent" | "path" | "border" | "meta", text: string): string;
}

export interface LayoutInput {
	model: Model;
	scroll: number;
	width: number;
	height: number;
}

export interface Layout {
	lines: string[];
	/** The largest scroll the diff area accepts for this model and size. */
	maxScroll: number;
	/** Lines hidden below the diff area at this scroll. */
	below: number;
}

const plainPaint: Paint = (_k, t) => t;

/** The panel: header, file list, separator, hunks of the focused file. Each line fits `width` columns. */
export function layout(input: LayoutInput, paint: Paint = plainPaint, measure: (s: string) => number = cpLength): Layout {
	const width = Math.max(20, input.width);
	const height = Math.max(6, input.height);
	const { model } = input;
	const lines: string[] = [];
	const fit = (s: string) => (measure(s) > width ? clipEnd(s, width, measure) : s);

	if (model.unavailable) {
		lines.push(paint("title", "diff"));
		lines.push(paint("muted", fit("Not a git repository — nothing to compare.")));
		return { lines, maxScroll: 0, below: 0 };
	}
	const t = totals(model.files);
	if (t.files === 0) {
		lines.push(paint("title", "diff"));
		lines.push(paint("muted", fit("No changes against HEAD.")));
		return { lines, maxScroll: 0, below: 0 };
	}

	lines.push(fit(`${paint("title", `${t.files} ${t.files === 1 ? "file" : "files"} changed`)} ${paint("add", `+${t.added}`)} ${paint("del", `-${t.removed}`)}`));
	lines.push("");

	// File list: keeps the focused row visible, never more than 40% of the height.
	const listMax = Math.max(3, Math.min(model.files.length, Math.floor(height * 0.4)));
	const focusIdx = Math.max(0, model.files.findIndex((f) => f.path === model.focus));
	let start = 0;
	if (model.files.length > listMax) start = Math.min(Math.max(0, focusIdx - listMax + 1), model.files.length - listMax);
	const end = Math.min(model.files.length, start + listMax);
	if (start > 0) lines.push(paint("muted", `↑ ${start} more above`));
	for (let i = start; i < end; i++) {
		const f = model.files[i]!;
		const right = f.binary ? "bin" : f.untracked ? `+${f.added} new` : `+${f.added} -${f.removed}`;
		const rightPainted = f.binary ? paint("muted", right) : f.untracked ? `${paint("add", `+${f.added}`)} ${paint("muted", "new")}` : `${paint("add", `+${f.added}`)} ${paint("del", `-${f.removed}`)}`;
		const marker = i === focusIdx ? "▸ " : "  ";
		const room = width - measure(marker) - measure(right) - 1;
		const name = clipStart(renameParts(f.path).display, Math.max(8, room), measure);
		const gap = Math.max(1, width - measure(marker) - measure(name) - measure(right));
		const nameP = i === focusIdx ? paint("accent", name) : paint("path", name);
		lines.push(`${i === focusIdx ? paint("accent", marker) : marker}${nameP}${" ".repeat(gap)}${rightPainted}`);
	}
	if (end < model.files.length) lines.push(paint("muted", `↓ ${model.files.length - end} more below`));

	lines.push(paint("border", "─".repeat(width)));
	if (!model.focus) return { lines, maxScroll: 0, below: 0 };
	lines.push(fit(paint("path", renameParts(model.focus).display)));

	const area = Math.max(1, height - lines.length);
	const focused = model.files.find((f) => f.path === model.focus);
	const body: string[] = focused?.binary ? [paint("muted", "binary file")] : renderDiff(model.diff, width, paint, measure);
	const rename = renameParts(model.focus);
	if (rename.paths.length === 2) body.unshift(paint("muted", fit(`renamed from ${rename.paths[0]}`)));
	if (body.length === 0) body.push(paint("muted", "no hunks (mode change or identical content)"));
	const maxScroll = Math.max(0, body.length - area);
	const scroll = Math.min(Math.max(0, input.scroll), maxScroll);
	const slice = body.slice(scroll, scroll + area);
	const below = body.length - scroll - slice.length;
	if (below > 0 && slice.length > 0) slice[slice.length - 1] = paint("muted", fit(`↓ ${below} more (alt+n scrolls)`));
	lines.push(...slice);
	return { lines, maxScroll, below };
}

function renderDiff(diff: readonly DiffLine[], width: number, paint: Paint, measure: (s: string) => number): string[] {
	let maxNo = 0;
	for (const d of diff) maxNo = Math.max(maxNo, d.newNo ?? 0, d.oldNo ?? 0);
	const gutter = Math.max(3, String(maxNo).length);
	const out: string[] = [];
	for (const d of diff) {
		if (d.kind === "hunk") {
			out.push(paint("hunk", clipEnd(d.text, width, measure)));
			continue;
		}
		if (d.kind === "meta") {
			out.push(paint("meta", clipEnd(d.text, width, measure)));
			continue;
		}
		const no = d.kind === "del" ? d.oldNo : d.newNo;
		const sign = d.kind === "add" ? "+" : d.kind === "del" ? "-" : " ";
		const head = `${String(no ?? "").padStart(gutter)} ${sign}`;
		const text = clipEnd(expandTabs(d.text), Math.max(1, width - measure(head)), measure);
		const kind = d.kind === "add" ? "add" : d.kind === "del" ? "del" : "ctx";
		out.push(paint(kind, `${head}${text}`));
	}
	return out;
}

export function expandTabs(s: string): string {
	return s.replace(/\t/g, "    ");
}

/** The footer text: `diff +324 -259`, `diff clean`, or nothing when git is unavailable. */
export function statusText(model: Model | null): string | undefined {
	if (!model || model.unavailable) return undefined;
	const t = totals(model.files);
	return t.files === 0 ? "diff clean" : `diff +${t.added} -${t.removed}`;
}

export function clampScroll(scroll: number, maxScroll: number): number {
	return Math.min(Math.max(0, Math.floor(scroll)), Math.max(0, maxScroll));
}

// ---- width helpers (code points; the extension re-clips with pi-tui's width-aware truncation) ----

export function cpLength(s: string): number {
	return Array.from(s).length;
}

/** Keep the head, drop the tail: `abc…`. */
export function clipEnd(s: string, width: number, measure: (s: string) => number = cpLength): string {
	if (measure(s) <= width) return s;
	const cps = Array.from(s);
	let out = "";
	for (const c of cps) {
		if (measure(out + c + "…") > width) break;
		out += c;
	}
	return out + "…";
}

/** Keep the tail, drop the head: `…c/file.ts` — paths read from the right. */
export function clipStart(s: string, width: number, measure: (s: string) => number = cpLength): string {
	if (measure(s) <= width) return s;
	const cps = Array.from(s);
	let out = "";
	for (let i = cps.length - 1; i >= 0; i--) {
		if (measure("…" + cps[i] + out) > width) break;
		out = cps[i] + out;
	}
	return "…" + out;
}
