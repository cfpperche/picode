import { useEffect, useMemo, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { Command } from "cmdk";
import PageFrame from "./PageFrame.jsx";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";
import { apiKeySchema, llamaLoginSchema, parseForm } from "../lib/schemas.js";
import { go } from "../lib/routes.js";
import LlamaPanel from "./LlamaPanel.jsx";
import { ProviderFace } from "./ProviderFaces.jsx";
import { readRecents, pushRecent, removeRecent, clearRecents, rememberProviders } from "../lib/providerRecents.js";
import { askConfirm } from "../lib/confirm.js";
import { showUsageButton, usagePath } from "../lib/providerUsage.js";
import UsageDialog from "./UsageDialog.jsx";
import QuotaStrip from "./QuotaStrip.jsx";
import {
  blastRadius, formatSpend, identityLine, indexUsage, matchesQuery,
  sourceLabel, spendByProvider, usageKey,
} from "../lib/providerRows.js";

function AccountName({ provider, acc, onSaved }) {
  const [editing, setEditing] = useState(false);
  const [val, setVal] = useState(acc.label);
  useEffect(() => { setVal(acc.label); }, [acc.label]);
  async function save() {
    const name = val.trim();
    setEditing(false);
    if (!name || name === acc.label) { setVal(acc.label); return; }
    try {
      await api("/api/providers/" + encodeURIComponent(provider) + "/accounts/" + encodeURIComponent(acc.id), {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ label: name }),
      });
      if (onSaved) await onSaved();
    } catch (ex) {
      toastError(ex);
      setVal(acc.label);
    }
  }
  if (!editing) {
    return <button type="button" className="prov-acc-label" title="Rename" onClick={() => setEditing(true)}>{acc.label}</button>;
  }
  return (
    <input
      className="prov-acc-input"
      value={val}
      autoFocus
      onChange={(e) => setVal(e.target.value)}
      onBlur={save}
      onKeyDown={(e) => {
        if (e.key === "Enter") { e.preventDefault(); e.currentTarget.blur(); }
        if (e.key === "Escape") { setVal(acc.label); setEditing(false); }
      }}
    />
  );
}

export default function Providers({ hidden, catalog, onSignOut, onRefresh, wantAdd, wantLlama }) {
  const list = catalog && catalog.providers ? catalog.providers : [];
  const signed = list.filter((p) => p.signedIn);
  const [add, setAdd] = useState(false);
  const [step, setStep] = useState("pick"); // pick | method | key | oauth
  const [pick, setPick] = useState(null);
  const [key, setKey] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const [userCode, setUserCode] = useState("");
  const [replacing, setReplacing] = useState(false);
  const [llamaUrl, setLlamaUrl] = useState("http://127.0.0.1:8080");
  const [recents, setRecents] = useState(readRecents);
  const [usageFor, setUsageFor] = useState(null);
  const [query, setQuery] = useState("");
  const [usageRows, setUsageRows] = useState(() => new Map());
  const [spend, setSpend] = useState(() => new Map());
  const [checking, setChecking] = useState("");
  const [verifying, setVerifying] = useState("");
  const [verdicts, setVerdicts] = useState({});

  // The roster reads the server's usage cache — never a vendor. The poll is
  // a local request that keeps the "fetched 4m ago" reading honest while the
  // page is open; the daemon is what actually talks to the providers.
  useEffect(() => {
    if (hidden) return undefined;
    let live = true;
    const load = () => {
      api("/api/providers/usage")
        .then((r) => { if (live) setUsageRows(indexUsage(r && r.entries)); })
        .catch(() => {});
    };
    load();
    const t = setInterval(load, 60000);
    return () => { live = false; clearInterval(t); };
  }, [hidden]);

  // Our own spend, from session files — the same aggregate the dashboard
  // shows, landed next to the account that produced it.
  useEffect(() => {
    if (hidden) return undefined;
    let live = true;
    api("/api/sessions/stats?range=7d")
      .then((s) => { if (live) setSpend(spendByProvider(s)); })
      .catch(() => {});
    return () => { live = false; };
  }, [hidden]);

  useEffect(() => {
    if (hidden || !wantAdd) return;
    openAdd();
  }, [hidden, wantAdd]);

  useEffect(() => {
    if (hidden || !wantLlama) return;
    const p = list.find((x) => x.id === "llama.cpp");
    if (p && !p.signedIn) {
      chooseProvider(p);
      setAdd(true);
      return;
    }
    requestAnimationFrame(() => document.getElementById("llama-panel")?.scrollIntoView({ block: "start" }));
  }, [hidden, wantLlama]);

  useEffect(() => {
    setRecents(rememberProviders(signed.map((p) => p.id)));
  }, [signed.map((p) => p.id).join(",")]);

  const available = useMemo(
    () => list.filter((p) => !p.signedIn).sort((a, b) => a.id.localeCompare(b.id)),
    [list],
  );
  const recentRows = recents
    .map((id) => list.find((p) => p.id === id) || { id, signedIn: false, login: "api_key" })
    .filter((p) => !p.signedIn);

  const rosterRows = useMemo(
    () => signed.filter((p) => matchesQuery(p, query)),
    [signed.map((p) => p.id).join(","), query, catalog],
  );

  // One row, fetched now. The dialog still exists for windows, banked resets
  // and money left; this is the reading the roster shows without it.
  async function refreshRow(provider, accountId) {
    const k = usageKey(provider, accountId);
    setChecking(k);
    try {
      const rep = await api(usagePath(provider, accountId));
      setUsageRows((prev) => {
        const next = new Map(prev);
        next.set(k, {
          provider,
          accountId: accountId === "live" ? "" : accountId,
          status: rep.status,
          plan: rep.plan,
          email: rep.email,
          error: rep.error,
          windows: rep.windows || [],
          resets: (rep.resets || []).length,
          fetchedAt: rep.fetchedAt,
          ageSec: 0,
        });
        return next;
      });
    } catch (ex) {
      toastError(ex);
    } finally {
      setChecking("");
    }
  }

  // Verify asks pi, not the vendor: pi is what runs the agent, so pi's
  // answer is the one that matters. No token is spent on a test completion.
  async function verifyProvider(p) {
    setVerifying(p.id);
    try {
      const res = await api("/api/providers/" + encodeURIComponent(p.id) + "/verify", { method: "POST" });
      setVerdicts((prev) => ({ ...prev, [p.id]: res }));
      if (res && res.ok) toast.ok("pi can use " + p.id + ".");
      else toast.error(p.id + ": " + ((res && (res.reason || res.status)) || "not ready"));
    } catch (ex) {
      toastError(ex);
    } finally {
      setVerifying("");
    }
  }

  async function pauseAccount(p, acc, paused) {
    try {
      await api("/api/providers/" + encodeURIComponent(p.id) + "/accounts/" + encodeURIComponent(acc.id) + "/pause", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ paused }),
      });
      toast.ok(paused ? "Paused. The credential is kept." : "Resumed.");
      if (onRefresh) await onRefresh();
    } catch (ex) { toastError(ex); }
  }

  async function useAccount(provider, aid) {
    try {
      await api("/api/providers/" + encodeURIComponent(provider) + "/accounts/" + encodeURIComponent(aid) + "/activate", { method: "POST" });
      toast.ok("Using this account.");
      if (onRefresh) await onRefresh();
    } catch (ex) { toastError(ex); }
  }

  async function removeAccount(provider, acc, p) {
    const radius = blastRadius(p);
    const ok = await askConfirm({
      title: "Sign out " + (acc.label || provider),
      message: radius
        ? "Remove this login from this machine. " + radius
        : "Remove this login from this machine.",
      confirmLabel: "Sign out",
      danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/providers/" + encodeURIComponent(provider) + "/accounts/" + encodeURIComponent(acc.id), { method: "DELETE" });
      setRecents(pushRecent(provider));
      toast.ok("Signed out.");
      if (onRefresh) await onRefresh();
    } catch (ex) { toastError(ex); }
  }

  function openAdd() {
    setPick(null);
    setStep("pick");
    setKey("");
    setErr("");
    setUserCode("");
    setReplacing(false);
    setLlamaUrl("http://127.0.0.1:8080");
    setAdd(true);
  }

  function closeAdd() {
    setAdd(false);
    setPick(null);
    setKey("");
    setErr("");
    setUserCode("");
    setReplacing(false);
    if (wantAdd) go("providers");
  }

  function replaceProvider(p) {
    setReplacing(true);
    setAdd(true);
    chooseProvider(p);
  }

  function goBack() {
    if ((step === "key" || step === "oauth") && pick && pick.login === "both") {
      setStep("method");
      return;
    }
    if (replacing) { closeAdd(); return; }
    setPick(null);
    setStep("pick");
  }

  function chooseProvider(p) {
    setPick(p);
    setErr("");
    if (p.id === "llama.cpp") setStep("llama");
    else if (p.login === "oauth") setStep("oauth");
    else if (p.login === "both") setStep("method");
    else setStep("key");
  }

  async function save(e) {
    e.preventDefault();
    const parsed = parseForm(apiKeySchema, { key });
    if (!parsed.ok) { setErr(parsed.error); return; }
    setBusy(true);
    setErr("");
    try {
      await api("/api/providers/" + encodeURIComponent(pick.id), {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key: parsed.value.key }),
      });
      setRecents(pushRecent(pick.id));
      toast.ok("Signed in to " + pick.id + ".");
      closeAdd();
      if (onRefresh) await onRefresh();
    } catch (ex) {
      toastError(ex);
    } finally {
      setBusy(false);
    }
  }

  async function saveLlama(e) {
    e.preventDefault();
    const parsed = parseForm(llamaLoginSchema, { url: llamaUrl, key });
    if (!parsed.ok) { setErr(parsed.error); return; }
    setBusy(true);
    setErr("");
    try {
      await api("/api/providers/llama.cpp", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: parsed.value.url, key: parsed.value.key || "" }),
      });
      setRecents(pushRecent("llama.cpp"));
      toast.ok("Signed in to llama.cpp.");
      closeAdd();
      if (onRefresh) await onRefresh();
    } catch (ex) {
      toastError(ex);
    } finally {
      setBusy(false);
    }
  }

  const canAccount = pick && ["anthropic", "openai-codex", "github-copilot", "kimi-coding", "xai"].includes(String(pick.id).toLowerCase());
  const title = !pick ? "Add provider" : step === "method" || step === "oauth" || step === "llama" ? pick.id : "API key · " + pick.id;

  async function startAccount() {
    if (!pick) return;
    setBusy(true);
    setErr("");
    try {
      const res = await api("/api/oauth/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ provider: pick.id, returnTo: location.origin + "/#/providers" }),
      });
      if (res && res.userCode) setUserCode(res.userCode);
      if (res && res.url) window.open(res.url, "_blank", "noopener");
      setWaiting(true);
      const t0 = Date.now();
      while (Date.now() - t0 < 5 * 60 * 1000) {
        await new Promise((r) => setTimeout(r, 1000));
        const st = await api("/api/oauth/status");
        if (st && st.done) {
          setWaiting(false);
          if (st.error) { setErr(st.error); return; }
          toast.ok("Signed in to " + pick.id + ".");
          setRecents(pushRecent(pick.id));
          closeAdd();
          if (onRefresh) await onRefresh();
          return;
        }
        if (st && !st.pending && !st.done) break;
      }
      setWaiting(false);
    } catch (ex) {
      toastError(ex);
      setWaiting(false);
    } finally {
      setBusy(false);
    }
  }

  return (
    <PageFrame id="providers-view" title="Providers" hidden={hidden}>
      <div className="set-row prov-bar">
        {signed.length > 3 ? (
          <input
            className="prov-search"
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search providers and accounts"
            aria-label="Search providers and accounts"
          />
        ) : <span />}
        <button type="button" className="btn btn-primary btn-sm" onClick={openAdd}>Add provider</button>
      </div>
      <section className="settings-section">
        {signed.length === 0 ? (
          <p className="side-empty">No providers yet. Add a provider to sign in.</p>
        ) : (
          <ul className="prov-list">
            {rosterRows.length === 0 ? (
              <li className="side-empty">No provider matches “{query}”.</li>
            ) : null}
            {rosterRows.map((p) => {
              const envVar = sourceLabel(p);
              const accs = p.accounts && p.accounts.length
                ? p.accounts
                : [{ id: "live", label: envVar || "Default", type: p.authType, active: true, quotaKind: p.quotaKind }];
              const money = formatSpend(spend.get(p.id));
              // Pause only makes sense while another live account can take
              // over; on the last one it is Sign out, and the server says so.
              const liveAccounts = accs.filter((x) => !x.paused).length;
              return (
                <li key={p.id} className="prov-group">
                  <div className="prov-row prov-head">
                    <ProviderFace id={p.id} />
                    <span className="prov-id">{p.id}</span>
                    {money ? <span className="prov-spend" title="What your sessions spent on this provider in the last 7 days">{money} · 7d</span> : null}
                    {envVar ? null : (
                      <button type="button" className="btn btn-ghost btn-sm" onClick={() => replaceProvider(p)}>Add account</button>
                    )}
                  </div>
                  <ul className="prov-accounts">
                    {accs.map((a) => {
                      const k = usageKey(p.id, a.id);
                      const entry = usageRows.get(k);
                      const identity = identityLine(a, entry);
                      const meterable = showUsageButton(p, a);
                      const verdict = verdicts[p.id];
                      return (
                        <li key={a.id} className={"prov-acc" + (a.active && !a.paused ? "" : " muted")}>
                          <div className="prov-acc-top">
                            {envVar ? (
                              <span className="prov-acc-fixed" title={"pi reads this key from " + envVar}>{envVar}</span>
                            ) : (
                              <AccountName provider={p.id} acc={a} onSaved={onRefresh} />
                            )}
                            <span className={"prov-auth" + (a.active && !a.paused ? " in" : "")}>
                              {envVar ? "environment" : a.type === "oauth" ? "account" : "api key"}
                              {a.paused ? " · paused" : a.active ? " · active" : ""}
                            </span>
                            {verdict ? (
                              <span className={"prov-verdict " + (verdict.ok ? "ok" : "bad")} title={verdict.reason || verdict.status}>
                                {verdict.ok ? "pi can use this" : verdict.reason || verdict.status}
                              </span>
                            ) : null}
                            <span className="prov-acc-gap" />
                            {meterable ? (
                              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setUsageFor({ provider: p, account: a })}>Usage</button>
                            ) : null}
                            {!a.active && !a.paused && !envVar ? (
                              <button type="button" className="btn btn-ghost btn-sm" onClick={() => useAccount(p.id, a.id)}>Use</button>
                            ) : null}
                            <DropdownMenu.Root>
                              <DropdownMenu.Trigger asChild>
                                <button type="button" className="btn btn-ghost btn-sm prov-more" aria-label={"More actions for " + (a.label || p.id)}>⋯</button>
                              </DropdownMenu.Trigger>
                              <DropdownMenu.Portal>
                                <DropdownMenu.Content className="prov-menu" align="end" sideOffset={6} collisionPadding={8}>
                                  <DropdownMenu.Item className="prov-menu-item" onSelect={() => verifyProvider(p)}>
                                    {verifying === p.id ? "Checking…" : "Verify with pi"}
                                  </DropdownMenu.Item>
                                  {!envVar && a.id !== "live" && (a.paused || liveAccounts > 1) ? (
                                    <DropdownMenu.Item className="prov-menu-item" onSelect={() => pauseAccount(p, a, !a.paused)}>
                                      {a.paused ? "Resume" : "Pause"}
                                    </DropdownMenu.Item>
                                  ) : null}
                                  {envVar ? (
                                    <DropdownMenu.Item className="prov-menu-item" disabled>Set by {envVar}</DropdownMenu.Item>
                                  ) : (
                                    <DropdownMenu.Item className="prov-menu-item danger" onSelect={() => removeAccount(p.id, a, p)}>
                                      Sign out
                                    </DropdownMenu.Item>
                                  )}
                                </DropdownMenu.Content>
                              </DropdownMenu.Portal>
                            </DropdownMenu.Root>
                          </div>
                          {identity || meterable ? (
                            <div className="prov-acc-bottom">
                              {identity ? <span className="prov-ident">{identity}</span> : <span />}
                              {meterable ? (
                                <QuotaStrip
                                  entry={entry}
                                  busy={checking === k}
                                  onRefresh={() => refreshRow(p.id, a.id)}
                                />
                              ) : null}
                            </div>
                          ) : null}
                        </li>
                      );
                    })}
                  </ul>
                </li>
              );
            })}
          </ul>
        )}
      </section>
      {signed.some((p) => p.id === "llama.cpp") ? <LlamaPanel onRefresh={onRefresh} /> : null}
      {recentRows.length ? (
        <section className="settings-section">
          <div className="set-row">
            <h3>Recently used</h3>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => setRecents(clearRecents())}>Clear</button>
          </div>
          <ul className="prov-list">
            {recentRows.map((p) => (
              <li key={p.id} className="prov-row muted">
                <ProviderFace id={p.id} />
                <span className="prov-id">{p.id}</span>
                <span className="prov-auth">signed out</span>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => { chooseProvider(p); setAdd(true); }}>Sign in</button>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => setRecents(removeRecent(p.id))}>Remove</button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      <UsageDialog
        provider={usageFor && usageFor.provider}
        account={usageFor && usageFor.account}
        onClose={() => setUsageFor(null)}
        onSignIn={() => {
          const p = usageFor && usageFor.provider;
          setUsageFor(null);
          if (p) replaceProvider(p);
        }}
      />
      <Dialog.Root open={add} onOpenChange={(o) => { if (!o) closeAdd(); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
            <Dialog.Title className="dlg-title">{title}</Dialog.Title>
            <Dialog.Description className="dlg-body">
              {step === "pick" ? "Pick a provider." : step === "method" ? "Choose how to sign in." : step === "llama" ? "Router URL. API key is optional." : step === "oauth" ? (userCode ? "Enter this code in the browser tab." : canAccount ? "Finish sign-in in the browser tab." : "Account login is not available here. Use an API key.") : "Paste the key. It is not shown again."}
            </Dialog.Description>

            {step === "pick" ? (
              <Command loop className="prov-pick">
                <Command.Input className="combo-input" placeholder="Search providers" />
                <Command.List className="prov-pick-list">
                  <Command.Empty className="combo-empty">No matches</Command.Empty>
                  {available.map((p) => (
                    <Command.Item key={p.id} value={p.id + " " + p.login} className="cockpit-opt" onSelect={() => chooseProvider(p)}>
                      <ProviderFace id={p.id} />
                      <span>{p.id}</span>
                      <span className="combo-hint">{p.id === "llama.cpp" ? "local router" : p.login === "both" ? "account or api key" : p.login === "oauth" ? "account" : "api key"}</span>
                    </Command.Item>
                  ))}
                </Command.List>
              </Command>
            ) : null}

            {step === "method" ? (
              <div className="prov-methods">
                <button type="button" className="btn btn-primary" onClick={() => setStep("key")}>Sign in with an API key</button>
                <button type="button" className="btn btn-ghost" onClick={() => setStep("oauth")}>Sign in with an account</button>
                <div className="dlg-actions">
                  <button type="button" className="btn btn-ghost btn-sm" onClick={goBack}>Back</button>
                </div>
              </div>
            ) : null}

            {step === "oauth" ? (
              <div>
                {userCode ? <p className="oauth-code">{userCode}</p> : null}
                <p className="form-error" hidden={!err}>{err}</p>
                <div className="dlg-actions">
                  <button type="button" className="btn btn-ghost btn-sm" onClick={goBack}>Back</button>
                  {canAccount ? (
                    <button type="button" className="btn btn-primary btn-sm" disabled={busy || waiting} onClick={startAccount}>{waiting ? "Waiting…" : "Continue in browser"}</button>
                  ) : (
                    <button type="button" className="btn btn-primary btn-sm" onClick={closeAdd}>Close</button>
                  )}
                </div>
              </div>
            ) : null}

            {step === "llama" ? (
              <form className="form-new" noValidate onSubmit={saveLlama}>
                <input type="url" autoComplete="off" placeholder="http://127.0.0.1:8080" value={llamaUrl} onChange={(e) => setLlamaUrl(e.target.value)} />
                <input type="password" autoComplete="off" placeholder="API key (optional)" value={key} onChange={(e) => setKey(e.target.value)} />
                <p className="form-error" hidden={!err}>{err}</p>
                <div className="dlg-actions">
                  <button type="button" className="btn btn-ghost btn-sm" onClick={goBack}>Back</button>
                  <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>Save</button>
                </div>
              </form>
            ) : null}

            {step === "key" ? (
              <form className="form-new" noValidate onSubmit={save}>
                <input type="password" autoComplete="off" placeholder="sk-…" value={key} onChange={(e) => setKey(e.target.value)} />
                <p className="form-error" hidden={!err}>{err}</p>
                <div className="dlg-actions">
                  <button type="button" className="btn btn-ghost btn-sm" onClick={goBack}>Back</button>
                  <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>Save</button>
                </div>
              </form>
            ) : null}
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </PageFrame>
  );
}
