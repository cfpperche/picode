import { useState } from "react";
import { api } from "../lib/api.js";
import { askConfirm } from "../lib/confirm.js";
import { toast } from "../lib/toast.js";
import { cliProfileSchema, cliLaunchSchema, parseForm } from "../lib/schemas.js";
import { launchDraft, launchConfig } from "../lib/cliLaunch.js";
import { LaunchFields, LaunchPreview, cliJSON, useLaunchGuard, confirmDiscard } from "./CliLaunchSettings.jsx";

export function CLIProfiles({ cli, profiles, run, busy }) {
  const rows = profiles.filter((p) => p.cli === cli.id);
  return <details className="cli-profiles"><summary>Launch profiles{rows.length ? ` · ${rows.length}` : ""}</summary>
    <div className="cli-section-heading"><h4>Reusable launch settings</h4><a className="btn btn-ghost btn-sm" href={`#/clis/profile/new/${cli.id}`}>New profile</a></div>
    {!rows.length ? <p className="cli-muted">No profiles for {cli.name} yet.</p> : rows.map((p) => <div className="cli-profile-row" key={p.id}><strong>{p.name}</strong><div className="cli-actions" data-align-row>
      <a className="btn btn-ghost btn-sm" href={`#/clis/new/${cli.id}?profile=${encodeURIComponent(p.id)}`}>Use</a>
      <a className="btn btn-ghost btn-sm" href={`#/clis/profile/${encodeURIComponent(p.id)}`}>Edit</a>
      <button className="btn btn-ghost btn-sm" disabled={busy} onClick={async () => {
        if (!(await askConfirm({ title: `Remove ${p.name}?`, message: "Existing terminals keep their copied settings.", confirmLabel: "Remove profile", danger: true }))) return;
        run("profile-remove", () => api(`/api/clis/profiles/${encodeURIComponent(p.id)}`, { method: "DELETE" })).catch(() => {});
      }}>Remove</button>
    </div></div>)}
  </details>;
}

export function CLIProfileEditor({ route, data, run, busy }) {
  const existing = data.profiles.find((p) => p.id === route.id);
  const cli = data.clis.find((c) => c.id === (existing?.cli || route.cli)) || data.clis[0];
  const initial = launchDraft(existing?.config || cli.config);
  const [name, setName] = useState(existing?.name || "");
  const [draft, setDraft] = useState(initial);
  const [error, setError] = useState("");
  const dirty = name !== (existing?.name || "") || JSON.stringify(draft) !== JSON.stringify(initial);
  const allowNavigation = useLaunchGuard(dirty);
  const settings = parseForm(cliLaunchSchema, draft);
  if (route.id !== "new" && !existing) return <div className="cli-notice"><span>That profile is gone.</span><a className="btn btn-ghost btn-sm" href="#/clis">Back to CLIs</a></div>;
  return <section className="cli-editor cli-profile-editor"><h3>{existing ? "Edit profile" : `New ${cli.name} profile`}</h3><form noValidate onSubmit={async (e) => {
    e.preventDefault(); const parsed = parseForm(cliProfileSchema, { name });
    if (!parsed.ok || !settings.ok) { setError(parsed.error || settings.error); return; }
    try { await run("profile-save", async () => {
      const id = existing?.id || crypto.randomUUID();
      await api(`/api/clis/profiles/${id}`, cliJSON("PUT", { cli: cli.id, name: parsed.value.name, config: launchConfig(settings.value) }));
      allowNavigation(); toast.ok("Launch profile saved."); location.hash = "#/clis/" + cli.id;
    }); } catch (e) { setError(e.message); }
  }}>
    <div className="cli-fields"><label>Profile name<input value={name} onChange={(e) => setName(e.target.value)} autoComplete="off" /></label></div>
    <LaunchFields draft={draft} setDraft={setDraft} includeIntegration />
    {error ? <p className="cli-field-error" role="alert">{error}</p> : null}
    <div className="cli-actions" data-align-row><button className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Saving…" : "Save profile"}</button><button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={async () => { if (!dirty || await confirmDiscard()) { allowNavigation(); location.hash = "#/clis/" + cli.id; } }}>Cancel</button></div>
    {settings.ok ? <LaunchPreview cli={cli.id} config={launchConfig(settings.value)} /> : null}
  </form></section>;
}
