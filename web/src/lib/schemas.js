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

export function parseForm(schema, data) {
  const got = schema.safeParse(data);
  if (got.success) return { ok: true, value: got.data, error: "" };
  return { ok: false, value: null, error: got.error.issues[0].message };
}
