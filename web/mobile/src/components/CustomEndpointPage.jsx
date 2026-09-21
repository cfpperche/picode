import { useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { toast, toastError } from "../lib/toast.js";
import { cliProvidersHash } from "@picode/shared/domain/cliProviders.js";
import { DOCS_BASE } from "../lib/commandDocs.js";
import { IconBack, IconChevronRight, IconDocs } from "./Icons.jsx";
import { validateCustomProvider, customProviderPayload, customProviderForm, customTakenIds, customModelIds, syncModelLimits, modelLimitRow, THINKING_LEVELS, THINKING_FORMAT_NEEDS, CUSTOM_INPUT_MODALITIES } from "@picode/shared/domain/customProviders.js";
import { loadModelsFor, modelLoadChanges, loadingLine } from "@picode/shared/client/modelLoad.js";
import { CUSTOM_PROVIDER_APIS, CUSTOM_THINKING_FORMATS, OMP_THINKING_FORMATS, customApiHint } from "@picode/shared/contracts/schemas.js";
import { pushRecent } from "@picode/shared/domain/providerRecents.js";

// Custom endpoint (ADR-0129) as a page, not a dialog step: benchmarks.md
// refuses modals for flows longer than 2 fields, and this form carries four
// core fields plus Load/Verify plus an Advanced section with a JSON editor.
// Adapted from Cursor's Customize page (docs/benchmarks/2026-09-12-connectors-ux.md):
// the pick flow stays a dialog, the definition form gets the full pane.
// Teaching copy lives in guide/providers.md — the page carries state and
// actions only.
export default function CustomEndpointPage({ catalog, rosterProviders, cli = "pi", onRefresh, editId = "" }) {
  // The definitions each CLI reads from its own file: pi's catalog serves
  // models.json, omp's roster carries models.yml rows (ADR-0129 and the
  // ADR-0169 amendment).
  const list = cli === "pi"
    ? (catalog && catalog.providers ? catalog.providers : [])
    : (Array.isArray(rosterProviders) ? rosterProviders : []);
  const editing = editId ? list.find((p) => p.id === editId && p.custom) || null : null;
  // pi's catalog row carries the editable shape on top; omp's roster nests it
  // under definition beside the row's own id.
  const editingShape = editing && cli !== "pi" ? { id: editing.id, ...(editing.definition || {}) } : editing;
  const isEdit = !!editId;
  const [cf, setCf] = useState(() => customProviderForm(editingShape));
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [load, setLoad] = useState({ state: "idle", text: "", tone: "hint" });
  const [verify, setVerify] = useState({ state: "idle", text: "", tone: "hint" });
  const title = isEdit ? "Edit provider" : "Custom provider";
  const back = cliProvidersHash(cli);
  // omp's models.yml accepts a narrower advanced set than pi: no include_usage
  // switch, five thinking formats, no chat-template objects and no per-model
  // level map. The controls that would write keys omp's schema refuses are
  // not rendered — the server refuses them too, but a control that cannot
  // work is a lie the form should not tell.
  const isPi = cli === "pi";
  const formats = isPi ? CUSTOM_THINKING_FORMATS : CUSTOM_THINKING_FORMATS.filter((f) => OMP_THINKING_FORMATS.has(f.value));

  // The id list drives the per-model limit rows: a row appears with its id and
  // disappears with it, and the numbers already typed survive the trip.
  const modelIds = customModelIds(cf.modelsText);
  function setModelsText(e) {
    const text = e.target.value;
    setCf((f) => ({ ...f, modelsText: text, modelLimits: syncModelLimits(f.modelLimits, customModelIds(text)) }));
  }
  function setLimit(id, field) {
    return (e) => {
      const value = e.target.value;
      setCf((f) => ({ ...f, modelLimits: { ...f.modelLimits, [id]: { ...modelLimitRow(f.modelLimits, id), [field]: value } } }));
    };
  }
  function setCost(id, field) {
    return (e) => {
      const value = e.target.value;
      setCf((f) => {
        const row = modelLimitRow(f.modelLimits, id);
        return { ...f, modelLimits: { ...f.modelLimits, [id]: { ...row, cost: { ...row.cost, [field]: value } } } };
      });
    };
  }
  function toggleInput(id, modality) {
    return (e) => setCf((f) => {
      const row = modelLimitRow(f.modelLimits, id);
      const next = new Set(row.input);
      if (e.target.checked) next.add(modality); else next.delete(modality);
      return { ...f, modelLimits: { ...f.modelLimits, [id]: { ...row, input: CUSTOM_INPUT_MODALITIES.filter((m) => next.has(m)) } } };
    });
  }
  function setLevelValue(lvl) {
    return (e) => {
      const value = e.target.value;
      setCf((f) => ({ ...f, thinkingLevelValues: { ...f.thinkingLevelValues, [lvl]: value } }));
    };
  }
  function setField(k) {
    return (e) => setCf((f) => ({ ...f, [k]: e.target.value }));
  }
  function setCheck(k) {
    return (e) => setCf((f) => ({ ...f, [k]: e.target.checked }));
  }
  // toggleLevel keeps the selection in pi's scale order, so the saved
  // thinkingLevelMap reads like the catalog does.
  function toggleLevel(lvl) {
    return (e) => setCf((f) => {
      const next = new Set(f.thinkingLevels);
      if (e.target.checked) next.add(lvl); else next.delete(lvl);
      return { ...f, thinkingLevels: THINKING_LEVELS.filter((l) => next.has(l)) };
    });
  }

  async function loadModels() {
    if (load.state === "loading") return;
    setLoad({ state: "loading", text: loadingLine(cf.baseUrl), tone: "hint" });
    try {
      const res = await loadModelsFor({ baseUrl: cf.baseUrl, api: cf.api, key: cf.key, id: cf.id, cli });
      const { changes, line } = modelLoadChanges(res, cf);
      setCf((f) => ({ ...f, ...changes }));
      setLoad({ state: "done", text: line, tone: "hint" });
    } catch (ex) {
      const body = ex && ex.body;
      setLoad({ state: "failed", text: (body && body.error) || (ex && ex.message) || "The model list could not be read.", tone: "bad" });
    }
  }

  // verifyEndpoint spends one minimal real request on what the form holds —
  // one word in, one token out — so a wrong key or URL is caught before
  // saving. The button says the cost; the line reports what it spent.
  async function verifyEndpoint() {
    if (verify.state === "loading") return;
    const model = customModelIds(cf.modelsText)[0] || "";
    if (!model) {
      setVerify({ state: "failed", tone: "bad", text: "Add a model id first — verification has to ask for one." });
      return;
    }
    setVerify({ state: "loading", tone: "hint", text: "Sending one 1-token request to " + model + "…" });
    try {
      const res = await api("/api/providers/" + encodeURIComponent(cf.id || "endpoint") + "/verify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ baseUrl: cf.baseUrl, api: cf.api, key: cf.key, model, cli }),
      });
      if (res && res.ok) setVerify({ state: "done", tone: "hint", text: (res.label || "The provider answered") + " — the key works." });
      else setVerify({ state: "failed", tone: "bad", text: (res && (res.reason || res.status)) || "The provider could not be verified." });
    } catch (ex) {
      const body = ex && ex.body;
      setVerify({ state: "failed", tone: "bad", text: (body && body.error) || (ex && ex.message) || "The provider could not be verified." });
    }
  }

  async function save(e) {
    e.preventDefault();
    const taken = customTakenIds(list, isEdit ? editId : null);
    const parsed = validateCustomProvider(cf, { takenIds: taken, requireKey: !isEdit });
    if (!parsed.ok) { setErr(parsed.error); return; }
    setBusy(true);
    setErr("");
    try {
      await api("/api/providers/custom/" + encodeURIComponent(parsed.value.id), {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...customProviderPayload(parsed.value), cli }),
      });
      toast.ok(isEdit ? "Provider updated." : "Provider added to " + (cli === "pi" ? "pi" : "omp") + ".");
      if (cli === "pi") pushRecent(parsed.value.id);
      if (onRefresh) await onRefresh();
      // pi returns to its canonical providers route; omp back to its own pane
      // (go("providers") is pi's legacy alias and would land on the wrong CLI).
      if (typeof location !== "undefined") location.hash = cliProvidersHash(cli);
    } catch (ex) {
      toastError(ex);
    } finally {
      setBusy(false);
    }
  }

  if (isEdit && !editing) {
    return (
      <section id="custom-endpoint-view" aria-label={title}>
        <p className="prov-empty"><span>Unknown provider.</span><a className="btn btn-ghost btn-sm" href={back}>Back to providers</a></p>
      </section>
    );
  }
  const hint = customApiHint(cf.api);
  return (
    <section id="custom-endpoint-view" aria-label={title}>
      <div className="prov-page-head">
        <a className="btn btn-ghost btn-sm" href={back}><IconBack /> Providers</a>
        <h3>{title}</h3>
        <a className="btn btn-ghost btn-sm" href={DOCS_BASE + "/guide/providers#custom-provider"} target="_blank" rel="noreferrer"><IconDocs /> Setup guide</a>
      </div>
      <form className="form-new prov-form prov-page" noValidate onSubmit={save}>
        <div className="prov-set">
          <span className="prov-legend">Identity</span>
          <div className="prov-grid">
            <div className="prov-field">
              <label className="prov-label" htmlFor="cf-name">Name</label>
              <input
                id="cf-name"
                value={cf.id}
                onChange={setField("id")}
                placeholder="e.g. cheaperinference"
                autoComplete="off" spellCheck="false"
                disabled={isEdit}
                aria-label="Provider name"
              />
            </div>
            <div className="prov-field">
              <label className="prov-label" htmlFor="cf-api">API type</label>
              <select id="cf-api" value={cf.api} onChange={setField("api")} aria-label="API type">
                {CUSTOM_PROVIDER_APIS.map((a) => (
                  <option key={a.value} value={a.value}>{a.label}</option>
                ))}
              </select>
            </div>
          </div>
        </div>
        <div className="prov-set">
          <span className="prov-legend">Connection</span>
          <div className="prov-field">
            <label className="prov-label" htmlFor="cf-url">Base URL</label>
            <input
              id="cf-url"
              type="url"
              value={cf.baseUrl}
              onChange={setField("baseUrl")}
              placeholder={hint.placeholder}
              autoComplete="off" spellCheck="false"
              aria-label="Base URL"
              aria-describedby="cf-url-hint"
            />
            <p className="prov-help" id="cf-url-hint">{hint.urlHint}</p>
          </div>
          <div className="prov-field">
            <label className="prov-label" htmlFor="cf-key">API key</label>
            <div className="prov-inline">
              <input
                id="cf-key"
                type="password"
                value={cf.key}
                onChange={setField("key")}
                placeholder={isEdit ? "Leave blank to keep the saved one" : ""}
                autoComplete="off"
                aria-label="API key"
              />
              <button type="button" className="btn btn-ghost" onClick={verifyEndpoint} disabled={verify.state === "loading" || load.state === "loading"}>
                {verify.state === "loading" ? "Verifying…" : "Verify key"}
              </button>
            </div>
            {verify.text ? <p className={verify.tone === "bad" ? "prov-hint prov-bad" : "prov-hint"} role="status">{verify.text}</p> : null}
          </div>
        </div>
        <div className="prov-set">
          <span className="prov-legend">Models</span>
          <div className="prov-field">
            <label className="prov-label" htmlFor="cf-models">Model ids</label>
            <textarea
              id="cf-models"
              value={cf.modelsText}
              onChange={setModelsText}
              rows={3}
              placeholder={"One per line — exactly as the gateway spells them"}
              spellCheck="false"
              aria-label="Model ids"
            />
            <div className="prov-load-row">
              <button type="button" className="btn btn-ghost" onClick={loadModels} disabled={load.state === "loading" || verify.state === "loading"}>
                {load.state === "loading" ? "Loading…" : "Load models"}
              </button>
            </div>
            {load.text ? <p className={load.tone === "bad" ? "prov-hint prov-bad" : "prov-hint"} role="status">{load.text}</p> : null}
          </div>
          {modelIds.length ? (
            <div className="prov-set">
              <span className="prov-legend">Model details</span>
              <p className="prov-help">Blank leaves the default. Cost is USD per 1M tokens — fill all four rates or leave them blank.</p>
              <div className="prov-limits">
                {modelIds.map((id) => {
                  const row = modelLimitRow(cf.modelLimits, id);
                  return (
                    <div className="prov-limit-row" key={id} data-model={id}>
                      <span className="prov-limit-id" title={id}>{id}</span>
                      <input
                        className="prov-limit-name"
                        value={row.name}
                        onChange={setLimit(id, "name")}
                        placeholder="Display name"
                        autoComplete="off" spellCheck="false"
                        aria-label={"Display name for " + id}
                      />
                      <input
                        className="prov-limit-ctx"
                        value={row.contextWindow}
                        onChange={setLimit(id, "contextWindow")}
                        inputMode="numeric"
                        placeholder="Context"
                        aria-label={"Context window for " + id}
                      />
                      <input
                        className="prov-limit-max"
                        value={row.maxTokens}
                        onChange={setLimit(id, "maxTokens")}
                        inputMode="numeric"
                        placeholder="Max output"
                        aria-label={"Max output for " + id}
                      />
                      <div className="prov-model-sub">
                        <span className="prov-model-reads">Reads
                          {CUSTOM_INPUT_MODALITIES.map((modality) => (
                            <label key={modality} className="prov-model-check">
                              <input
                                type="checkbox"
                                checked={row.input.includes(modality)}
                                onChange={toggleInput(id, modality)}
                                aria-label={modality + " input for " + id}
                              />
                              <span>{modality}</span>
                            </label>
                          ))}
                        </span>
                        <span className="prov-model-cost">$/1M
                          {[["input", "In"], ["output", "Out"], ["cacheRead", "Rd"], ["cacheWrite", "Wr"]].map(([field, short]) => (
                            <input
                              key={field}
                              value={row.cost[field]}
                              onChange={setCost(id, field)}
                              inputMode="decimal"
                              placeholder={short}
                              aria-label={short + " cost ($/1M tokens) for " + id}
                            />
                          ))}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          ) : null}
        </div>
        <details className="prov-adv">
          <summary>
            <IconChevronRight />
            Advanced
          </summary>
          <div className="prov-set">
            <span className="prov-legend">Request compatibility</span>
            <div className="prov-checks">
              <label className="prov-check">
                <input type="checkbox" checked={cf.compatDeveloper} onChange={setCheck("compatDeveloper")} />
                <span>OpenAI <code>developer</code> role — most gateways: leave off</span>
              </label>
              <label className="prov-check">
                <input type="checkbox" checked={cf.compatReasoning} onChange={setCheck("compatReasoning")} />
                <span><code>reasoning_effort</code> — reasoning models only</span>
              </label>
              {isPi ? (
                <label className="prov-check">
                  <input type="checkbox" checked={cf.compatStreaming} onChange={setCheck("compatStreaming")} />
                  <span><code>include_usage</code> in streams — token counts ride the stream</span>
                </label>
              ) : null}
            </div>
          </div>
          <div className="prov-set">
            <span className="prov-legend">Thinking</span>
            <div className="prov-field">
              <label className="prov-label" htmlFor="cf-thinking">Thinking format</label>
              <select
                id="cf-thinking"
                value={cf.thinkingFormat}
                onChange={setField("thinkingFormat")}
                aria-label="Thinking format"
              >
                {formats.map((f) => (
                  <option key={f.value} value={f.value}>{f.label}</option>
                ))}
              </select>
            </div>
            {isPi && (THINKING_FORMAT_NEEDS[cf.thinkingFormat] === "kwargs" || THINKING_FORMAT_NEEDS[cf.thinkingFormat] === "args") ? (
              <div className="prov-field">
                <label className="prov-label" htmlFor="cf-json">
                  {THINKING_FORMAT_NEEDS[cf.thinkingFormat] === "kwargs" ? "Chat template kwargs" : "Chat template args"}
                </label>
                <textarea
                  id="cf-json"
                  value={THINKING_FORMAT_NEEDS[cf.thinkingFormat] === "kwargs" ? cf.chatTemplateKwargs : cf.chatTemplateArgs}
                  onChange={setField(THINKING_FORMAT_NEEDS[cf.thinkingFormat] === "kwargs" ? "chatTemplateKwargs" : "chatTemplateArgs")}
                  rows={3}
                  spellCheck="false"
                  className="prov-json"
                  placeholder={'{"thinking": {"$var": "thinking.enabled"}}'}
                  aria-label={THINKING_FORMAT_NEEDS[cf.thinkingFormat] === "kwargs" ? "Chat template kwargs" : "Chat template args"}
                />
                <p className="prov-help">
                  JSON sent as <code>{THINKING_FORMAT_NEEDS[cf.thinkingFormat] === "kwargs" ? "chat_template_kwargs" : "chat_template_args"}</code>.
                  Use <code>{'{"$var": "thinking.enabled"}'}</code> (or thinking.effort / thinking.budget) to let pi decide the value.
                </p>
              </div>
            ) : null}
            <div className="prov-checks">
              <label className="prov-check">
                <input type="checkbox" checked={cf.reasoningModel} onChange={setCheck("reasoningModel")} />
                <span>Reasoning model{isPi ? " — lets you pick thinking levels" : ""}</span>
              </label>
            </div>
            {cf.reasoningModel && isPi ? (
              <div className="prov-field">
                <label className="prov-label">Thinking levels</label>
                <div className="prov-levels" role="group" aria-label="Thinking levels">
                  {THINKING_LEVELS.map((lvl) => {
                    const on = cf.thinkingLevels.includes(lvl);
                    return (
                      <span key={lvl} className="prov-level-pair">
                        <label className={"prov-level" + (on ? " on" : "")}>
                          <input type="checkbox" checked={on} onChange={toggleLevel(lvl)} />
                          <span>{lvl}</span>
                        </label>
                        {on ? (
                          <input
                            className="prov-level-value"
                            value={(cf.thinkingLevelValues || {})[lvl] || ""}
                            onChange={setLevelValue(lvl)}
                            placeholder={lvl}
                            autoComplete="off" spellCheck="false"
                            aria-label={"Provider value for " + lvl}
                          />
                        ) : null}
                      </span>
                    );
                  })}
                </div>
                <p className="prov-help">Unchecked levels are hidden in the model picker. A filled value overrides the provider string a level sends.</p>
              </div>
            ) : null}
          </div>
        </details>
        <p className="form-error" hidden={!err}>{err}</p>
        <div className="prov-page-actions" data-align-row>
          <a className="btn btn-ghost btn-sm" href={back}>Cancel</a>
          <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>{isEdit ? "Save" : "Add provider"}</button>
        </div>
      </form>
    </section>
  );
}
