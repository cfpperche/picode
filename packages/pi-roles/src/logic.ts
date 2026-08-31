/** Pure routing logic for pi-roles. No pi imports — node:test covers this. */

export type ThinkingLevel =
	| "off"
	| "minimal"
	| "low"
	| "medium"
	| "high"
	| "xhigh"
	| "max";

export type Assignment = {
	model: string;
	thinking?: ThinkingLevel;
};

export type RolesConfig = {
	builtin: {
		default?: Assignment;
		vision?: Assignment;
		plan?: Assignment;
	};
	custom: Array<{ name: string } & Assignment>;
};

export type Mode =
	| { kind: "auto" }
	| { kind: "lock"; role: string };

export type Decision =
	| { kind: "noop" }
	| { kind: "switch"; target: Assignment; why: string }
	| { kind: "error"; message: string };

export type ParseResult =
	| { ok: true; config: RolesConfig }
	| { ok: false; error: string };

const THINKING = new Set<string>([
	"off",
	"minimal",
	"low",
	"medium",
	"high",
	"xhigh",
	"max",
]);

export const THINKING_LEVELS: ThinkingLevel[] = [
	"off",
	"minimal",
	"low",
	"medium",
	"high",
	"xhigh",
	"max",
];
export const BUILTIN_ROLES = ["default", "vision", "plan"] as const;

const NAME_RE = /^[a-zA-Z][a-zA-Z0-9_-]*$/;
const RESERVED = new Set(["auto", "default", "vision", "plan", "role", "roles"]);
const IMAGE_EXT = /\.(png|jpe?g|gif|webp|bmp)(\b|$)/i;
const BUILTIN = new Set<string>(BUILTIN_ROLES);

/** Group `provider/id` strings. Providers sorted; ids keep first-seen order. */
export function groupModels(models: string[]): Map<string, string[]> {
	const map = new Map<string, string[]>();
	for (const m of models) {
		const parsed = parseModelId(m);
		if (!parsed) continue;
		const list = map.get(parsed.provider) ?? [];
		if (!list.includes(parsed.id)) list.push(parsed.id);
		map.set(parsed.provider, list);
	}
	return map;
}

export function providersOf(models: string[]): string[] {
	return [...groupModels(models).keys()].sort();
}

export function idsForProvider(models: string[], provider: string): string[] {
	return groupModels(models).get(provider) ?? [];
}

export function parseModelId(model: string): { provider: string; id: string } | null {
	const i = model.indexOf("/");
	if (i <= 0 || i === model.length - 1) return null;
	return { provider: model.slice(0, i), id: model.slice(i + 1) };
}

export function wantsVision(text: string, images: unknown[] | undefined): boolean {
	if (images && images.length > 0) return true;
	return IMAGE_EXT.test(text ?? "");
}

export function resolveRole(config: RolesConfig, role: string): Assignment | undefined {
	if (role === "default") return config.builtin.default;
	if (role === "vision") return config.builtin.vision;
	if (role === "plan") return config.builtin.plan;
	return config.custom.find((c) => c.name === role);
}

export function parseConfig(raw: unknown): ParseResult {
	if (raw === null || typeof raw !== "object" || Array.isArray(raw)) {
		return { ok: false, error: "roles.json must be an object" };
	}
	const obj = raw as Record<string, unknown>;
	const builtin: RolesConfig["builtin"] = {};
	if (obj.builtin !== undefined) {
		if (obj.builtin === null || typeof obj.builtin !== "object" || Array.isArray(obj.builtin)) {
			return { ok: false, error: "builtin must be an object" };
		}
		const b = obj.builtin as Record<string, unknown>;
		for (const key of ["default", "vision", "plan"] as const) {
			if (b[key] === undefined) continue;
			const parsed = parseAssignment(b[key], `builtin.${key}`);
			if (!parsed.ok) return parsed;
			builtin[key] = parsed.assignment;
		}
	}

	const custom: RolesConfig["custom"] = [];
	const seen = new Set<string>();
	if (obj.custom !== undefined) {
		if (!Array.isArray(obj.custom)) {
			return { ok: false, error: "custom must be an array" };
		}
		for (let i = 0; i < obj.custom.length; i++) {
			const item = obj.custom[i];
			if (item === null || typeof item !== "object" || Array.isArray(item)) {
				return { ok: false, error: `custom[${i}] must be an object` };
			}
			const rec = item as Record<string, unknown>;
			if (typeof rec.name !== "string" || rec.name.length === 0) {
				return { ok: false, error: `custom[${i}].name is required` };
			}
			if (!NAME_RE.test(rec.name)) {
				return { ok: false, error: `custom[${i}].name "${rec.name}" is not a valid role name` };
			}
			if (RESERVED.has(rec.name) || BUILTIN.has(rec.name)) {
				return { ok: false, error: `custom[${i}].name "${rec.name}" is reserved` };
			}
			if (seen.has(rec.name)) {
				return { ok: false, error: `custom role "${rec.name}" is duplicated` };
			}
			seen.add(rec.name);
			const parsed = parseAssignment(item, `custom[${i}]`);
			if (!parsed.ok) return parsed;
			custom.push({ name: rec.name, ...parsed.assignment });
		}
	}

	return { ok: true, config: { builtin, custom } };
}

function parseAssignment(
	raw: unknown,
	path: string,
): ParseResult | { ok: true; assignment: Assignment } {
	if (raw === null || typeof raw !== "object" || Array.isArray(raw)) {
		return { ok: false, error: `${path} must be an object` };
	}
	const rec = raw as Record<string, unknown>;
	if (typeof rec.model !== "string" || rec.model.trim() === "") {
		return { ok: false, error: `${path}.model is required` };
	}
	if (!parseModelId(rec.model)) {
		return { ok: false, error: `${path}.model "${rec.model}" must be provider/id` };
	}
	let thinking: ThinkingLevel | undefined;
	if (rec.thinking !== undefined) {
		if (typeof rec.thinking !== "string" || !THINKING.has(rec.thinking)) {
			return { ok: false, error: `${path}.thinking "${String(rec.thinking)}" is not a valid thinking level` };
		}
		thinking = rec.thinking as ThinkingLevel;
	}
	const assignment: Assignment = { model: rec.model };
	if (thinking) assignment.thinking = thinking;
	return { ok: true, assignment };
}

export function decideOnInput(args: {
	config: RolesConfig | null;
	mode: Mode;
	text: string;
	images?: unknown[];
	source?: string;
}): Decision {
	if (args.source === "extension") return { kind: "noop" };
	if (!args.config) return { kind: "noop" };

	if (args.mode.kind === "lock") {
		const target = resolveRole(args.config, args.mode.role);
		if (!target) {
			return { kind: "error", message: `Role "${args.mode.role}" is not configured` };
		}
		return { kind: "switch", target, why: `lock /${args.mode.role}` };
	}

	if (wantsVision(args.text, args.images)) {
		const target = resolveRole(args.config, "vision");
		if (!target) return { kind: "noop" };
		return { kind: "switch", target, why: "image detected" };
	}

	const target = resolveRole(args.config, "default");
	if (!target) return { kind: "noop" };
	return { kind: "switch", target, why: "text" };
}

export function lockRole(config: RolesConfig | null, role: string): Decision {
	if (!config) {
		return { kind: "error", message: "No .pi/roles.json — create one or /auto does nothing" };
	}
	const target = resolveRole(config, role);
	if (!target) {
		return { kind: "error", message: `Role "${role}" is not configured` };
	}
	return { kind: "switch", target, why: `lock /${role}` };
}

export type MutateResult =
	| { ok: true; config: RolesConfig }
	| { ok: false; error: string };

export function emptyConfig(): RolesConfig {
	return { builtin: {}, custom: [] };
}

function validAssignment(assignment: Assignment): string | null {
	if (!parseModelId(assignment.model)) return `model "${assignment.model}" must be provider/id`;
	if (assignment.thinking !== undefined && !THINKING.has(assignment.thinking)) {
		return `thinking "${assignment.thinking}" is not a valid thinking level`;
	}
	return null;
}

export function editRole(config: RolesConfig, role: string, assignment: Assignment): MutateResult {
	const bad = validAssignment(assignment);
	if (bad) return { ok: false, error: bad };
	if (role === "default" || role === "vision" || role === "plan") {
		return { ok: true, config: { builtin: { ...config.builtin, [role]: assignment }, custom: config.custom } };
	}
	const idx = config.custom.findIndex((c) => c.name === role);
	if (idx < 0) return { ok: false, error: `Role "${role}" is not configured` };
	const custom = config.custom.slice();
	custom[idx] = { name: role, ...assignment };
	return { ok: true, config: { builtin: config.builtin, custom } };
}

export function addCustom(config: RolesConfig, name: string, assignment: Assignment): MutateResult {
	const bad = validAssignment(assignment);
	if (bad) return { ok: false, error: bad };
	if (!NAME_RE.test(name)) return { ok: false, error: `"${name}" is not a valid role name` };
	if (RESERVED.has(name) || BUILTIN.has(name)) {
		return { ok: false, error: `"${name}" is reserved` };
	}
	if (config.custom.some((c) => c.name === name)) {
		return { ok: false, error: `custom role "${name}" already exists` };
	}
	return {
		ok: true,
		config: { builtin: config.builtin, custom: [...config.custom, { name, ...assignment }] },
	};
}

export function removeCustom(config: RolesConfig, name: string): MutateResult {
	if (BUILTIN.has(name) || name === "auto") {
		return { ok: false, error: `Cannot remove builtin role "${name}"` };
	}
	if (!config.custom.some((c) => c.name === name)) {
		return { ok: false, error: `Role "${name}" is not configured` };
	}
	return {
		ok: true,
		config: { builtin: config.builtin, custom: config.custom.filter((c) => c.name !== name) },
	};
}

function assignmentJson(a: Assignment): Record<string, unknown> {
	const row: Record<string, unknown> = { model: a.model };
	if (a.thinking) row.thinking = a.thinking;
	return row;
}

/** Merge the parsed config back onto the original JSON so unknown keys survive. */
export function serializeConfig(
	config: RolesConfig,
	raw?: Record<string, unknown>,
): Record<string, unknown> {
	const out: Record<string, unknown> = { ...(raw ?? {}) };
	const prevBuiltin =
		raw && typeof raw.builtin === "object" && raw.builtin && !Array.isArray(raw.builtin)
			? { ...(raw.builtin as Record<string, unknown>) }
			: {};
	const builtin: Record<string, unknown> = { ...prevBuiltin };
	for (const key of BUILTIN_ROLES) {
		if (config.builtin[key]) builtin[key] = assignmentJson(config.builtin[key]!);
		else delete builtin[key];
	}
	out.builtin = builtin;
	out.custom = config.custom.map((c) => ({ name: c.name, ...assignmentJson(c) }));
	return out;
}
