import { useCallback, useEffect, useRef, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import PageFrame from "./PageFrame.jsx";
import { api } from "@picode/shared/client/api.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { askPrompt } from "../lib/prompt.js";
import { apiKeySchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { cliPaneHash } from "@picode/shared/domain/cliLaunch.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { identityLine } from "@picode/shared/domain/providerRows.js";
import { vaultProblemText } from "@picode/shared/domain/credentials.js";
import { healthChip, kindLabel, orderAccounts, providerName, sourceLabel, VERIFY_LABEL } from "@picode/shared/domain/credentials.js";

// The vault roster for a guest CLI (ADR-0165): every account this CLI can use,
// the login it already has of its own, and the four verbs — Import, Add API
// key, Verify, Sign out — plus rename and pause. This pane never activates a
// credential for pi: pi's own editor writes auth.json, this one writes the
// shared vault. Both apps carry this file (ADR-0072), so the only difference
// between them is the modal primitive on line 2.

const credPath = (provider, id, suffix = "") =>
  "/api/credentials/" + encodeURIComponent(provider) + "/" + encodeURIComponent(id) + suffix;

const json = (method, body) => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

export default function CliCredentials({ hidden, cli, add = false }) {
  const [data, setData] = useState(null);
  const [problem, setProblem] = useState(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState("");
  const [nativeError, setNativeError] = useState({});
  const [addOpen, setAddOpen] = useState(false);
  const [form, setForm] = useState({ provider: "", key: "" });
  const [formError, setFormError] = useState("");
  const [saving, setSaving] = useState(false);
  const live = useRef(false);
  const sequence = useRef(0);

  // The vault is the daemon's file, not a vendor's: a load is one local
  // request, and a refresh keeps the last good roster on screen.
  const load = useCallback(async () => {
    const request = ++sequence.current;
    setLoading(true);
    try {
      const next = await api("/api/credentials?cli=" + encodeURIComponent(cli));
      if (!live.current || request !== sequence.current) return;
      setData(next);
      setProblem(null);
    } catch (ex) {
      if (live.current && request === sequence.current) setProblem({ message: ex.message, missing: ex.status === 404 });
    } finally {
      if (live.current && request === sequence.current) setLoading(false);
    }
  }, [cli]);

  useEffect(() => {
    live.current = true;
    load();
    window.addEventListener("focus", load);
    return () => { live.current = false; sequence.current++; window.removeEventListener("focus", load); };
  }, [load]);

  const providers = data && Array.isArray(data.providers) ? data.providers : [];
  const cliName = (data && data.cliName) || terminalCliLabel(cli);
  const unreadable = !!(data && data.vault && data.vault.readable === false);
  // The server knows no credential mechanism for this CLI: the pane is one
  // line, and any roster a previous load left on screen goes with it.
  const missing = !!(problem && problem.missing);

  function openAdd(provider) {
    setForm({ provider: provider || (providers[0] && providers[0].id) || "", key: "" });
    setFormError("");
    setAddOpen(true);
  }

  function closeAdd() {
    setAddOpen(false);
    setForm({ provider: "", key: "" });
    setFormError("");
    if (add && typeof location !== "undefined") location.hash = cliPaneHash(cli, "providers");
  }

  useEffect(() => {
    if (add && data && !unreadable) openAdd("");
    // The route asked for the dialog; the roster is what fills its provider list.
  }, [add, !!data, unreadable]);

  async function run(key, action, done) {
    setBusy(key);
    try {
      await action();
      await load();
      if (done) done();
    } catch (ex) {
      toastError(ex);
    } finally {
      setBusy("");
    }
  }

  async function save(e) {
    e.preventDefault();
    const parsed = parseForm(apiKeySchema, { key: form.key });
    if (!parsed.ok) { setFormError(parsed.error); return; }
    setSaving(true);
    setFormError("");
    try {
      await api("/api/credentials", json("POST", { provider: form.provider, label: "", key: parsed.value.key }));
      setSaving(false);
      closeAdd();
      toast.ok("Key saved to the vault.");
      await load();
    } catch (ex) {
      // The form owns the error: a toast would leave the dialog looking fine.
      setSaving(false);
      setFormError(ex.message);
    }
  }

  async function importLogin(provider) {
    setNativeError((prev) => ({ ...prev, [provider.id]: "" }));
    setBusy("import");
    try {
      await api("/api/credentials/import", json("POST", { cli }));
      toast.ok("Saved to the vault.");
      await load();
    } catch (ex) {
      // The CLI's own row owns this failure — it is what the person clicked.
      setNativeError((prev) => ({ ...prev, [provider.id]: ex.message }));
    } finally {
      setBusy("");
    }
  }

  async function rename(provider, account) {
    const name = await askPrompt({
      title: "Rename " + (account.label || providerName(provider.id)),
      message: "The name only — the credential is untouched.",
      defaultValue: account.label || "",
      confirmLabel: "Save",
    });
    if (!name || name === account.label) return;
    await run("rename:" + account.id, () => api(credPath(provider.id, account.id), json("PATCH", { label: name })));
  }

  async function verify(provider, account) {
    // One listing call, on this click, with the cost on the control that
    // spends it; the row carries the answer afterwards.
    await run("verify:" + account.id, () => api(credPath(provider.id, account.id, "/verify"), { method: "POST" }));
  }

  async function pause(provider, account) {
    const paused = !account.paused;
    await run("pause:" + account.id, () => api(credPath(provider.id, account.id, "/pause"), json("POST", { paused })),
      () => toast.ok(paused ? "Paused. The credential is kept." : "Resumed."));
  }

  async function signOut(provider, account) {
    const identity = identityLine(account);
    const ok = await askConfirm({
      title: "Sign out " + (account.label || providerName(provider.id)),
      message: "Removes this login from the vault." + (identity ? " " + identity + "." : ""),
      confirmLabel: "Sign out",
      danger: true,
    });
    if (!ok) return;
    await run("remove:" + account.id, () => api(credPath(provider.id, account.id), { method: "DELETE" }),
      () => toast.ok("Signed out."));
  }

  const total = providers.reduce((n, p) => n + ((p.accounts || []).length), 0);

  return (
    <PageFrame id="cli-credentials-view" title="Providers" hidden={hidden} embedded>
      {/* The vault cannot be read: say why, and offer nothing that would fail. */}
      {unreadable ? (
        <div className="cli-notice" role="status">
          <span>{vaultProblemText(data.vault && data.vault.problem)}</span>
        </div>
      ) : null}

      {problem && !problem.missing ? (
        <div className="cli-notice is-error" role="alert">
          <span>{problem.message}</span>
          <button type="button" className="btn btn-ghost btn-sm" disabled={loading} onClick={load}>
            {loading ? "Retrying…" : "Try again"}
          </button>
        </div>
      ) : null}
      {/* The server knows no credential mechanism for this CLI: one line, and
          no Add button — it could only fail. */}
      {missing ? (
        <div className="cli-notice" role="status"><span>{"No credential store for " + cliName + " yet."}</span></div>
      ) : null}

      {!data && !problem ? (
        <div className="cred-loading" aria-label="Loading accounts" role="status">
          <div className="cred-skel-head" />
          <div className="cred-skel-row" />
          <div className="cred-skel-row" />
          <div className="cred-skel-head" />
          <div className="cred-skel-row" />
        </div>
      ) : null}

      {data && !unreadable && !missing ? <>
        {providers.length ? (
          <div className="set-row cred-bar" data-align-row>
            {total ? <span className="cred-count">{total === 1 ? "1 account" : total + " accounts"}</span> : null}
            <button type="button" className="btn btn-primary btn-sm" onClick={() => openAdd("")}>Add API key</button>
          </div>
        ) : (
          <p className="cred-empty"><span>{"No provider credentials for " + cliName + " yet."}</span></p>
        )}

        <section className="cred-pane" aria-label={cliName + " accounts"}>
          {providers.map((p) => {
            const name = providerName(p.id);
            const note = String(p.note || "").trim();
            const rows = orderAccounts(p.accounts);
            return (
              <div key={p.id} className="cred-provider">
                <div className="cred-provider-head">
                  <h3 className="cred-provider-name">{name}</h3>
                  {name === p.id ? null : <span className="cred-provider-id">{p.id}</span>}
                  {rows.length ? (
                    <button type="button" className="btn btn-ghost btn-sm" onClick={() => openAdd(p.id)}>Add API key</button>
                  ) : null}
                </div>
                {/* A provider that cannot take a credential says why, once. */}
                {note ? <p className="cred-note">{note}</p> : null}

                {/* The CLI's own login is not a vault row yet: one action. */}
                {p.native && !p.native.imported ? (
                  <div className="cred-native">
                    <span className="cred-native-text">{(p.native.label || cliName) + " is signed in here"}</span>
                    {kindLabel(p.native.kind) ? <span className="cred-chip">{kindLabel(p.native.kind)}</span> : null}
                    <button
                      type="button"
                      className="btn btn-primary btn-sm"
                      disabled={busy === "import"}
                      onClick={() => importLogin(p)}
                    >
                      {busy === "import" ? "Importing…" : "Import into the vault"}
                    </button>
                  </div>
                ) : null}
                {nativeError[p.id] ? <p className="cred-note is-error" role="alert">{nativeError[p.id]}</p> : null}

                {rows.length ? (
                  <ul className="cred-rows">
                    {rows.map((a) => (
                      <AccountRow
                        key={a.id}
                        provider={p}
                        account={a}
                        note={note}
                        busy={busy}
                        onRename={() => rename(p, a)}
                        onVerify={() => verify(p, a)}
                        onPause={() => pause(p, a)}
                        onSignOut={() => signOut(p, a)}
                      />
                    ))}
                  </ul>
                ) : (
                  <p className="cred-empty">
                    <span>{name + " · No accounts yet."}</span>
                    <button type="button" className="btn btn-ghost btn-sm" onClick={() => openAdd(p.id)}>Add API key</button>
                  </p>
                )}
              </div>
            );
          })}
        </section>
      </> : null}

      <Dialog.Root open={addOpen} onOpenChange={(open) => { if (!open) closeAdd(); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg">
            <Dialog.Title className="dlg-title">Add API key</Dialog.Title>
            <Dialog.Description className="dlg-body">{"Saved to this machine’s vault."}</Dialog.Description>
            <form className="cred-form" noValidate onSubmit={save}>
              <label className="cred-field">
                <span>Provider</span>
                <select
                  className="cred-input"
                  value={form.provider}
                  onChange={(e) => setForm({ provider: e.target.value, key: form.key })}
                >
                  {providers.map((p) => <option key={p.id} value={p.id}>{providerName(p.id)}</option>)}
                </select>
              </label>
              <label className="cred-field">
                <span>API key</span>
                <input
                  className="cred-input"
                  type="password"
                  autoComplete="off"
                  spellCheck={false}
                  placeholder="sk-…"
                  value={form.key}
                  onChange={(e) => setForm({ provider: form.provider, key: e.target.value })}
                />
                <em className="cred-help">This key stays on this machine. Nothing here uploads it.</em>
              </label>
              <p className="form-error" role="alert" hidden={!formError}>{formError}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={closeAdd}>Cancel</button>
                <button type="submit" className="btn btn-primary btn-sm" disabled={saving}>{saving ? "Saving…" : "Save"}</button>
              </div>
            </form>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </PageFrame>
  );
}

function AccountRow({ provider, account, note, busy, onRename, onVerify, onPause, onSignOut }) {
  const identity = identityLine(account);
  const kind = kindLabel(account.type);
  const source = sourceLabel(account.origin);
  const chip = healthChip(account.health);
  const label = account.label || providerName(provider.id);
  const title = chip
    ? [chip.message, chip.age ? (chip.age === "now" ? "Checked just now." : "Checked " + chip.age + " ago.") : ""].filter(Boolean).join(" ")
    : "";
  return (
    <li className={"cred-row" + (account.paused ? " is-paused" : "")}>
      <span className="cred-cell">
        <span className="cred-label" title={label}>{label}</span>
      </span>
      <span className="cred-cell cred-ident" data-empty={identity ? undefined : true} title={identity || undefined}>
        {identity || <span className="cred-none">—</span>}
      </span>
      <span className="cred-cell cred-chips">
        {kind ? <span className="cred-chip">{kind}</span> : null}
        {source ? <span className="cred-chip">{source}</span> : null}
        {account.paused ? <span className="cred-chip is-muted">Paused</span> : null}
      </span>
      {/* The masked echo of the stored key — never the key. */}
      <span className="cred-cell cred-hint" data-empty={account.hint ? undefined : true} title={account.hint || undefined}>
        {account.hint || ""}
      </span>
      <span className="cred-cell cred-health">
        {chip ? <span className={"cred-chip is-" + chip.tone} title={title || undefined}>{chip.label}</span> : null}
        {chip && chip.message ? <span className="cred-health-msg" title={chip.message}>{chip.message}</span> : null}
      </span>
      <span className="cred-actions">
        <DropdownMenu.Root>
          <DropdownMenu.Trigger asChild>
            <button type="button" className="btn btn-ghost btn-sm cred-more" aria-label={"More actions for " + label}>⋯</button>
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content className="prov-menu cred-menu" align="end" sideOffset={6} collisionPadding={8}>
              <DropdownMenu.Item className="prov-menu-item" onSelect={onRename}>Rename</DropdownMenu.Item>
              {/* Verify spends a listing call; the button names its cost. A
                  provider with a note has no honest check, so it hides. */}
              {provider.verifier && !note ? (
                <DropdownMenu.Item className="prov-menu-item" onSelect={onVerify}>
                  {busy === "verify:" + account.id ? "Verifying…" : VERIFY_LABEL}
                </DropdownMenu.Item>
              ) : null}
              <DropdownMenu.Item className="prov-menu-item" onSelect={onPause}>
                {account.paused ? "Resume" : "Pause"}
              </DropdownMenu.Item>
              <DropdownMenu.Item className="prov-menu-item danger" onSelect={onSignOut}>Sign out</DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      </span>
    </li>
  );
}
