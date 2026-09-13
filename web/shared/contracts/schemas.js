import { z } from "zod";

export const llamaServiceSchema = z.object({
  port: z.number().int("Use a whole-number port.").min(1024, "Use a port from 1024 to 65535.").max(65535, "Use a port from 1024 to 65535."),
  context: z.number().int("Use a whole-number context size.").min(512, "Use a context size from 512 to 131072.").max(131072, "Use a context size from 512 to 131072."),
  threads: z.number().int("Use a whole-number thread count.").min(1, "Use 1 to 4 CPU threads.").max(4, "Use 1 to 4 CPU threads."),
  jinja: z.boolean(),
});
import { looksLikeRepoUrl } from "../domain/cloneUrl.js";
import { rowsError } from "../domain/automationSchedule.js";
import { CANVAS_LIMITS } from "../domain/canvas.js";
import { parseSnip, utf8Bytes, SNIP_LIMITS } from "../domain/snipDraft.js";

const required = (label) => z.string().trim().min(1, label + " is required.");

export const webhookSchema = z.object({
  url: z.string().trim().max(2048).refine((raw) => {
    try { const u = new URL(raw); return ["http:", "https:"].includes(u.protocol) && !!u.hostname && !u.username && !u.password && !u.hash; } catch { return false; }
  }, "Use an HTTP or HTTPS URL without a username, password or fragment."),
  types: z.string().transform((raw) => [...new Set(raw.split(",").map(t => t.trim().toLowerCase()).filter(Boolean))].sort())
    .refine(types => types.length > 0 && types.length <= 32 && types.every(t => /^[a-z][a-z0-9_.-]{0,79}$/.test(t) && !t.startsWith("webhook")), "Choose event prefixes such as agent. or inbox.; webhook events are internal."),
});

// Server-driven App forms still validate with the same browser-independent
// schema layer. Select/confirm values must come from the offered choices;
// free-form replies retain their existing optional, literal text semantics.
export function appFormSchema(fields) {
  const shape = {};
  for (const field of fields) {
    const choices = field.method === "confirm" ? ["yes", "no"] : field.options;
    shape[field.name] = field.method === "select" || field.method === "confirm"
      ? z.string().refine((value) => choices.includes(value), "Choose an available option for " + (field.title || field.name) + ".")
      : z.string();
  }
  return z.object(shape);
}

export const cliLaunchSchema = z.object({
  executable: z.string().max(8192),
  argsText: z.string().max(32768),
  pathText: z.string().max(8192),
  envText: z.string().max(32768),
  integration: z.boolean(),
}).superRefine((v, ctx) => {
  const fail = (message) => ctx.addIssue({ code: "custom", message });
  if (Object.values(v).some((x) => typeof x === "string" && /[\0\r]/.test(x))) fail("Launch settings contain an invalid character.");
  if (v.executable.includes("\n")) fail("Executable must be one path or command name.");
  for (const p of v.pathText.split("\n").filter((x) => x.trim())) {
    if (!p.trim().startsWith("/") || p.includes(":")) { fail("PATH entries must be absolute directories without colons."); break; }
  }
  const keys = new Set();
  for (const line of v.envText.split("\n").filter((x) => x.trim())) {
    const i = line.indexOf("="); const key = line.slice(0, i).trim();
    if (i < 1 || !/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) { fail("Use NAME=value for each environment variable."); break; }
    if (key.startsWith("PICODE_") || ["PATH", "HOME", "SHELL", "GROK_HOME", "HERMES_HOME"].includes(key)) { fail(`${key} is managed by the launcher.`); break; }
    if (keys.has(key)) { fail(`${key} appears more than once.`); break; }
    keys.add(key);
  }
});

export const cliTerminalSchema = z.object({
  name: required("Name").max(80, "Use up to 80 characters for the name."),
  workspaceId: z.string(),
  cwd: z.string(),
});

export const cliProfileSchema = z.object({ name: required("Profile name").max(80, "Use up to 80 characters for the name.") });

// Cross-CLI session handoff (ADR-0088): the choices the dialog sends.
export const sessionHandoffSchema = z.object({
  to: required("Target CLI"),
  mode: z.enum(["native", "brief"], { message: "Choose how the session travels." }),
  landing: z.enum(["agent", "terminal"], { message: "Choose where the session opens." }).optional(),
  window: z.enum(["recent", "all"]),
  tools: z.enum(["native", "text"]),
});

const modelPick = z.object({
  provider: required("Provider"),
  model: required("Model"),
  thinking: required("Thinking"),
});

// A workspace is just a folder (ADR-0027): no agent, so no model pick.
export const createWorkspaceSchema = z.object({
  name: required("Name"),
  path: required("Folder path"),
});

// Remote mode of the same form (ADR-0034): the server re-validates the URL.
export const createWorkspaceCloneSchema = z.object({
  url: required("Repository URL").refine(looksLikeRepoUrl, "That doesn't look like a git URL."),
  name: required("Name"),
  path: required("Destination"),
});

export const createFreeAgentSchema = modelPick.extend({
  name: required("Name"),
  path: z.string().trim(),
});

export const createWsAgentSchema = modelPick.extend({
  name: required("Name"),
});

export const apiKeySchema = z.object({
  key: z.string().trim().min(1, "API key is required."),
});

// Custom provider endpoints (ADR-0129): the closed list of pi API types the
// form offers, in display order.
export const CUSTOM_PROVIDER_APIS = [
  { value: "openai-completions", label: "OpenAI Chat Completions (most gateways)" },
  { value: "openai-responses", label: "OpenAI Responses" },
  { value: "anthropic-messages", label: "Anthropic Messages" },
  { value: "google-generative-ai", label: "Google Generative AI" },
];

// customProviderSchema validates the Add/Edit form. takenIds names ids the
// user may not claim (built-ins plus other custom definitions); requireKey
// is false while editing, where a blank key keeps the stored credential.
// customModelIds parses the one-id-per-line models textarea. It lives here
// because the schema's superRefine uses it.
export function customModelIds(modelsText) {
  return String(modelsText || "").split("\n").map((s) => s.trim()).filter(Boolean);
}

export function customProviderSchema({ takenIds = [], requireKey = true } = {}) {
  return z.object({
    id: z.string().trim().toLowerCase().min(1, "Name is required.")
      .regex(/^[a-z0-9][a-z0-9-]{0,63}$/, "Use lowercase letters, digits and dashes."),
    baseUrl: z.string().trim().min(1, "Base URL is required.").max(2048, "Base URL is too long.")
      .refine((u) => { try { const x = new URL(u); return (x.protocol === "http:" || x.protocol === "https:") && !!x.hostname; } catch { return false; } },
        "URL must start with http:// or https://."),
    api: z.enum(CUSTOM_PROVIDER_APIS.map((a) => a.value)),
    modelsText: z.string(),
    contextWindow: z.string(),
    maxTokens: z.string(),
    compatDeveloper: z.boolean(),
    compatReasoning: z.boolean(),
    key: z.string(),
  }).superRefine((v, ctx) => {
    const fail = (message) => ctx.addIssue({ code: "custom", message });
    if (takenIds.includes(v.id)) fail(v.id + " already exists. Pick another name.");
    const ids = customModelIds(v.modelsText);
    if (!ids.length) fail("Add at least one model id.");
    if (ids.some((id) => /\s/.test(id))) fail("Model ids cannot contain spaces.");
    if (new Set(ids).size !== ids.length) fail("Each model id can be listed only once.");
    for (const [field, label] of [["contextWindow", "Context window"], ["maxTokens", "Max output"]]) {
      const raw = v[field].trim();
      if (raw && (!/^\d+$/.test(raw) || Number(raw) <= 0)) fail(label + " must be a positive whole number.");
    }
    if (requireKey && !v.key.trim()) fail("API key is required.");
  });
}

export const llamaLoginSchema = z.object({
  url: z.string().trim().min(1, "Router URL is required.").refine(
    (u) => { try { const x = new URL(u); return x.protocol === "http:" || x.protocol === "https:"; } catch { return false; } },
    "URL must be http or https.",
  ),
  key: z.string().optional(),
});

export function parseForm(schema, data) {
  const got = schema.safeParse(data);
  if (got.success) return { ok: true, value: got.data, error: "" };
  return { ok: false, value: null, error: got.error.issues[0].message };
}

const pairRow = z.object({
  key: z.string(),
  value: z.string(),
});

export const mcpAddSchema = z.object({
  name: required("Name").regex(/^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/, "Name must be letters, digits, dot, dash, or underscore."),
  kind: z.enum(["stdio", "url"]),
  command: z.string(),
  args: z.string(),
  url: z.string(),
  auth: z.enum(["", "oauth", "bearer"]),
  token: z.string(),
  pairs: z.array(pairRow),
}).superRefine((v, ctx) => {
  if (v.kind === "stdio") {
    if (!v.command.trim()) ctx.addIssue({ code: "custom", message: "Command is required." });
  } else {
    const u = v.url.trim();
    if (!u) {
      ctx.addIssue({ code: "custom", message: "URL is required." });
    } else {
      try {
        const x = new URL(u);
        if (x.protocol !== "http:" && x.protocol !== "https:") {
          ctx.addIssue({ code: "custom", message: "URL must be http or https." });
        }
      } catch {
        ctx.addIssue({ code: "custom", message: "URL must be http or https." });
      }
    }
    if (v.auth === "bearer" && !v.token.trim()) {
      ctx.addIssue({ code: "custom", message: "Token is required." });
    }
  }
  for (const row of v.pairs) {
    const k = row.key.trim();
    if (!k && !String(row.value || "").trim()) continue;
    if (!k) {
      ctx.addIssue({ code: "custom", message: "Name is required." });
      break;
    }
    if (v.kind === "stdio" && !/^[A-Za-z_][A-Za-z0-9_]*$/.test(k)) {
      ctx.addIssue({ code: "custom", message: "Variable name must be letters, digits, or underscore." });
      break;
    }
    if (v.kind === "url" && !/^[A-Za-z0-9!#$%&'*+.^_`|~-]+$/.test(k)) {
      ctx.addIssue({ code: "custom", message: "Header name is invalid." });
      break;
    }
    if (/[\n\r]/.test(row.value || "")) {
      ctx.addIssue({ code: "custom", message: "Values must be a single line." });
      break;
    }
  }
});

export function pairsToMap(pairs) {
  const out = {};
  for (const row of pairs || []) {
    const k = String(row.key || "").trim();
    if (k) out[k] = String(row.value ?? "");
  }
  return out;
}

// Automations editor (ADR-0045). Numbers arrive as strings from inputs;
// the server re-validates everything.
const numField = z.string().trim();

export const snipSchema = z.object({
  title: required("Title").max(200, "Use up to 200 characters for the title."),
  slug: z.string().trim().max(64, "Use up to 64 characters for the slug.")
    .refine((s) => !s || /^[a-z0-9][a-z0-9_-]{0,63}$/.test(s), "Slug must be lowercase letters, digits, dash or underscore."),
  description: z.string().max(500, "Use up to 500 characters for the description."),
  body: z.string(),
  tags: z.string(),
}).superRefine((v, ctx) => {
  if (utf8Bytes(v.body) > SNIP_LIMITS.bodyBytes) {
    ctx.addIssue({ code: "custom", message: "Body is too long (max 100 KB)." });
  }
  const parsed = parseSnip(v.body);
  if (!parsed.ok) ctx.addIssue({ code: "custom", message: parsed.error === "unclosed placeholder" ? "Close every {{placeholder}}." : "Fix the placeholders in the body." });
});

export const automationSchema = z.object({
  name: required("Name").max(60, "Name is longer than 60 characters."),
  action: z.enum(["start", "message"]),
  targetAgentId: z.string().trim(),
  prompt: required("Prompt"),
  scheduleOn: z.boolean(),
  schedules: z.array(z.object({ label: z.string(), cron: z.string(), enabled: z.boolean() }).passthrough()),
  webhook: z.boolean(),
  notifyUrl: z.string().trim(),
  maxCostUsd: numField,
  maxRuns: numField,
  maxRunsWindowMin: numField,
}).superRefine((v, ctx) => {
  const issue = (message) => ctx.addIssue({ code: "custom", message });
  if (v.action === "message" && !v.targetAgentId) issue("Pick the agent to message.");
  if (!v.scheduleOn && !v.webhook) issue("Turn on a schedule or a webhook.");
  if (v.notifyUrl && !/^https?:\/\/[^\s]+$/.test(v.notifyUrl)) issue("Notify URL must be an http(s) address.");
  if (v.scheduleOn) {
    const err = rowsError(v.schedules);
    if (err) issue(err);
  }
  if (v.maxCostUsd !== "" && !(Number(v.maxCostUsd) > 0)) issue("Max cost must be a number above zero.");
  const runs = v.maxRuns === "" ? 0 : Number(v.maxRuns);
  if (v.maxRuns !== "" && !(Number.isInteger(runs) && runs >= 1)) issue("Max runs must be a whole number of at least 1.");
  if (runs > 0 && !(Number(v.maxRunsWindowMin) >= 1)) issue("Pick the window for max runs.");
});

// The Inspector's commit message is typed into a terminal as one line of
// keystrokes (ADR-0078): no line breaks, nothing that reads as a flag, and
// short enough to read in the prompt before pressing Enter.
export const commitMessageSchema = z.object({
  message: z.string().trim()
    .min(1, "A commit message is required.")
    .max(200, "Use up to 200 characters; a longer message belongs in the terminal.")
    .refine((v) => !/[\0-\x1f\x7f]/.test(v), "One line only: no line breaks or control characters.")
    .refine((v) => !v.startsWith("-"), "A message cannot start with a dash."),
});

// pi-roles configuration (ADR-0028/0033): one workspace file plus a
// per-agent overlay. `model` is provider/id; an omitted thinking level
// leaves the current level alone on that switch. Mirrors the server's
// validation (internal/pipkg/rolesconfig.go) and the extension's
// packages/pi-roles/src/logic.ts — the files are the only source of truth.
export const ROLES_THINKING_LEVELS = ["off", "minimal", "low", "medium", "high", "xhigh", "max"];
export const ROLES_BUILTIN = ["default", "vision", "plan"];
export const ROLES_RESERVED = ["auto", "default", "vision", "plan", "role", "roles"];

export const rolesModelIdSchema = z.string().trim()
  .min(1, "Pick a model.")
  .refine((v) => /^[^/]+\/.+/.test(v), "Model must be provider/id, e.g. zai/glm-5.3.");

export const rolesAssignmentSchema = z.object({
  model: rolesModelIdSchema,
  thinking: z.enum(ROLES_THINKING_LEVELS).optional(),
});

export const rolesCustomSchema = z.object({
  name: z.string().trim()
    .min(1, "A preset name is required.")
    .max(64, "Use up to 64 characters.")
    .refine((v) => /^[a-zA-Z][a-zA-Z0-9_-]*$/.test(v), "Use letters, digits, - or _ (must start with a letter).")
    .refine((v) => !ROLES_RESERVED.includes(v), "auto, default, vision, plan, role and roles are reserved names."),
  model: rolesModelIdSchema,
  thinking: z.enum(ROLES_THINKING_LEVELS).optional(),
});

export const rolesConfigSchema = z.object({
  builtin: z.object({
    default: rolesAssignmentSchema.nullish(),
    vision: rolesAssignmentSchema.nullish(),
    plan: rolesAssignmentSchema.nullish(),
  }),
  custom: z.array(rolesCustomSchema),
}).superRefine((v, ctx) => {
  const names = v.custom.map((c) => c.name);
  for (const name of names) {
    if (names.filter((n) => n === name).length > 1) {
      ctx.addIssue({ code: "custom", message: `Preset "${name}" is duplicated.`, path: ["custom"] });
      break;
    }
  }
});

// Canvas names (ADR-0108, renamed by ADR-0118): the store's own words,
// counted in characters the way it counts runes, so the dialog and a 400
// read the same.
export const canvasNameSchema = z.object({
  name: z.string().trim().min(1, "name is required")
    .refine((s) => Array.from(s).length <= CANVAS_LIMITS.name, `name is too long (max ${CANVAS_LIMITS.name} characters)`),
});

export const peerParticipantsSchema = z.object({
  participants: z.array(z.object({
    kind: z.enum(["agent", "terminal"]), ownerId: z.string().min(1),
    enabled: z.boolean(), revision: z.number().int().nonnegative(),
  })).min(1, "Select a participant first.").max(100, "Update up to 100 participants at a time."),
});

// Descriptor-driven package config (docs/plans/package-config-manifest.md):
// the server's descriptor carries the rules; this builds the same grammar
// client-side, so the form and the PUT agree before anything is sent. The
// payload omits unset fields — the file only gains keys the user set.
export function descriptorValuesSchema(fields) {
  const shape = {};
  for (const f of fields || []) {
    const required = !!f.required;
    let s;
    if (f.type === "enum") {
      const options = (f.options || []).filter((o) => o !== "");
      s = z.enum(options.length ? options : ["—"]);
      s = required ? s.refine((v) => v !== "—", { message: `${f.label} is required.` }) : s.or(z.literal(""));
    } else if (f.type === "boolean") {
      s = required ? z.boolean({ required_error: `${f.label} is required.` }) : z.boolean().optional();
    } else if (f.type === "number") {
      s = z.coerce.number({ invalid_type_error: `${f.label} must be a number.`, required_error: `${f.label} is required.` });
      if (f.min != null) s = s.min(f.min, `${f.label} must be at least ${f.min}.`);
      if (f.max != null) s = s.max(f.max, `${f.label} must be at most ${f.max}.`);
      if (!required) s = s.optional();
    } else {
      s = z.string({ required_error: `${f.label} is required.` });
      if (required) s = s.min(1, `${f.label} is required.`);
      else s = s.optional();
    }
    shape[f.key] = s;
  }
  return z.object(shape);
}

// The describe form (ADR-0119 C5): the owner describes a package's config
// file and fields once; options travel comma-separated and bounds as
// strings, converted to the descriptor's shape on save.
export const packageDescribeSchema = z.object({
  id: z.string().min(1, "Id is required."),
  title: z.string().min(1, "Title is required."),
  application: z.string().optional(),
  match: z.string().optional(),
  files: z.array(z.object({
    scope: z.enum(["agent", "workspace"]),
    path: z.string().min(1, "Path is required.")
      .refine((p) => !p.startsWith("/") && !p.includes(".."), "Relative path, no .."),
    format: z.literal("json"),
  })).length(1, "Exactly one file."),
  fields: z.array(z.object({
    key: z.string().regex(/^[a-zA-Z_][a-zA-Z0-9_.-]*$/, "Letters, digits, _ - or ."),
    label: z.string().min(1, "Label is required."),
    type: z.enum(["string", "enum", "boolean", "number", "secret"]),
    required: z.boolean().default(false),
    options: z.string().optional(),
    min: z.union([z.coerce.number(), z.literal("")]).optional(),
    max: z.union([z.coerce.number(), z.literal("")]).optional(),
    help: z.string().optional(),
  })).min(1, "Describe at least one field."),
}).superRefine((v, ctx) => {
  const keys = v.fields.map((f) => f.key);
  for (const k of keys) {
    if (keys.filter((x) => x === k).length > 1) {
      ctx.addIssue({ code: "custom", message: `Field "${k}" is duplicated.`, path: ["fields"] });
      break;
    }
  }
  for (const [i, f] of v.fields.entries()) {
    if (f.type === "enum" && !(f.options || "").trim()) {
      ctx.addIssue({ code: "custom", message: `Field "${f.key}" needs options.`, path: ["fields", i, "options"] });
    }
    if (f.type === "number" && f.min !== "" && f.max !== "" && f.min != null && f.max != null && Number(f.min) > Number(f.max)) {
      ctx.addIssue({ code: "custom", message: `Field "${f.key}": min is above max.`, path: ["fields", i, "min"] });
    }
  }
});
