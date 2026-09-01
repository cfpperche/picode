import { api, humanizeError } from "./api.js";
import { createWorkspaceSchema, createWorkspaceCloneSchema, createFreeAgentSchema, createWsAgentSchema, parseForm } from "./schemas.js";

// The create flow's request builder, lifted out of desktop/App.jsx's
// submitNew so the phone's sheet (ADR-0044) posts exactly what the desktop
// dialog does. Pure: form values in, { path, body } or { error } out.
//   kind: "workspace" | "free" | "agent"
//   values: { name, path, source, url }   (FormData of CreateForm)
//   cfg: { provider, model, thinking }    (ConfigFields)
export function createRequest(kind, values, cfg, workspaceId) {
  const v = values || {};
  const name = String(v.name || "");
  const path = String(v.path || "");
  if (kind === "workspace" && String(v.source || "") === "remote") {
    const parsed = parseForm(createWorkspaceCloneSchema, { url: String(v.url || ""), name, path });
    if (!parsed.ok) return { error: parsed.error };
    return { path: "/api/workspaces/clone", body: parsed.value, clone: true };
  }
  const schema = kind === "workspace" ? createWorkspaceSchema : kind === "free" ? createFreeAgentSchema : createWsAgentSchema;
  const parsed = parseForm(schema, kind === "workspace" ? { name, path } : { name, path, ...(cfg || {}) });
  if (!parsed.ok) return { error: parsed.error };
  const body = parsed.value;
  if (kind === "workspace") return { path: "/api/workspaces", body };
  if (kind === "free") return { path: "/api/agents", body };
  if (!workspaceId) return { error: "Pick a workspace first." };
  return {
    path: "/api/workspaces/" + encodeURIComponent(workspaceId) + "/agents",
    body: { name: body.name, provider: body.provider, model: body.model, thinking: body.thinking },
  };
}

export function formValues(form) {
  const fd = new FormData(form);
  return { name: fd.get("name"), path: fd.get("path"), source: fd.get("source"), url: fd.get("url") };
}

// submitCreate posts the request; resolves with the created object (a
// workspace or an agent) or rejects with a humanised message.
export async function submitCreate(kind, values, cfg, workspaceId) {
  const req = createRequest(kind, values, cfg, workspaceId);
  if (req.error) throw new Error(req.error);
  try {
    const created = await api(req.path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(req.body) });
    return { kind, created, clone: !!req.clone };
  } catch (e) {
    throw new Error(humanizeError((e && e.message) || String(e)));
  }
}
