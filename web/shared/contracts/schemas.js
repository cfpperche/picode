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

export const webappUrlSchema = z.object({
  url: required("URL").max(2048, "Use a URL up to 2048 characters.").refine((raw) => {
    try {
      const u = new URL(raw.includes("://") ? raw : "https://" + raw);
      return ["http:", "https:"].includes(u.protocol) && !!u.hostname && !u.username && !u.password && !/[\s\\]/.test(raw);
    } catch { return false; }
  }, "Use an HTTP or HTTPS address without a username or password."),
});
export const webappNameSchema = z.object({
  name: required("Name").refine((name) => [...name].length <= 200, "Use up to 200 characters for the name."),
});
export const webappInstallSchema = webappUrlSchema.extend(webappNameSchema.shape);

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
  tools: z.array(z.string().regex(/^[a-z][a-z0-9-]{0,31}$/, "PiCode tools are named computer, browser, …")).max(8, "Too many PiCode tools.").default([]),
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

// Workspace catalog picker (ADR-0159 Fatia 3): bind a launchable CLI as a
// managed principal. Empty name lets the server use the catalog name.
export const managedPrincipalSchema = z.object({
  cli: required("CLI").max(64, "Unknown CLI."),
  name: z.string().trim().max(80, "Use up to 80 characters for the name."),
});

// Free New → Agent (ADR-0179): any launchable CLI, a name (the work folder
// is derived from it when no folder is given) and an optional folder.
export const freeAgentPickSchema = z.object({
  cli: required("CLI").max(64, "Unknown CLI."),
  name: required("Name").max(80, "Use up to 80 characters for the name."),
  path: z.string().trim(),
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

// Fork agent…: a name for the new agent and where it works. The task is
// free text (empty opens the copy waiting); the server bounds its length.
export const forkAgentSchema = z.object({
  name: required("Name").max(80, "Use up to 80 characters for the name."),
  where: z.enum(["worktree", "same"], { message: "Choose where the fork works." }),
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

export const createWsAgentSchema = modelPick.extend({
  name: required("Name"),
});

export const apiKeySchema = z.object({
  key: z.string().trim().min(1, "API key is required."),
});

// Codex on Amazon Bedrock (ADR-0191): the two ways Codex's own app-server
// signs in to Bedrock. The server checks the same rules.
export const codexBedrockSchema = z.object({
  type: z.enum(["amazonBedrock", "amazonBedrockAccessKeys"]),
  region: z.string().trim().regex(/^[a-z0-9-]{2,32}$/, "An AWS region is required, like us-east-1."),
  apiKey: z.string().trim().optional(),
  accessKeyId: z.string().trim().optional(),
  secretAccessKey: z.string().trim().optional(),
  sessionToken: z.string().trim().optional(),
}).superRefine((v, ctx) => {
  if (v.type === "amazonBedrock" && !v.apiKey) ctx.addIssue({ code: "custom", path: ["apiKey"], message: "A Bedrock API key is required." });
  if (v.type === "amazonBedrockAccessKeys" && !(v.accessKeyId && v.secretAccessKey)) ctx.addIssue({ code: "custom", path: ["accessKeyId"], message: "An access key ID and its secret are required." });
});

// Claude Code on a third-party platform (ADR-0189): the fields Claude Code's
// own /login wizard asks for, per platform and sign-in method. The server
// checks the same rules; these give the same message in every browser.
const REGION = /^[a-z0-9-]{2,32}$/;
export const claudePlatformSchema = z.object({
  kind: z.enum(["bedrock", "vertex", "foundry", "gateway"]),
  auth: z.string().min(1, "Pick how Claude Code signs in."),
  region: z.string().trim().optional(),
  profile: z.string().trim().optional(),
  bearerToken: z.string().trim().optional(),
  accessKeyId: z.string().trim().optional(),
  secretAccessKey: z.string().trim().optional(),
  sessionToken: z.string().trim().optional(),
  project: z.string().trim().optional(),
  keyFile: z.string().trim().optional(),
  resource: z.string().trim().optional(),
  apiKey: z.string().trim().optional(),
  baseUrl: z.string().trim().optional(),
  authToken: z.string().trim().optional(),
  model: z.string().trim().optional(),
  fastModel: z.string().trim().optional(),
}).superRefine((v, ctx) => {
  const need = (ok, path, message) => { if (!ok) ctx.addIssue({ code: "custom", path: [path], message }); };
  if (v.kind === "bedrock") {
    need(REGION.test(v.region || ""), "region", "An AWS region is required, like us-east-1.");
    if (v.auth === "bearer") need(!!v.bearerToken, "bearerToken", "A Bedrock API key is required.");
    if (v.auth === "profile") need(!!v.profile, "profile", "An AWS profile name is required.");
    if (v.auth === "accessKey") need(!!v.accessKeyId && !!v.secretAccessKey, "accessKeyId", "An access key ID and its secret are required.");
  }
  if (v.kind === "vertex") {
    need(!!v.project, "project", "A Google Cloud project ID is required.");
    need(REGION.test(v.region || ""), "region", "A region is required, like us-east5 or global.");
    if (v.auth === "serviceAccount") need((v.keyFile || "").startsWith("/"), "keyFile", "The service account key file needs its full path.");
  }
  if (v.kind === "gateway") {
    let ok = false;
    try {
      const u = new URL(v.baseUrl || "");
      const local = ["localhost", "127.0.0.1", "[::1]"].includes(u.hostname);
      ok = u.protocol === "https:" || (u.protocol === "http:" && local);
    } catch { ok = false; }
    need(ok, "baseUrl", "The gateway URL must start with https:// (http:// only for this machine).");
    need(!!v.authToken, "authToken", "The gateway's token is required.");
  }
  if (v.kind === "foundry") {
    need(/^[A-Za-z0-9-]{2,64}$/.test(v.resource || ""), "resource", "The Foundry resource name is required (letters, digits and dashes).");
    if (v.auth === "apiKey") need(!!v.apiKey, "apiKey", "A Foundry API key is required.");
  }
});

// Custom provider endpoints (ADR-0129): the closed list of pi API types the
// form offers, in display order. urlHint is what pi expects in the base URL
// (pi appends the route itself), taken from pi's own examples so the form
// never invents a rule: an OpenAI-style root ends with /v1, Google's with
// /v1beta, and Anthropic-compatible proxies differ — Anthropic's own root is
// https://api.anthropic.com/v1 while some proxies take no suffix.
export const CUSTOM_PROVIDER_APIS = [
  {
    value: "openai-completions",
    label: "OpenAI Chat Completions (most gateways)",
    placeholder: "Base URL — e.g. https://api.example.com/v1",
    urlHint: "The API root — the route is appended to it.",
  },
  {
    value: "openai-responses",
    label: "OpenAI Responses",
    placeholder: "Base URL — e.g. https://api.example.com/v1",
    urlHint: "The API root — the route is appended to it.",
  },
  {
    value: "anthropic-messages",
    label: "Anthropic Messages",
    placeholder: "Base URL — e.g. https://api.anthropic.com/v1",
    urlHint: "The route /messages is appended — the root usually ends in /v1.",
  },
  {
    value: "google-generative-ai",
    label: "Google Generative AI",
    placeholder: "Base URL — e.g. https://generativelanguage.googleapis.com/v1beta",
    urlHint: "The model root — Google's ends in /v1beta.",
  },
];

// customApiHint is the hint for one API type, with a fallback so a type added
// later cannot leave the field unexplained.
export function customApiHint(api) {
  const found = CUSTOM_PROVIDER_APIS.find((a) => a.value === api);
  return found || CUSTOM_PROVIDER_APIS[0];
}

// THINKING_LEVELS are the pi thinking levels the provider form manages.
// "off" is deliberately absent: pi's default map already covers it, and its
// provider value is not always the level name (several pi catalogs send
// "none"), so the form never invents one. The schema validates against this
// list, so it lives here next to it.
export const THINKING_LEVELS = ["minimal", "low", "medium", "high", "xhigh", "max"];

// DEFAULT_THINKING_LEVELS mirrors what pi offers when a model carries no
// thinkingLevelMap: the standard levels through high. xhigh and max are
// extended levels and stay hidden until a map claims them.
export const DEFAULT_THINKING_LEVELS = ["minimal", "low", "medium", "high"];

// CUSTOM_INPUT_MODALITIES are pi's per-model input vocabulary
// (pi-ai types.d.ts Model.input): exactly text and image.
export const CUSTOM_INPUT_MODALITIES = ["text", "image"];

// CUSTOM_THINKING_FORMATS are the compat.thinkingFormat values the form
// offers, in display order. The values are pi's own union
// (pi-ai dist/types.d.ts); "openai" is the default and the branch that sends
// reasoning_effort, which is why the form writes "openai" rather than the
// undocumented "reasoning_effort" a hand-edited file may carry (that value
// falls through to the same branch, and the read path maps it back to
// "openai"). needs names the compat object a format requires, so the editor
// for it appears only when it is in play.
export const CUSTOM_THINKING_FORMATS = [
  { value: "", label: "Default — pi's choice for this API type", needs: "" },
  { value: "openai", label: "openai — reasoning_effort (the OpenAI standard)", needs: "" },
  { value: "deepseek", label: "deepseek — thinking: {type} plus reasoning_effort", needs: "" },
  { value: "qwen", label: "qwen — enable_thinking (DashScope)", needs: "" },
  { value: "qwen-chat-template", label: "qwen-chat-template — enable_thinking + preserve_thinking", needs: "" },
  { value: "openrouter", label: "openrouter — reasoning: {effort}", needs: "" },
  { value: "together", label: "together — reasoning: {enabled}", needs: "" },
  { value: "zai", label: "zai — thinking: {type}", needs: "" },
  { value: "ant-ling", label: "ant-ling — reasoning: {effort}", needs: "" },
  { value: "string-thinking", label: "string-thinking — a top-level thinking string", needs: "" },
  { value: "chat-template", label: "chat-template — chat_template_kwargs", needs: "kwargs" },
  { value: "baseten", label: "baseten — chat_template_args", needs: "args" },
];

// THINKING_FORMAT_NEEDS maps a format to the object it reads, so the form can
// skip rendering an editor nothing would use.
export const THINKING_FORMAT_NEEDS = Object.fromEntries(
  CUSTOM_THINKING_FORMATS.map((f) => [f.value, f.needs]),
);

// OMP_THINKING_FORMATS is the subset omp's own compat table documents
// (models.yml, read 2026-09-21 from omp 18.2.8). pi's other formats are pi's
// wire code; omp's form does not offer what omp would send wrong.
export const OMP_THINKING_FORMATS = new Set(["openai", "openrouter", "zai", "qwen", "qwen-chat-template"]);

// CHAT_TEMPLATE_VARS are the pi-controlled values a kwargs/args entry may
// reference instead of a literal; anything else is the provider's own key.
export const CHAT_TEMPLATE_VARS = ["thinking.enabled", "thinking.effort", "thinking.budget"];

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
    // One row per listed id: pi takes contextWindow/maxTokens per model, and
    // the form used to write one number for the whole list. Name, input
    // modalities and cost ride the same row; the server deletes a stored key
    // the row leaves blank, so the form owns what it shows.
    modelLimits: z.record(z.string(), z.object({
      contextWindow: z.string(),
      maxTokens: z.string(),
      name: z.string(),
      input: z.array(z.enum(CUSTOM_INPUT_MODALITIES)),
      cost: z.object({ input: z.string(), output: z.string(), cacheRead: z.string(), cacheWrite: z.string() }),
    })),
    compatDeveloper: z.boolean(),
    compatReasoning: z.boolean(),
    compatStreaming: z.boolean(),
    thinkingFormat: z.enum(CUSTOM_THINKING_FORMATS.map((f) => f.value)),
    chatTemplateKwargs: z.string(),
    chatTemplateArgs: z.string(),
    reasoningModel: z.boolean(),
    thinkingLevels: z.array(z.enum(THINKING_LEVELS)),
    // thinkingLevelValues overrides the provider value per selected level: a
    // blank value keeps the level's own name (xhigh), a filled one writes it
    // (xhigh -> "high"). Values for unselected levels never reach the wire.
    thinkingLevelValues: z.record(z.string(), z.string()),
    key: z.string(),
  }).superRefine((v, ctx) => {
    const fail = (message) => ctx.addIssue({ code: "custom", message });
    if (takenIds.includes(v.id)) fail(v.id + " already exists. Pick another name.");
    const ids = customModelIds(v.modelsText);
    if (!ids.length) fail("Add at least one model id.");
    if (ids.some((id) => /\s/.test(id))) fail("Model ids cannot contain spaces.");
    if (new Set(ids).size !== ids.length) fail("Each model id can be listed only once.");
    // A row is refused where it sits, and the message names the model: with a
    // list of ids, "Max output must be a positive whole number" alone would
    // leave the person hunting for the row.
    for (const [id, row] of Object.entries(v.modelLimits || {})) {
      for (const [field, label] of [["contextWindow", "Context window"], ["maxTokens", "Max output"]]) {
        const raw = String((row && row[field]) || "").trim();
        if (raw && (!/^\d+$/.test(raw) || Number(raw) <= 0)) {
          fail(id + ": " + label.toLowerCase() + " must be a positive whole number.");
        }
      }
      if (String((row && row.name) || "").trim().length > 120) {
        fail(id + ": name must be 120 characters or less.");
      }
      // Cost is all or nothing: pi's ModelCost needs all four rates, and a
      // half-filled row would silently zero real money. Zeros are fine.
      const cost = (row && row.cost) || {};
      const rates = ["input", "output", "cacheRead", "cacheWrite"].map((k) => String(cost[k] || "").trim());
      if (rates.some(Boolean) && rates.some((r) => !r)) {
        fail(id + ": fill all four cost rates or leave them blank.");
      }
      for (const r of rates) {
        if (r && (!/^\d+(\.\d+)?$/.test(r) || !Number.isFinite(Number(r)))) {
          fail(id + ": cost rates must be zero or above.");
        }
      }
    }
    for (const [level, value] of Object.entries(v.thinkingLevelValues || {})) {
      const trimmed = String(value || "").trim();
      if (!trimmed) continue;
      if (!THINKING_LEVELS.includes(level)) fail("Unknown thinking level: " + level + ".");
      else if (trimmed.length > 64 || /\s/.test(trimmed)) fail("Provider value for " + level + " must be one word.");
    }
    if (v.reasoningModel && !v.thinkingLevels.length) fail("Select at least one thinking level.");
    // The kwargs/args editors hold JSON; the same parse builds the payload, so
    // a typo is caught here with a message instead of reaching the file.
    for (const [field, label] of [["chatTemplateKwargs", "Chat template kwargs"], ["chatTemplateArgs", "Chat template args"]]) {
      const raw = String(v[field] || "").trim();
      if (!raw) continue;
      const bad = chatTemplateObjectError(raw);
      if (bad) fail(label + ": " + bad);
    }
    if (requireKey && !v.key.trim()) fail("API key is required.");
  });
}

// chatTemplateObjectError parses a chat template object and returns the
// message for the first problem, or "" when it is usable. Values may be a
// string, number, boolean or null, or a pi-controlled reference such as
// {"$var": "thinking.enabled"} with an optional omitWhenOff.
export function chatTemplateObjectError(raw) {
  let obj;
  try {
    obj = JSON.parse(raw);
  } catch {
    return "that is not valid JSON.";
  }
  if (obj === null || typeof obj !== "object" || Array.isArray(obj)) return "use a JSON object, e.g. {\"enable_thinking\": true}.";
  for (const [key, value] of Object.entries(obj)) {
    if (value === null || ["string", "number", "boolean"].includes(typeof value)) continue;
    if (typeof value === "object" && !Array.isArray(value)) {
      const keys = Object.keys(value);
      if (!keys.includes("$var")) return key + ": an object value needs a \"$var\" (pi-controlled thinking value).";
      if (keys.some((k) => k !== "$var" && k !== "omitWhenOff")) return key + ": only \"$var\" and \"omitWhenOff\" may sit beside each other.";
      if (!CHAT_TEMPLATE_VARS.includes(value.$var)) return key + ": \"$var\" must be one of " + CHAT_TEMPLATE_VARS.join(", ") + ".";
      if ("omitWhenOff" in value && typeof value.omitWhenOff !== "boolean") return key + ": \"omitWhenOff\" is true or false.";
      continue;
    }
    return key + ": use a string, number, true/false, null or a {\"$var\": …} reference.";
  }
  return "";
}

// chatTemplateObject parses what the editor holds, or null when it is empty.
// A payload builder calls it only after the schema has accepted the text.
export function chatTemplateObject(raw) {
  const text = String(raw || "").trim();
  if (!text) return null;
  return JSON.parse(text);
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

// Every permission kind the daemon stores and the shell maps (the store's
// closed list, `internal/store/browser_permissions.go`). The dialog shows a
// titled subset and lets the rest be authored by name: a control nobody can
// reach is worse than a longer list.
export const BROWSER_PERMISSION_KINDS = [
  "camera",
  "microphone",
  "location",
  "notifications",
  "clipboard",
  "autoplay",
  "sensors",
  "midi",
  "fonts",
  "filesystem",
];

// One site exception, authored by hand in Settings ▸ Browser (the reference's
// "+ Add"): a site or pattern, a kind, and the decision to remember. The
// normalization mirrors browserGrantSchema, and `*` is every site — those are
// the two spellings the store and the shell already match.
export const browserSiteSchema = z.object({
  site: z.string().transform((raw) => {
    let host = String(raw || "").trim().toLowerCase().replace(/^https?:\/\//, "").split("/")[0];
    return host.startsWith("[") ? host.slice(1).split("]")[0] : host.split(":")[0];
  }).refine(
    (host) => host === "*" || /^[a-z0-9*.\-]+$/.test(host),
    "Use a site like example.com, a pattern like *.example.com, or * for every site.",
  ),
  kind: z.enum(BROWSER_PERMISSION_KINDS),
  decision: z.enum(["allow", "deny", "ask"]),
});

// A browser grant (ADR-0128) as the editor edits it: the tier plus the
// domain table as one comma-separated line. The transform forgives what
// people paste — scheme, path, port are stripped — and what survives must
// be a bare host the origin rule can match.
export const browserGrantSchema = z.object({
  tier: z.enum(["read", "act", "full"]),
  domains: z.string().transform((raw) => raw.split(",").map((entry) => {
    let host = entry.trim().toLowerCase().replace(/^https?:\/\//, "").split("/")[0];
    host = host.startsWith("[") ? host.slice(1).split("]")[0] : host.split(":")[0];
    return host;
  }).filter(Boolean)).refine(
    (hosts) => hosts.every((h) => /^[a-z0-9*.\-:]+$/.test(h)),
    "Use bare hosts like example.com or *.example.com — no scheme, no path.",
  ),
});

// Workspace settings (the card menu's Settings…): the name on the card, and
// the project's integration declaration (ADR-0182) — mirrors the store's
// limits: at most eight checks, each one line of at most 300 bytes. Blank
// rows are the form's, not the declaration's: they are dropped before this.
export const workspaceSettingsSchema = z.object({
  name: required("Name").max(120, "Use up to 120 characters for the name."),
  // The engine the project declares (ADR-0186): its own merge queue, PiCode's
  // local runner, or nothing declared — which runs nothing.
  mode: z.union([z.literal(""), z.enum(["provider", "local"])]),
  ffOnly: z.boolean(),
  checks: z.array(
    z.string().trim().min(1, "A check cannot be empty.")
      .refine((c) => new TextEncoder().encode(c).length <= 300, "Keep each check under 300 characters.")
      .refine((c) => !/[\r\n]/.test(c), "Each check is one line."),
  ).max(8, "Use up to eight checks."),
});

// A skill source (ADR-0196): owner/repo[/path][#ref], a GitHub URL, an
// https:// site with a .well-known skills index, or an absolute folder.
export const skillSourceSchema = z.object({
  source: required("Source").max(500, "Use a source up to 500 characters.").refine((raw) => {
    const s = raw.trim();
    if (s.startsWith("/") || s.startsWith("~/")) return true;
    if (/^[A-Za-z0-9][A-Za-z0-9-]{0,38}\/[A-Za-z0-9._-]{1,100}(\/[A-Za-z0-9._@-]+)*(#[A-Za-z0-9._/-]+)?$/.test(s)) return true;
    try {
      const u = new URL(s);
      return (u.protocol === "https:" || (u.protocol === "http:" && u.hostname === "github.com")) && !u.username && !u.password;
    } catch { return false; }
  }, "Use owner/repo, a GitHub or https:// address, or a folder path starting with / or ~/."),
});
