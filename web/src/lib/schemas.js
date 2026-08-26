import { z } from "zod";

const required = (label) => z.string().trim().min(1, label + " is required.");

const modelPick = z.object({
  provider: required("Provider"),
  model: required("Model"),
  thinking: required("Thinking"),
});

export const createWorkspaceSchema = modelPick.extend({
  name: required("Name"),
  path: required("Folder path"),
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
