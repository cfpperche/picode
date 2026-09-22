import { useEffect, useMemo, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { Command, defaultFilter } from "cmdk";
import { api } from "@picode/shared/client/api.js";
import { toast, toastError } from "../lib/toast.js";
import { apiKeySchema, llamaLoginSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { go } from "../lib/routes.js";
import { cliProvidersHash, cliProvidersReturnTo } from "@picode/shared/domain/cliProviders.js";
import { providerName } from "@picode/shared/domain/credentials.js";

import { ProviderFace } from "./ProviderFaces.jsx";
import { pushRecent } from "@picode/shared/domain/providerRecents.js";

// The Add-provider dialog, on its own so more than one pane can open it.
// It owns the whole add flow — pick a provider, choose a method, paste a key
// or finish an account login in the browser — against the catalog the parent
// already fetched, and reports back twice: onSaved once a credential landed
// (the parent reloads its roster) and onClose when the dialog is dismissed.
// The sign-in recents it writes are the app-wide store, not parent state.
//
// pi's flow is the model every CLI gets (the owner's call, 2026-09-22): a
// guest CLI passes its roster instead of pi's catalog, and the dialog offers
// exactly the doors that CLI has natively — a key where the roster names the
// variable it is passed in, an account where the roster says how the sign-in
// runs (PiCode's browser flow, or the CLI's own login in a terminal, handed
// back through onTerminalSignin), and Custom provider where the CLI keeps
// definitions of its own.
export default function AddProviderDialog({ open, catalog, onClose, onSaved, cli = "pi", roster = null, onTerminalSignin }) {
  const guest = cli !== "pi";
  // A login in flight is abandoned by closing or going back, never by a
  // late status poll landing on the step the user has already left.
  const oauthAttempt = useRef(0);
  useEffect(() => () => { oauthAttempt.current++; }, []);
  const [step, setStep] = useState("pick"); // pick | method | key | oauth | llama
  const [pick, setPick] = useState(null);
  const [key, setKey] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const [userCode, setUserCode] = useState("");
  const [llamaUrl, setLlamaUrl] = useState("http://127.0.0.1:8080");
  // The picker is filtered and ordered here, not by cmdk: cmdk re-sorts the
  // DOM on every keystroke and keeps its own scroll, which left the list
  // scrolled past the highlighted match after "o…" and in the last search's
  // order after clearing it (the owner hit both, 2026-09-22). Here the
  // untouched list is alphabetical, a search ranks by cmdk's own score, the
  // first row is the selection and the list starts at the top.
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState("");
  const listRef = useRef(null);

  const list = useMemo(
    () => (guest ? guestPicks(roster) : catalog && catalog.providers ? catalog.providers : []),
    [guest, roster, catalog],
  );
  const available = useMemo(
    () => guest
      ? [...list].sort((a, b) => a.label.localeCompare(b.label, undefined, { sensitivity: "base" }))
      : list.filter((p) => !p.signedIn).sort((a, b) => a.id.localeCompare(b.id)),
    [list],
  );
  const q = query.trim();
  const shown = useMemo(() => rankPicks(available, q), [available, q]);
  // The selection moves in the same update as the search: cmdk scrolls its
  // selected row into view after each change, and a selection set a render
  // later let it scroll to the previous row first (measured on the live
  // bundle, 2026-09-22). The frame after, the list starts at the top.
  function search(v) {
    const next = rankPicks(available, v.trim());
    setQuery(v);
    setSelected(next.length ? pickValue(next[0]) : "");
    requestAnimationFrame(() => { if (listRef.current) listRef.current.scrollTop = 0; });
  }
  // Keyed by the rows' ids, never the array: a new array on a re-render
  // (a hover, an arrow key) must not snap the selection back to row 0.
  const rowsKey = available.map((p) => p.id).join("\n");
  useEffect(() => {
    if (step !== "pick") return;
    setSelected(shown.length ? pickValue(shown[0]) : "");
    requestAnimationFrame(() => { if (listRef.current) listRef.current.scrollTop = 0; });
  }, [step, rowsKey]);
  const customDoor = guest ? !!(roster && roster.custom && roster.custom.available) : true;
  const label = (p) => (p ? (guest ? p.label : p.id) : "");

  function reset() {
    setStep("pick");
    setPick(null);
    setKey("");
    setErr("");
    setUserCode("");
    setLlamaUrl("http://127.0.0.1:8080");
    setQuery("");
    setBusy(false);
    setWaiting(false);
  }

  // Every open starts at the picker; every close abandons whatever the
  // dialog was waiting on, including a poll that is still ticking.
  useEffect(() => {
    oauthAttempt.current++;
    if (!open) { setWaiting(false); setBusy(false); return; }
    reset();
  }, [open]);

  function close() {
    oauthAttempt.current++;
    reset();
    if (onClose) onClose();
  }

  // A write landed: the dialog is done with the provider, the parent is not.
  // Close first — the old close-then-refresh order — so a roster that is
  // slow to reload never holds an open modal.
  async function saved() {
    reset();
    if (onClose) onClose();
    if (onSaved) await onSaved();
  }

  function goBack() {
    oauthAttempt.current++;
    setWaiting(false);
    setBusy(false);
    if ((step === "key" || step === "oauth") && pick && pick.login === "both") {
      setStep("method");
      return;
    }
    setPick(null);
    setStep("pick");
  }

  function chooseProvider(p) {
    setPick(p);
    setErr("");
    if (!guest && p.id === "llama.cpp") { setStep("llama"); return; }
    // An unsigned custom definition has a row nowhere: picking it from the
    // picker opens the Edit form so a wrong URL/model can be fixed, instead
    // of a key-only sign-in into a broken endpoint.
    if (p.custom && !p.signedIn) { editCustom(p); return; }
    if (p.login === "oauth") { setStep("oauth"); return; }
    if (p.login === "both") { setStep("method"); return; }
    setStep("key");
  }

  async function save(e) {
    e.preventDefault();
    const parsed = parseForm(apiKeySchema, { key });
    if (!parsed.ok) { setErr(parsed.error); return; }
    setBusy(true);
    setErr("");
    try {
      if (guest) {
        // A guest's key lands in the vault; the CLI reads it through the
        // variable its roster names (omp: at launch, ADR-0178).
        await api("/api/credentials", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ provider: pick.id, label: "", key: parsed.value.key }),
        });
      } else {
        await api("/api/providers/" + encodeURIComponent(pick.id), {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ key: parsed.value.key }),
        });
        pushRecent(pick.id);
      }
      toast.ok(guest ? "Key saved for " + label(pick) + "." : "Signed in to " + pick.id + ".");
      await saved();
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
      pushRecent("llama.cpp");
      toast.ok("Signed in to llama.cpp.");
      await saved();
    } catch (ex) {
      toastError(ex);
    } finally {
      setBusy(false);
    }
  }

  async function startAccount() {
    if (!pick) return;
    const attempt = ++oauthAttempt.current;
    const current = () => attempt === oauthAttempt.current;
    setBusy(true);
    setErr("");
    try {
      const res = guest
        ? await api("/api/credentials/signin", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ cli, provider: pick.id }),
        })
        : await api("/api/oauth/start", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ provider: pick.id, returnTo: cliProvidersReturnTo(location.href) }),
        });
      if (!current()) return;
      // The CLI's own login runs in a terminal: the pane's sign-in strip owns
      // it from here (the door to that terminal and Check now), not a modal.
      if (guest && !res.oauth) {
        close();
        if (onTerminalSignin) onTerminalSignin(res);
        return;
      }
      if (res && res.userCode) setUserCode(res.userCode);
      if (res && res.url) window.open(res.url, "_blank", "noopener");
      setWaiting(true);
      const t0 = Date.now();
      while (Date.now() - t0 < 5 * 60 * 1000) {
        await new Promise((r) => setTimeout(r, 1000));
        if (!current()) return;
        const st = await api("/api/oauth/status");
        if (!current()) return;
        if (st && st.done) {
          setWaiting(false);
          if (st.error) { setErr(st.error); return; }
          toast.ok("Signed in to " + label(pick) + ".");
          if (!guest) pushRecent(pick.id);
          await saved();
          return;
        }
        if (st && !st.pending && !st.done) break;
      }
      setWaiting(false);
    } catch (ex) {
      if (!current()) return;
      setErr(ex.message);
      setWaiting(false);
    } finally {
      if (current()) setBusy(false);
    }
  }

  const canAccount = pick && (guest ? !!pick.signin : ["anthropic", "openai-codex", "github-copilot", "kimi-coding", "xai"].includes(String(pick.id).toLowerCase()));
  const inTerminal = guest && pick && pick.signin === "terminal";
  const cliName = guest ? (roster && roster.cliName) || cli : "pi";
  const title = !pick ? "Add provider" : step === "method" || step === "oauth" || step === "llama" ? label(pick) : "API key · " + label(pick);

  // Custom endpoint (ADR-0129) lives on its own page (CustomEndpointPage):
  // the form outgrew the dialog. These entries close the dialog and navigate;
  // the close lands first, so the page being left is not left holding an open
  // modal.
  function startCustom() {
    close();
    if (guest) { location.hash = cliProvidersHash(cli, { custom: true }); return; }
    go("providers-custom");
  }

  function editCustom(p) {
    close();
    if (guest) { location.hash = cliProvidersHash(cli, { custom: true, customId: p.id }); return; }
    go("providers-custom", "", { customId: p.id });
  }

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) close(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">
            {step === "pick" ? "Pick a provider." : step === "method" ? "Choose how to sign in." : step === "llama" ? "Router URL. API key is optional." : step === "oauth" ? (userCode ? "Enter this code in the browser tab." : inTerminal ? cliName + " signs in to " + label(pick) + " in a terminal of its own; PiCode opens it for you." : canAccount ? "Finish sign-in in the browser tab." : "Account login is not available here. Use an API key.") : "Paste the key. It is not shown again."}
          </Dialog.Description>

          {step === "pick" ? (
            <>
            <Command loop className="prov-pick" shouldFilter={false} value={selected} onValueChange={setSelected}>
              <Command.Input className="combo-input" placeholder="Search providers" value={query} onValueChange={search} />
              <Command.List className="prov-pick-list" ref={listRef}>
                <Command.Empty className="combo-empty">No matches</Command.Empty>
                {shown.map((p) => (
                  <Command.Item key={p.id} value={pickValue(p)} className="cockpit-opt" onSelect={() => chooseProvider(p)}>
                    <ProviderFace id={p.id} name={label(p)} />
                    <span>{label(p)}</span>
                    <span className="combo-hint">{p.id === "llama.cpp" ? "local router" : p.custom ? "custom provider" : p.login === "both" ? "account or api key" : p.login === "oauth" ? "account" : "api key"}</span>
                  </Command.Item>
                ))}
              </Command.List>
            </Command>
            {/* The custom door sits under the list, not in it: a row in the
                list is ranked and highlighted by cmdk, and a door whose
                words ("gateway", "base url") outscored a real match stole
                Enter from Baseten and the AI gateways, then left the list
                scrolled past the match (the owner hit it, 2026-09-22). Here
                it survives any search — typing a name the list does not know
                is exactly when it is needed. */}
            {customDoor ? (
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" title="Any OpenAI-compatible gateway" onClick={startCustom}>+ Custom provider</button>
              </div>
            ) : null}
            </>
          ) : null}

          {step === "method" ? (
            <div className="prov-methods">
              <button type="button" className="btn btn-primary" onClick={() => setStep("key")}>Sign in with an API key</button>
              <button type="button" className="btn btn-ghost" onClick={() => setStep("oauth")}>Sign in with an account</button>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={goBack}>Back</button>
              </div>
            </div>
          ) : null}

          {step === "oauth" ? (
            <div>
              {userCode ? <p className="oauth-code">{userCode}</p> : null}
              <p className="form-error" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={goBack}>Back</button>
                {canAccount ? (
                  <button type="button" className="btn btn-primary btn-sm" disabled={busy || waiting} onClick={startAccount}>{waiting ? "Waiting…" : inTerminal ? (busy ? "Opening…" : "Open sign-in") : "Continue in browser"}</button>
                ) : (
                  <button type="button" className="btn btn-primary btn-sm" onClick={close}>Close</button>
                )}
              </div>
            </div>
          ) : null}

          {step === "llama" ? (
            <form className="form-new" noValidate onSubmit={saveLlama}>
              <input type="url" autoComplete="off" placeholder="http://127.0.0.1:8080" value={llamaUrl} onChange={(e) => setLlamaUrl(e.target.value)} />
              <input type="password" autoComplete="off" placeholder="API key (optional)" value={key} onChange={(e) => setKey(e.target.value)} />
              <p className="form-error" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={goBack}>Back</button>
                <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>Save</button>
              </div>
            </form>
          ) : null}

          {step === "key" ? (
            <form className="form-new" noValidate onSubmit={save}>
              <input type="password" autoComplete="off" placeholder="sk-…" value={key} onChange={(e) => setKey(e.target.value)} />
              <p className="form-error" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={goBack}>Back</button>
                <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>Save</button>
              </div>
            </form>
          ) : null}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

// guestPicks turns a guest CLI's roster into the picker's rows, keeping only
// the doors that CLI has natively: a key where the roster names the variable
// it is passed in, an account where it says how the sign-in runs, and its own
// custom definitions (which open their Edit page). A provider with neither is
// not offered.
function guestPicks(roster) {
  const rows = roster && Array.isArray(roster.providers) ? roster.providers : [];
  const out = [];
  for (const p of rows) {
    const name = providerName(p.id, p.name);
    if (p.custom && p.definition) {
      out.push({ id: p.id, label: name, login: "key", custom: true, signedIn: false });
      continue;
    }
    const kinds = p.kinds || [];
    const key = kinds.includes("api_key") && !!(p.env && p.env.api_key);
    const account = kinds.includes("oauth") && !!p.signin;
    if (!key && !account) continue;
    out.push({ id: p.id, label: name, login: key && account ? "both" : key ? "key" : "oauth", signin: p.signin || "" });
  }
  return out;
}

// rankPicks is the picker's order: alphabetical untouched, cmdk's own score
// under a search.
function rankPicks(rows, q) {
  if (!q) return rows;
  return rows
    .map((p) => ({ p, score: defaultFilter(pickValue(p), q) }))
    .filter((r) => r.score > 0)
    .sort((a, b) => b.score - a.score)
    .map((r) => r.p);
}

// A picker row's search text: its id, what it reads as, and its login kind.
function pickValue(p) {
  return (p.id + " " + (p.label || "") + " " + (p.login || "")).toLowerCase();
}
