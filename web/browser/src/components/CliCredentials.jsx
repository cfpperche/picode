import { useCallback, useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import PageFrame from "./PageFrame.jsx";
import AddProviderDialog from "./AddProviderDialog.jsx";
import ClaudeCodeLoginDialog from "./ClaudeCodeLoginDialog.jsx";
import CodexLoginDialog from "./CodexLoginDialog.jsx";
import GrokLoginDialog from "./GrokLoginDialog.jsx";
import MuseLoginDialog from "./MuseLoginDialog.jsx";
import AgyLoginDialog from "./AgyLoginDialog.jsx";
import OpenCodeLoginDialog from "./OpenCodeLoginDialog.jsx";
import CustomEndpointPage from "./CustomEndpointPage.jsx";
import QuotaStrip from "./QuotaStrip.jsx";
import TermSurface from "./TermSurface.jsx";
import { ProviderFace } from "./ProviderFaces.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { closeTerm } from "../lib/terms.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { askPrompt } from "../lib/prompt.js";
import { apiKeySchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { cliPaneHash } from "@picode/shared/domain/cliLaunch.js";
import { cliProvidersHash, supportsCustomProviders } from "@picode/shared/domain/cliProviders.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { usagePath } from "@picode/shared/domain/providerUsage.js";
import { formatSpend, identityLine, spendByProvider } from "@picode/shared/domain/providerRows.js";
import { vaultProblemText } from "@picode/shared/domain/credentials.js";
import { healthChip, kindLabel, orderAccounts, providerName, sourceLabel, VERIFY_LABEL } from "@picode/shared/domain/credentials.js";

// The providers surface for all nine agent CLIs (ADR-0169). One roster
// (GET /api/credentials?cli=), one grid — providers.css owns the columns and
// the 840px card fold, this file owns the state — and one vocabulary for pi
// and for the eight guests. It carries pi's two readings (Usage, 7d spend) to
// every CLI and renders only the doors the roster's flags allow: the bar's
// Add, the per-provider Add, Import, Sign in, Verify (pi's own `auth check` at
// the provider, a guest's listing probe on the row), Check usage, Rename,
// Pause/Resume, Sign out, Edit provider. A number we did not fetch is never
// drawn as one (ADR-0031): a row with no report says so, and the one action
// that would get it sits next to the word. Both apps carry this file
// (ADR-0072), so the only difference between them is the modal primitive on
// line 2.

const credPath = (provider, id, suffix = "") =>
  "/api/credentials/" + encodeURIComponent(provider) + "/" + encodeURIComponent(id) + suffix;

const json = (method, body) => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

export default function CliCredentials({ hidden, cli, add = false, custom = "", customId = "", onCatalogChange }) {
  const [data, setData] = useState(null);
  const [problem, setProblem] = useState(null);
  const [loading, setLoading] = useState(true);
  // Our own 7-day spend by provider, from the session files: one call for the
  // whole pane, never a vendor (ADR-0031). Empty means "we do not know", and
  // the column says so.
  const [spend, setSpend] = useState(() => new Map());
  const [busy, setBusy] = useState("");
  const [verdicts, setVerdicts] = useState({});
  const [nativeError, setNativeError] = useState({});
  // A sign-in in flight: the CLI's own login runs in a PiCode terminal, and
  // this holds the hint plus whatever the last check answered.
  const [signin, setSignin] = useState(null);
  // The sign-in terminal shown in the card's dialog (ADR-0184).
  const [signinView, setSigninTerm] = useState(null);
  // pi's provider + model catalog (ADR-0129): the Add-provider dialog reads it,
  // and so does the app's model picker through onCatalogChange. No other CLI
  // has one to fetch.
  const [catalog, setCatalog] = useState(null);
  const [catalogError, setCatalogError] = useState("");
  const [addProviderOpen, setAddProviderOpen] = useState(false);
  const [claudeOpen, setClaudeOpen] = useState(false);
  // The platform being edited, when the Claude Code dialog opens on it.
  const [claudeEdit, setClaudeEdit] = useState(null);
  const [codexOpen, setCodexOpen] = useState(false);
  const [codexEdit, setCodexEdit] = useState(null);
  const [grokOpen, setGrokOpen] = useState(false);
  const [museOpen, setMuseOpen] = useState(false);
  const [agyOpen, setAgyOpen] = useState(false);
  const [opencodeOpen, setOpencodeOpen] = useState(false);
  const [addOpen, setAddOpen] = useState(false);
  const [form, setForm] = useState({ provider: "", key: "" });
  const [formError, setFormError] = useState("");
  const [saving, setSaving] = useState(false);
  const live = useRef(false);
  const sequence = useRef(0);
  const catalogSeq = useRef(0);
  const notify = useRef(onCatalogChange);
  notify.current = onCatalogChange;

  // The vault is the daemon's file, not a vendor's: a load is one local
  // request, and a refresh keeps the last good roster on screen.
  const load = useCallback(async () => {
    const request = ++sequence.current;
    setLoading(true);
    // Fired with the roster, once per pane load: a local aggregate, so the
    // column costs the pane nothing and a failure just leaves its dash.
    api("/api/sessions/stats?range=7d")
      .then((stats) => { if (live.current) setSpend(spendByProvider(stats)); })
      .catch(() => {});
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

  // fresh: "Try again" asks Pi again; every other read — mount, focus, after
  // a save — is answered from the server's kept list, which a sign-in or a
  // custom-provider save already updates.
  const loadCatalog = useCallback(async (fresh = false) => {
    if (cli !== "pi") return;
    const request = ++catalogSeq.current;
    try {
      const next = await api("/api/catalog" + (fresh === true ? "?fresh=1" : ""));
      if (!Array.isArray(next?.providers)) throw new Error("Invalid provider catalog.");
      if (!live.current || request !== catalogSeq.current) return;
      setCatalog(next);
      setCatalogError("");
      notify.current?.(next);
    } catch {
      if (live.current && request === catalogSeq.current) setCatalogError("Couldn’t load the provider catalog.");
    }
  }, [cli]);

  const refresh = useCallback(() => { load(); loadCatalog(); }, [load, loadCatalog]);

  useEffect(() => {
    live.current = true;
    refresh();
    window.addEventListener("focus", refresh);
    return () => {
      live.current = false;
      sequence.current++;
      catalogSeq.current++;
      window.removeEventListener("focus", refresh);
    };
  }, [refresh]);

  // The pane survives a CLI switch without remounting; a sign-in in flight
  // belongs to the CLI it was started for, so it goes with it.
  useEffect(() => {
    setSignin(null);
    setSigninTerm(null);
    // A sign-in still running for this CLI comes back to its strip: the
    // card is the only way to it (ADR-0184). The server kept the stamp of
    // the account the store held when it started, so Check now still tells
    // a new login from the old one.
    let live = true;
    api("/api/credentials/signin?cli=" + encodeURIComponent(cli)).then((res) => {
      if (live && res && res.terminalId) setSignin((prev) => prev || { hint: res.hint || "", error: "", terminalId: res.terminalId, stamp: res.stamp || "" });
    }).catch(() => {});
    return () => { live = false; };
  }, [cli]);
  // The server owns the sign-in terminal (ADR-0184): it closes it once the
  // credential lands, when its process ends, or after fifteen minutes. The
  // strip says so instead of pointing at a terminal that is gone.
  const signinTerm = signin ? signin.terminalId : "";
  useEffect(() => {
    if (!signinTerm) return undefined;
    return subscribeFeed((e) => {
      if (e.type === "terminal.deleted" && e.data && e.data.id === signinTerm) {
        setSignin((prev) => (prev && prev.terminalId === signinTerm ? { ...prev, terminalId: "", closed: true } : prev));
        setSigninTerm(null);
        closeTerm("sh:" + signinTerm);
      }
    });
  }, [signinTerm]);

  // The sign-in terminal opens here, in a dialog of the card that owns it —
  // it is not in the sidebar or any terminal list (ADR-0184).
  async function openSigninTerm() {
    const id = signin && signin.terminalId;
    if (!id) return;
    try {
      setSigninTerm(await api("/api/terminals/" + encodeURIComponent(id) + "/open", { method: "POST" }));
    } catch (ex) {
      toastError(ex);
    }
  }

  // Cancel ends the sign-in terminal with the strip: nothing keeps running
  // where nobody sees it.
  async function cancelSignin() {
    const id = signin && signin.terminalId;
    setSignin(null);
    setSigninTerm(null);
    if (id) closeTerm("sh:" + id);
    if (id) await api("/api/terminals/" + encodeURIComponent(id), { method: "DELETE" }).catch(() => {});
  }

  const providers = data && Array.isArray(data.providers) ? data.providers : [];
  const cliName = (data && data.cliName) || terminalCliLabel(cli);
  const unreadable = !!(data && data.vault && data.vault.readable === false);
  // The server knows no credential mechanism for this CLI: the pane is one
  // line, and any roster a previous load left on screen goes with it.
  const missing = !!(problem && problem.missing);
  // What the bar's primary opens, straight from the roster: pi adds a provider
  // through its own login, a guest adds a key to a provider it already reads.
  const addSpec = data && data.add ? data.add : null;
  // "provider" is pi's Add flow, "claude-code" Claude Code's own sign-in
  // dialog (ADR-0187), anything else the key form.
  const addKind = addSpec ? (["provider", "claude-code", "codex", "grok", "muse", "agy", "opencode"].includes(addSpec.kind) ? addSpec.kind : "key") : "";
  const addLabel = (addSpec && addSpec.label) || (addKind === "provider" ? "Add provider" : "Add API key");
  // pi's models.json page: the one provider surface a guest CLI has no
  // equivalent for, linked rather than hidden behind a menu item (ADR-0169).
  const customDoor = data && data.custom && data.custom.available ? (data.custom.href || cliProvidersHash("pi", { custom: true })) : "";
  const total = providers.reduce((n, p) => n + ((p.accounts || []).length), 0);
  const canAddKey = addKind === "key" && providers.length > 0;
  // The bar's primary exists wherever the CLI can add anything: pi opens its
  // catalog dialog, a guest opens the key dialog whose select lists the CLI's
  // own providers. With that button up in the bar, a per-provider Add would
  // say the same thing one row below it — so it only earns its place when the
  // bar cannot offer a primary at all (no providers, or nothing to add).
  const barPrimary = !!addSpec && providers.length > 0;
  const perProviderAdd = canAddKey && !barPrimary ? addLabel : "";
  // The bar outlives an empty roster only where Add can work on its own: pi's
  // dialog also carries the custom-provider door, so it needs no provider to
  // already exist.
  const bar = providers.length > 0 || addKind === "provider";
  // The table lists what the user has, not what the CLI could reach: a
  // provider with no account, no custom definition and no login waiting to be
  // imported stays in the Add dialog's select and out of the rows (pi's
  // catalog alone is ~25 providers, each an "No accounts yet" line).
  const shown = providers.filter((p) => (p.accounts || []).length > 0
    || (p.custom && p.definition)
    || (p.native && !p.native.imported)
    || nativeError[p.id]);
  // The legend names columns: with no rows under it there is nothing to name.
  const anyRows = shown.some((p) => (p.accounts || []).length > 0);
  // The selected provider can also be signed in the CLI's own way: where the
  // roster carries an oauth kind and the CLI has a sign-in at all, the dialog
  // offers the guided flow — the vendor's OAuth happens in the CLI (ADR-0168),
  // never here.
  const picked = providers.find((p) => p.id === form.provider);
  const pickedName = providerName(form.provider, picked?.name);
  const guidedPick = !!addSpec && !!data && data.signin && data.signin.available
    && (picked?.kinds || []).includes("oauth");
  // A provider that takes no key (a subscription-only one, like most of Omp's
  // own catalog) offers no key field and no Save: a key saved for it would be
  // a vault row nothing ever reads.
  const keyless = !!picked && !(picked.kinds || []).includes("api_key");
  // The Add select can hold Omp's whole /login roster (~80): alphabetical is
  // the only order a person can scan.
  const pickList = [...providers].sort((a, b) =>
    providerName(a.id, a.name).localeCompare(providerName(b.id, b.name), undefined, { sensitivity: "base" }));

  function openPrimary() {
    if (addKind === "provider") setAddProviderOpen(true);
    else if (addKind === "claude-code") setClaudeOpen(true);
    else if (addKind === "codex") setCodexOpen(true);
    else if (addKind === "grok") setGrokOpen(true);
    else if (addKind === "muse") setMuseOpen(true);
    else if (addKind === "agy") setAgyOpen(true);
    else if (addKind === "opencode") setOpencodeOpen(true);
    else openAdd("");
  }

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

  function closeAddProvider() {
    // The route is what opened this door, so closing it clears the route. The
    // dialog's own custom-provider item closes first and then moves, so the
    // move writes the last hash and wins.
    setAddProviderOpen(false);
    if (add && typeof location !== "undefined") location.hash = cliPaneHash(cli, "providers");
  }

  useEffect(() => {
    if (!add || !data || unreadable) return;
    // The route asked for the door the bar's primary opens; the roster is what
    // says which that is, and pi's catalog is what fills the provider list.
    if (addKind === "provider") { if (cli !== "pi" || catalog) setAddProviderOpen(true); return; }
    if (addKind === "claude-code") { setClaudeOpen(true); return; }
    if (addKind === "codex") { setCodexOpen(true); return; }
    if (addKind === "grok") { setGrokOpen(true); return; }
    if (addKind === "muse") { setMuseOpen(true); return; }
    if (addKind === "agy") { setAgyOpen(true); return; }
    if (addKind === "opencode") { setOpencodeOpen(true); return; }
    if (addKind === "key") openAdd("");
    // The route asked for the dialog; the roster is what fills its provider list.
  }, [add, !!data, unreadable, addKind, !!catalog]);

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
    if (keyless) return;
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

  async function startSignin() {
    setBusy("signin");
    try {
      const started = await api("/api/credentials/signin", json("POST", { cli }));
      // The terminal the server opened is the whole point of the flow: keep
      // its id so the strip can lead to it (2026-09-21). priorId is the account
      // already in the file: Check now must not toast success while that same
      // account is still what the file holds.
      const launchError = started.terminal && started.terminal.launchError;
      setSignin({
        hint: started.hint || "",
        error: launchError ? String(launchError) : "",
        terminalId: started.terminalId || "",
        stamp: started.stamp || "",
      });
    } catch (ex) {
      toastError(ex);
    } finally {
      setBusy("");
    }
  }

  // The CLI's own login writes its own store; this is the one click that
  // notices and files it. Nothing here performs the vendor's OAuth.
  async function checkSignin() {
    setBusy("check");
    try {
      // Ask BEFORE the write: preview says whether this login would replace
      // an unnamed one, and after the write the replaced tokens are already
      // gone — the name has to come first.
      const pre = await api("/api/credentials/import", json("POST", { cli, preview: true }));
      // The file still holds the account that was there when Sign in was
      // clicked. Closing the strip here is a lie: no second account arrived.
      if (signin && signin.stamp && pre.stamp === signin.stamp) {
        setSignin((prev) => ({
          ...(prev || {}),
          error: "That's still the account already saved. Sign into the other one in the terminal, then check again.",
        }));
        return;
      }
      // A store that carries no account name would have this sign-in replace
      // the row already there. That write is the person's choice, made here:
      // name the new login, or keep the old one beside it.
      if (!pre.created && !pre.identity) {
        setSignin((prev) => ({ ...(prev || {}), replacing: true }));
        return;
      }
      const row = await api("/api/credentials/import", json("POST", { cli }));
      setSignin(null);
      toast.ok(row.who ? "Signed in as " + row.who + "." : "Saved to the vault.");
      await load();
    } catch (ex) {
      setSignin((prev) => ({
        ...(prev || {}),
        error: ex.status === 404 ? "Nothing saved yet — finish the sign-in, then check again." : ex.message,
      }));
    } finally {
      setBusy("");
    }
  }

  async function finishSignin(extra) {
    setBusy("check");
    try {
      const row = await api("/api/credentials/import", json("POST", { cli, ...extra }));
      setSignin(null);
      if (extra.as) toast.ok("Saved as " + extra.as + ".");
      else if (extra.keep) toast.ok("Both logins saved.");
      else toast.ok(row.who ? "Signed in as " + row.who + "." : "Saved to the vault.");
      await load();
    } catch (ex) {
      toastError(ex);
    } finally {
      setBusy("");
    }
  }

  async function nameAndAdd() {
    setBusy("name");
    try {
      const pre = await api("/api/credentials/import", json("POST", { cli, preview: true }));
      const name = await askPrompt({
        title: "Name this login",
        message: "This CLI's store carries no account name, so its logins cannot be told apart. Name this one and the next sign-in is kept beside it instead of on top of it.",
        defaultValue: pre.who || "",
        confirmLabel: "Name it",
      });
      setBusy("");
      if (!name) return;
      await finishSignin({ as: name });
    } catch (ex) {
      setBusy("");
      setSignin((prev) => ({ ...(prev || {}), error: ex.message }));
    }
  }

  async function rename(provider, account) {
    const name = await askPrompt({
      title: "Rename " + (account.label || providerName(provider.id, provider.name)),
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

  // pi's own check: pi is what runs the agent, so pi's answer is the one that
  // matters, and for a built-in it costs nothing. A custom endpoint is the
  // exception — pi only reports that a credential is present, so its row is
  // verified with one minimal real request (the action names the cost).
  async function verifyProvider(provider) {
    setBusy("verify:" + provider.id);
    try {
      // cli picks the checker: pi's own answer, or — for an omp custom
      // endpoint — the saved models.yml key spent on one minimal request.
      const res = await api("/api/providers/" + encodeURIComponent(provider.id) + "/verify", json("POST", { cli }));
      setVerdicts((prev) => ({ ...prev, [provider.id]: res }));
      if (res && res.ok) toast.ok(res.label ? provider.id + ": " + res.label + "." : cliName + " can use " + providerName(provider.id, provider.name) + ".");
      else toast.error(provider.id + ": " + ((res && (res.reason || res.status)) || "not ready"));
    } catch (ex) {
      toastError(ex);
    } finally {
      setBusy("");
    }
  }

  // A CLI on a cloud platform (Claude Code: ADR-0189; Codex: ADR-0191):
  // stopping removes it from that CLI's own files, secret included.
  async function stopPlatform(platform) {
    const codex = cli === "codex";
    const ok = await askConfirm({
      title: "Stop using " + platform.name,
      message: codex
        ? "Signs Codex out of " + platform.name + " and removes the key it saved. Sign in again with ChatGPT or an API key to keep using Codex."
        : "Removes " + platform.name + " from Claude Code's settings, with any key saved there. New terminals go back to your Claude subscription or Console key.",
      confirmLabel: "Stop using",
      danger: true,
    });
    if (!ok) return;
    await run("platform", () => api(codex ? "/api/codex/platform" : "/api/claude-code/platform", { method: "DELETE" }),
      () => toast.ok(cliName + " no longer uses " + platform.name + "."));
  }

  // A custom provider's definition and its key are one row in the CLI's own
  // file: removing it is the only way to un-key a gateway, so the confirm
  // names both halves.
  async function removeProvider(provider) {
    const name = providerName(provider.id, provider.name);
    const ok = await askConfirm({
      title: "Remove " + name,
      message: "Removes the definition and its saved key from " + cliName + "’s models.yml. The gateway itself is not touched.",
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    await run("remove:" + provider.id,
      () => api("/api/providers/custom/" + encodeURIComponent(provider.id) + "?cli=" + encodeURIComponent(cli), { method: "DELETE" }),
      () => toast.ok(name + " removed."));
  }

  // Guided sign-in from the Add provider dialog: the server either starts
  // the CLI's own browser flow — the authorize page opens in a tab, PiCode
  // polls until the callback lands, the vault row appears — or runs the
  // terminal strip, per the CLI and the provider that was picked.
  async function guidedSignin() {
    const provider = form.provider;
    setBusy("signin");
    try {
      const res = await api("/api/credentials/signin", json("POST", { cli, provider }));
      if (res.oauth) {
        closeAdd();
        if (res.url) window.open(res.url, "_blank", "noopener");
        const t0 = Date.now();
        while (Date.now() - t0 < 5 * 60 * 1000) {
          await new Promise(r => setTimeout(r, 1000));
          const st = await api("/api/oauth/status");
          if (st && st.done) {
            setBusy("");
            if (st.error) { toastError(new Error(st.error)); return; }
            toast.ok("Signed in to " + providerName(provider) + ".");
            await load();
            return;
          }
        }
        setBusy("");
        return;
      }
      closeAdd();
      const launchError = res.terminal && res.terminal.launchError;
      setSignin({ hint: res.hint || "", error: launchError ? String(launchError) : "", terminalId: res.terminalId || "", stamp: res.stamp || "" });
      setBusy("");
    } catch (ex) {
      setBusy("");
      toastError(ex);
    }
  }

  // One vendor call, on this click, under ADR-0129's rule: the roster comes
  // back carrying the report (and whatever identity the vendor volunteered).
  async function checkUsage(provider, account) {
    await run("check:" + account.id, () => api(usagePath(provider.id, account.id)));
  }

  async function pause(provider, account) {
    const paused = !account.paused;
    await run("pause:" + account.id, () => api(credPath(provider.id, account.id, "/pause"), json("POST", { paused })),
      () => toast.ok(paused ? "Paused. The credential is kept." : "Resumed."));
  }

  // Use writes the chosen account into the CLI's own login file (ADR-0166):
  // the file the CLI already reads, never a new home and never an env var.
  // The confirm says what is replaced; the first write keeps a copy of it.
  async function use(provider, account) {
    const label = account.label || providerName(provider.id, provider.name);
    const ok = await askConfirm({
      title: "Use " + label + " in " + cliName + "?",
      // A key is not written into a file for these two: it rides new
      // launches, and each CLI's own login steps aside (ADR-0187, ADR-0192).
      message: account.type === "api_key" && cli === "grok"
        ? "New Grok terminals use this key. Grok's own sign-in is kept in the vault and Grok is signed out — Use on it brings it back."
        : account.type === "api_key" && cli === "claude-code"
          ? "New Claude Code terminals use this key instead of your subscription."
          : cliName + "’s own login file is replaced. The first time, PiCode keeps a copy of what is in it now.",
      confirmLabel: "Use",
    });
    if (!ok) return;
    await run("use:" + account.id, async () => {
      const res = await api(credPath(provider.id, account.id, "/activate"), json("POST", { cli }));
      return res;
    }, (res) => toast.ok(res && res.backup ? "Now using " + label + ". Original kept at " + res.backup : "Now using " + label + "."));
  }

  async function signOut(provider, account) {
    const identity = identityLine(account, account.usage);
    const ok = await askConfirm({
      title: "Sign out " + (account.label || providerName(provider.id, provider.name)),
      message: "Removes this login from the vault." + (identity ? " " + identity + "." : ""),
      confirmLabel: "Sign out",
      danger: true,
    });
    if (!ok) return;
    await run("remove:" + account.id, () => api(credPath(provider.id, account.id), { method: "DELETE" }),
      () => toast.ok("Signed out."));
  }

  // A custom provider is the one row whose definition lives somewhere else
  // (pi's models.json, omp's models.yml): the edit is that page, on this
  // CLI's own pane.
  function editProvider(provider) {
    if (typeof location !== "undefined") location.hash = cliProvidersHash(cli, { custom: true, customId: provider.id });
  }

  // The custom-endpoint page (ADR-0129) is the pane's sub-page: it renders
  // its own head and needs the provider list its CLI reads definitions from —
  // pi's catalog, omp's roster — so it is rendered here rather than one level
  // up.
  if (custom && supportsCustomProviders(cli)) {
    if (cli === "pi") {
      if (catalog) return <CustomEndpointPage key={custom + (customId || "")} cli={cli} catalog={catalog} onRefresh={loadCatalog} editId={custom === "edit" ? customId : ""} />;
    } else {
      if (data) return <CustomEndpointPage key={custom + (customId || "")} cli={cli} rosterProviders={data.providers} onRefresh={load} editId={custom === "edit" ? customId : ""} />;
    }
    return (
      <section id="cli-credentials-view" hidden={hidden} aria-label="Custom provider">
        {catalogError && cli === "pi" ? (
          <div className="cli-notice is-error" role="alert">
            <span>{catalogError}</span>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => loadCatalog(true)}>Try again</button>
          </div>
        ) : null}
        <div className="cred-loading" aria-label="Loading the provider catalog" role="status">
          <div className="cred-skel-head" />
          <div className="cred-skel-row" />
        </div>
      </section>
    );
  }

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
          <button type="button" className="btn btn-ghost btn-sm" disabled={loading} onClick={refresh}>
            {loading ? "Retrying…" : "Try again"}
          </button>
        </div>
      ) : null}
      {/* pi's roster is built from the catalog, so a catalog that did not load
          is a door that would open empty — say so instead of pretending. */}
      {catalogError && !missing ? (
        <div className="cli-notice is-error" role="alert">
          <span>{catalogError}</span>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => loadCatalog(true)}>Try again</button>
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
        {bar ? (
          <div className="set-row cred-bar" data-align-row>
            {total ? <span className="cred-count">{total === 1 ? "1 account" : total + " accounts"}</span> : null}
            {/* The bar's controls are one group at the end of the row, never a
                set of items each claiming the free space. Add lives in the
                provider group as soon as there is more than one provider, so
                the bar never shows a second one right above it. */}
            <span className="cred-bar-actions">
              {data.signin && data.signin.available ? (
                <button type="button" className="btn btn-ghost btn-sm" disabled={busy === "signin"} onClick={startSignin}>
                  {busy === "signin" ? "Opening…" : "Sign in"}
                </button>
              ) : null}
              {customDoor ? <a className="btn btn-ghost btn-sm cred-bar-custom" href={customDoor}>Custom provider</a> : null}
              {barPrimary ? (
                <button type="button" className="btn btn-primary btn-sm" onClick={openPrimary}>{addLabel}</button>
              ) : null}
            </span>
          </div>
        ) : null}

        {/* The sign-in is in flight in a terminal of its own: the hint, the
            door to that terminal (the owner asked "where do I type /login?"
            on 2026-09-21 — a strip that says "in the terminal" and offers no
            way there is a dead end), and the one action that files the
            result. */}
        {signin ? (
          <div className="cli-notice cred-signin" role="status">
            <span className="cred-signin-hint">{signin.closed ? "The sign-in terminal closed. Check now if you finished, or sign in again." : signin.hint || "Finish the sign-in, then check again."}</span>
            {signin.error ? <span className="cred-signin-error" role="alert">{signin.error}</span> : null}
            <span className="cred-signin-actions">
              {signin.terminalId ? (
                <button type="button" className="btn btn-ghost btn-sm" onClick={openSigninTerm}>Open terminal</button>
              ) : signin.closed ? (
                <button type="button" className="btn btn-ghost btn-sm" disabled={busy === "signin"} onClick={startSignin}>Sign in again</button>
              ) : null}
              {signin.replacing ? (
                <>
                  <button type="button" className="btn btn-primary btn-sm" disabled={busy === "name"} onClick={nameAndAdd}>
                    {busy === "name" ? "Naming…" : "Name and add"}
                  </button>
                  <button type="button" className="btn btn-ghost btn-sm" disabled={busy === "keep"} onClick={() => finishSignin({ keep: true })}>
                    Keep both
                  </button>
                </>
              ) : (
                <button type="button" className="btn btn-ghost btn-sm" disabled={busy === "check"} onClick={checkSignin}>
                  {busy === "check" ? "Checking…" : "Check now"}
                </button>
              )}
              <button type="button" className="btn btn-ghost btn-sm" onClick={cancelSignin}>Cancel</button>
            </span>
          </div>
        ) : null}

        {data.platform ? (
          <div className="cred-native">
            <span className="cred-native-text">
              {cliName + " uses " + data.platform.name
                + (data.platform.region ? " · " + data.platform.region : "")
                + (data.platform.project ? " · " + data.platform.project : "")
                + (data.platform.resource ? " · " + data.platform.resource : "")
                + (data.platform.baseUrl ? " · " + gatewayHost(data.platform.baseUrl) : "")
                + (shown.length ? ", instead of the accounts below." : ".")}
            </span>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => {
              if (cli === "codex") { setCodexEdit(data.platform); setCodexOpen(true); } else { setClaudeEdit(data.platform); setClaudeOpen(true); }
            }}>Edit</button>
            <button type="button" className="btn btn-ghost btn-sm" disabled={busy === "platform"} onClick={() => stopPlatform(data.platform)}>
              {busy === "platform" ? "Stopping…" : "Stop using"}
            </button>
          </div>
        ) : null}

        {shown.length ? (
          <div className="prov-table">
            {/* The column labels are the roster's legend: two readings are only
                comparable if the eye can name the columns they sit under. */}
            {anyRows ? (
              <div className="prov-th" aria-hidden="true">
                <span>Provider</span>
                <span>Account</span>
                <span>Identity</span>
                <span>Usage</span>
                <span className="prov-num">7d spend</span>
                <span />
              </div>
            ) : null}
            {shown.map((p) => {
              const name = providerName(p.id, p.name);
              const note = String(p.note || "").trim();
              const rows = orderAccounts(p.accounts);
              const rowSpend = spend.get(p.id);
              const verdict = verdicts[p.id];
              const money = formatSpend(rowSpend);
              return (
                <div key={p.id} className="cred-provider">
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
                  {/* Its subscription login has no account name to tell two
                      apart, so the vault keeps one per provider here. */}
                  {p.singleOAuth && rows.some((a) => a.type === "oauth" && !a.paused) ? (
                    <p className="cred-note">{"This login carries no account name, so " + name + " keeps one here — a second sign-in can be kept by naming it when asked."}</p>
                  ) : null}

                  {rows.length ? (
                    <ul className="prov-rows">
                      {rows.map((a, i) => (
                        <AccountRow
                          key={a.id}
                          provider={p}
                          account={a}
                          first={i === 0}
                          cliName={cliName}
                          money={money}
                          verdict={verdict}
                          busy={busy}
                          addLabel={perProviderAdd}
                          onUse={() => use(p, a)}
                          onRename={() => rename(p, a)}
                          onVerify={() => verify(p, a)}
                          onVerifyProvider={() => verifyProvider(p)}
                          onCheck={() => checkUsage(p, a)}
                          onPause={() => pause(p, a)}
                          onSignOut={() => signOut(p, a)}
                          onEdit={() => editProvider(p)}
                          onAdd={() => openAdd(p.id)}
                        />
                      ))}
                    </ul>
                  ) : p.custom && p.definition ? (
                    // A definition row has no vault account to list: the
                    // definition and its key live in the CLI's own file, and
                    // the line says which state that file is in.
                    <div className="cred-native">
                      <span className="cred-native-text">
                        {name + " · " + (p.definition.keyed ? "Saved with a key in models.yml." : "Saved in models.yml — no key yet.")}
                      </span>
                      <button type="button" className="btn btn-ghost btn-sm" onClick={() => editProvider(p)}>Edit provider</button>
                      <button
                        type="button"
                        className="btn btn-ghost btn-sm"
                        disabled={busy === "verify:" + p.id}
                        onClick={() => verifyProvider(p)}
                      >
                        {busy === "verify:" + p.id ? "Checking…" : VERIFY_LABEL}
                      </button>
                      <button
                        type="button"
                        className="btn btn-ghost btn-sm"
                        disabled={busy === "remove:" + p.id}
                        onClick={() => removeProvider(p)}
                      >
                        {busy === "remove:" + p.id ? "Removing…" : "Remove provider"}
                      </button>
                    </div>
                  ) : (
                    <p className="cred-empty">
                      <span>{name + " · No accounts yet."}</span>
                      {perProviderAdd ? (
                        <button type="button" className="btn btn-ghost btn-sm" onClick={() => openAdd(p.id)}>{perProviderAdd}</button>
                      ) : null}
                    </p>
                  )}
                </div>
              );
            })}
          </div>
        ) : data.platform ? null : (
          <p className="cred-empty"><span>{"No provider credentials for " + cliName + " yet."}</span></p>
        )}
      </> : null}

      {/* pi's Add flow, for pi (its catalog, ADR-0169) and for the guests whose
          roster asks for it (omp): search, method, key or account, custom. */}
      <OpenCodeLoginDialog
        open={opencodeOpen}
        onClose={() => { setOpencodeOpen(false); if (add && typeof location !== "undefined") location.hash = cliPaneHash(cli, "providers"); }}
        onSaved={load}
        onTerminalSignin={(res) => {
          const launchError = res && res.terminal && res.terminal.launchError;
          setSignin({ hint: (res && res.hint) || "", error: launchError ? String(launchError) : "", terminalId: (res && res.terminalId) || "", stamp: (res && res.stamp) || "" });
        }}
      />

      <AgyLoginDialog
        open={agyOpen}
        onClose={() => { setAgyOpen(false); if (add && typeof location !== "undefined") location.hash = cliPaneHash(cli, "providers"); }}
        onSaved={load}
      />

      <MuseLoginDialog
        open={museOpen}
        onClose={() => { setMuseOpen(false); if (add && typeof location !== "undefined") location.hash = cliPaneHash(cli, "providers"); }}
        onSaved={load}
        onTerminalSignin={(res) => {
          const launchError = res && res.terminal && res.terminal.launchError;
          setSignin({ hint: (res && res.hint) || "", error: launchError ? String(launchError) : "", terminalId: (res && res.terminalId) || "", stamp: (res && res.stamp) || "" });
        }}
      />

      <GrokLoginDialog
        open={grokOpen}
        onClose={() => { setGrokOpen(false); if (add && typeof location !== "undefined") location.hash = cliPaneHash(cli, "providers"); }}
        onSaved={load}
        onTerminalSignin={(res) => {
          const launchError = res && res.terminal && res.terminal.launchError;
          setSignin({ hint: (res && res.hint) || "", error: launchError ? String(launchError) : "", terminalId: (res && res.terminalId) || "", stamp: (res && res.stamp) || "" });
        }}
      />

      <CodexLoginDialog
        open={codexOpen}
        editPlatform={codexEdit}
        onClose={() => { setCodexOpen(false); setCodexEdit(null); if (add && typeof location !== "undefined") location.hash = cliPaneHash(cli, "providers"); }}
        onSaved={load}
        onTerminalSignin={(res) => {
          const launchError = res && res.terminal && res.terminal.launchError;
          setSignin({ hint: (res && res.hint) || "", error: launchError ? String(launchError) : "", terminalId: (res && res.terminalId) || "", stamp: (res && res.stamp) || "" });
        }}
      />

      <ClaudeCodeLoginDialog
        open={claudeOpen}
        editPlatform={claudeEdit}
        onClose={() => { setClaudeOpen(false); setClaudeEdit(null); if (add && typeof location !== "undefined") location.hash = cliPaneHash(cli, "providers"); }}
        onSaved={load}
        onTerminalSignin={(res) => {
          const launchError = res && res.terminal && res.terminal.launchError;
          setSignin({ hint: (res && res.hint) || "", error: launchError ? String(launchError) : "", terminalId: (res && res.terminalId) || "", stamp: (res && res.stamp) || "" });
        }}
      />

      <AddProviderDialog
        open={addProviderOpen}
        catalog={catalog}
        cli={cli}
        roster={cli === "pi" ? null : data}
        onClose={closeAddProvider}
        onSaved={async () => { await load(); await loadCatalog(); }}
        onTerminalSignin={(res) => {
          const launchError = res && res.terminal && res.terminal.launchError;
          setSignin({ hint: (res && res.hint) || "", error: launchError ? String(launchError) : "", terminalId: (res && res.terminalId) || "", stamp: (res && res.stamp) || "" });
        }}
      />

      <Dialog.Root open={!!signinView} onOpenChange={(open) => { if (!open) setSigninTerm(null); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content
            className="dlg dlg-signin-term"
            // A vendor login answers Esc and clicks of its own: they belong
            // to the terminal, and only Close or Cancel leave the dialog.
            onEscapeKeyDown={(e) => e.preventDefault()}
            onPointerDownOutside={(e) => e.preventDefault()}
            onInteractOutside={(e) => e.preventDefault()}
          >
            <Dialog.Title className="dlg-title">{cliName} sign-in</Dialog.Title>
            <Dialog.Description className="dlg-body">{(signin && signin.hint) || "Finish the sign-in, then check again."} Closing this keeps the sign-in running; Cancel ends it.</Dialog.Description>
            <div className="cred-signin-term">{signinView ? <TermSurface term={signinView} hidden={false} /> : null}</div>
            <div className="dlg-actions">
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setSigninTerm(null)}>Close</button>
              <button type="button" className="btn btn-primary btn-sm" disabled={busy === "check"} onClick={() => { setSigninTerm(null); checkSignin(); }}>Check now</button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
      <Dialog.Root open={addOpen} onOpenChange={(open) => { if (!open) closeAdd(); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg">
            <Dialog.Title className="dlg-title">Add provider</Dialog.Title>
            <Dialog.Description className="dlg-body">{"Saved to this machine’s vault."}</Dialog.Description>
            <form className="cred-form" noValidate onSubmit={save}>
              <label className="cred-field">
                <span>Provider</span>
                <select
                  className="cred-input"
                  value={form.provider}
                  onChange={(e) => setForm({ provider: e.target.value, key: form.key })}
                >
                  {pickList.map((p) => <option key={p.id} value={p.id}>{providerName(p.id, p.name)}</option>)}
                </select>
              </label>
              {keyless ? (
                <p className="cred-help">
                  {guidedPick
                    ? pickedName + " signs in through " + cliName + " — no key needed."
                    : pickedName + " signs in inside " + cliName + " itself; there is no key to save here."}
                </p>
              ) : <label className="cred-field">
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
              </label>}
              {guidedPick && !keyless ? (
                <p className="cred-help">
                  {pickedName} also signs in — the OAuth happens in {cliName}, not here.
                </p>
              ) : null}
              <p className="form-error" role="alert" hidden={!formError}>{formError}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                {guidedPick ? (
                  <button
                    type="button"
                    className={keyless ? "btn btn-primary btn-sm" : "btn btn-ghost btn-sm"}
                    disabled={busy === "signin"}
                    onClick={guidedSignin}
                  >
                    {busy === "signin" ? "Opening…" : "Guided sign-in"}
                  </button>
                ) : null}
                <button type="button" className="btn btn-ghost btn-sm" onClick={closeAdd}>Cancel</button>
                {keyless ? null : (
                  <button type="submit" className="btn btn-primary btn-sm" disabled={saving}>{saving ? "Saving…" : "Save"}</button>
                )}
              </div>
            </form>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </PageFrame>
  );
}

// One account, one row: what it is, whose it is, what a vendor says is left,
// what it cost us, and a menu holding only the verbs this roster's flags allow.
function AccountRow({
  provider, account, first, cliName, money, verdict, busy, addLabel,
  onUse, onRename, onVerify, onVerifyProvider, onCheck, onPause, onSignOut, onEdit, onAdd,
}) {
  const name = providerName(provider.id, provider.name);
  const label = account.label || name;
  const identity = identityLine(account, account.usage);
  const kind = kindLabel(account.type);
  const source = sourceLabel(account.origin);
  const chip = healthChip(account.health);
  const state = [kind, account.paused ? "paused" : account.active ? "in use" : ""].filter(Boolean).join(" · ");
  const checked = chip
    ? [chip.message, chip.age ? (chip.age === "now" ? "Checked just now." : "Checked " + chip.age + " ago.") : ""].filter(Boolean).join(" ")
    : "";
  const note = String(provider.note || "").trim();
  // Verify spends a call, and it is asked of whoever can answer: pi checks at
  // the provider, a guest probes the row's credential (ADR-0129). A provider
  // with a note has no honest probe, so its rows hide the item.
  const rowVerify = provider.verify === "row" && !note;
  const providerVerify = provider.verify === "provider";
  const checking = busy === "check:" + account.id;
  return (
    <li className={"prov-tr" + (account.active && !account.paused ? "" : " muted") + (first ? "" : " cont")}>
      <span className="prov-cell prov-provider">
        {first ? <ProviderFace id={provider.id} /> : null}
        {first && provider.custom ? <span className="prov-custom">custom</span> : null}
        {first ? <span className="cred-provider-name" title={name === provider.id ? undefined : provider.id}>{name}</span> : null}
      </span>
      <span className="prov-cell prov-account">
        <span className="cred-label" title={label}>{label}</span>
        {state ? <span className={"prov-auth" + (account.active && !account.paused ? " in" : "")}>{state}</span> : null}
        {source ? <span className="cred-chip">{source}</span> : null}
        {chip ? <span className={"cred-chip is-" + chip.tone} title={checked || undefined}>{chip.label}</span> : null}
        {verdict ? (
          <span className={"prov-verdict " + (verdict.ok ? "ok" : "bad")} title={verdict.reason || verdict.status}>
            {verdict.ok ? verdict.label || cliName + " can use this" : verdict.reason || verdict.status}
          </span>
        ) : null}
      </span>
      <span className="prov-cell prov-ident" data-empty={identity ? undefined : true} title={identity || undefined}>
        {identity || <span className="prov-none">—</span>}
      </span>
      {/* The reading a vendor returned (QuotaStrip, unchanged), or the honest
          word for not having one plus the one click that would get it. Nothing
          here draws a bar we did not fetch (ADR-0031). */}
      <span className="prov-cell prov-left" title="Usage comes from the provider — one request per check.">
        {account.usage ? (
          <QuotaStrip entry={account.usage} busy={checking} onRefresh={onCheck} />
        ) : (
          <span className="quota-strip quota-empty">
            <span className="quota-note">unknown</span>
            <button
              type="button"
              className="quota-refresh"
              disabled={checking}
              title={"Asks " + name + " once for this account's usage."}
              onClick={onCheck}
            >
              {checking ? "Checking" : "Check"}
            </button>
          </span>
        )}
      </span>
      <span className="prov-cell prov-spend" title="What your sessions spent on this provider in the last 7 days">
        {money || <span className="prov-none">—</span>}
      </span>
      <span className="prov-cell prov-actions">
        {/* With several providers the per-provider Add is the only one left:
            the bar would say the same thing one row above it. */}
        {addLabel ? <button type="button" className="btn btn-ghost btn-sm" onClick={onAdd}>{addLabel}</button> : null}
        <DropdownMenu.Root>
          <DropdownMenu.Trigger asChild>
            <button type="button" className="btn btn-ghost btn-sm prov-more" aria-label={"More actions for " + label}>⋯</button>
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content className="prov-menu cred-menu" align="end" sideOffset={6} collisionPadding={8}>
              {account.activatable && !account.active ? (
                <DropdownMenu.Item className="prov-menu-item" onSelect={onUse}>Use</DropdownMenu.Item>
              ) : null}
              <DropdownMenu.Item className="prov-menu-item" onSelect={onRename}>Rename</DropdownMenu.Item>
              {rowVerify ? (
                <DropdownMenu.Item className="prov-menu-item" onSelect={onVerify}>
                  {busy === "verify:" + account.id ? "Verifying…" : VERIFY_LABEL}
                </DropdownMenu.Item>
              ) : null}
              {/* pi's check answers for the provider, not for this row, so
                  every row of that provider offers it (the item names pi, and
                  the verdict lands in every row of the group). */}
              {providerVerify ? (
                <DropdownMenu.Item className="prov-menu-item" onSelect={onVerifyProvider}>
                  {busy === "verify:" + provider.id ? "Checking…" : provider.custom ? VERIFY_LABEL : "Verify with " + cliName}
                </DropdownMenu.Item>
              ) : null}
              {first && provider.custom ? (
                <DropdownMenu.Item className="prov-menu-item" onSelect={onEdit}>Edit provider</DropdownMenu.Item>
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

// gatewayHost is the part of a gateway URL a person recognises.
function gatewayHost(u) {
  try { return new URL(u).host; } catch { return u; }
}
