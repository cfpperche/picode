import { useEffect, useMemo, useState } from "react";
import { api } from "../lib/api.js";
import { toastError } from "../lib/toast.js";
import PageFrame from "./PageFrame.jsx";
import TermAppearance from "./TermAppearance.jsx";
import { termsetRoute } from "../lib/routes.js";
import {
  INHERIT, selectedKey, choicesFor, effectText,
  inheritedValueFor, matchesQuery, groupCatalog, withChoice,
} from "../lib/termSettings.js";

// Terminal behaviour over the FULL tmux option space (ADR-0024). Two tiers:
// curated flags up top with rich controls and consequences spelled out, then
// the whole catalog of the running tmux — searchable, grouped by reach, the
// dangerous entries labelled instead of hidden. Values are validated by tmux
// itself at apply time; its refusal message is what the user sees.
//
// Appearance (font, colours, cursor) is not here on purpose: it belongs to
// the browser (Preferences), while everything on this page belongs to the
// terminal and is shared by every device that opens it.
export default function TermSettingsPage({ hidden, terminals }) {
  const [termId, setTermId] = useState(termsetRoute());
  useEffect(() => {
    const onHash = () => setTermId(termsetRoute());
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  const isGlobal = !termId;
  const term = (terminals || []).find((t) => t.id === termId) || null;
  const path = isGlobal ? "/api/terminals/settings" : `/api/terminals/${termId}/settings`;

  const [data, setData] = useState(null);
  const [catalog, setCatalog] = useState(null);
  const [query, setQuery] = useState("");

  useEffect(() => {
    if (hidden) return;
    let alive = true;
    setData(null);
    api(path).then((d) => { if (alive) setData(d); }).catch((e) => { if (alive) toastError(e); });
    api("/api/terminals/settings/catalog")
      .then((c) => { if (alive) setCatalog(c.catalog || []); })
      .catch(() => { if (alive) setCatalog([]); });
    return () => { alive = false; };
  }, [hidden, path]);

  function patch(key, value) {
    return api(path, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ [key]: value }),
    }).then((fresh) => { setData(fresh); return true; })
      .catch((e) => { toastError(e); return false; });
  }

  // Featured picks move optimistically (a segmented control must not sit on
  // the old value for a round trip); generic rows confirm on response instead
  // — a text value can be refused by tmux, and showing it accepted first
  // would be a lie the toast then has to walk back.
  function pickFeatured(key, value) {
    setData((prev) => {
      if (!prev) return prev;
      patch(key, value).then((ok) => { if (!ok) setData(prev); });
      return { ...prev, values: withChoice(prev.values, key, value) };
    });
  }

  const groups = useMemo(
    () => groupCatalog((catalog || []).filter((r) => matchesQuery(r, query)), { isGlobal }),
    [catalog, query, isGlobal],
  );
  const catalogValue = useMemo(() => {
    const m = {};
    for (const r of catalog || []) m[r.name] = r.value;
    return m;
  }, [catalog]);

  const title = isGlobal ? "Terminal defaults" : (term ? term.name : "Terminal settings");
  const context = isGlobal
    ? "Defaults every terminal inherits; each can override."
    : "Only what this terminal changes; the rest follows the defaults.";

  return (
    <PageFrame id="termset-view" title={title} context={context} hidden={hidden} wide>
      {!data || !catalog ? <PageSkeleton /> : (
        <div className="termset-page">
          {isGlobal ? (
            <section className="termset-cat">
              <h3 className="termset-cat-title">Appearance — this browser</h3>
              <p className="termset-cat-note">
                Font, colors and cursor are remembered by this browser, so each
                device can look its own way. Everything below travels with the
                terminal instead.
              </p>
              <TermAppearance active={!hidden} />
            </section>
          ) : null}
          <section className="termset-fields">
            {(data.flags || [])
              .filter((f) => isGlobal || f.scope !== "server")
              // The search filters the featured tier too: with a query active,
              // an unfiltered featured control sits exactly where a result is
              // expected and collects the click meant for it — measured, not
              // hypothetical: it caught the author.
              .filter((f) => matchesQuery({ name: f.key + " " + f.label, danger: f.danger }, query))
              .map((flag) => (
                <FeaturedField key={flag.key} flag={flag} data={data} isGlobal={isGlobal} onPick={pickFeatured} />
              ))}
          </section>

          <div className="termset-search-row">
            <input
              type="search"
              className="dlg-input termset-search"
              placeholder={`Search ${catalog.length} tmux options…`}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              aria-label="Search tmux options"
            />
          </div>

          <CatalogSection
            title={isGlobal ? "All PiCode terminals" : "This terminal"}
            note={isGlobal
              ? "Session and window options. Defaults for every terminal; each can override."
              : "Session and window options, for this terminal only."}
            rows={groups.perTerminal}
            data={data}
            catalogValue={catalogValue}
            onPatch={patch}
          />

          {isGlobal ? (
            <CatalogSection
              title="This machine's tmux"
              note="Server options: one value for every tmux session on this machine — PiCode's and yours alike. tmux keeps these per server, not per terminal."
              rows={groups.server}
              data={data}
              catalogValue={catalogValue}
              onPatch={patch}
            />
          ) : (
            <p className="termset-foot">
              Appearance (font, colors, cursor) and server-wide options live in
              the <a href="#/termset">global panel</a> — the first belongs to
              the browser, the second to the whole machine, so neither can be
              promised per terminal.
            </p>
          )}

          <p className="termset-foot">
            Values in the sections above Appearance are checked by tmux itself —
            an entry it refuses is reported in its own words.
          </p>
        </div>
      )}
    </PageFrame>
  );
}

function FeaturedField({ flag, data, isGlobal, onPick }) {
  const chosen = selectedKey(data.values, flag.key);
  const name = `termset-${flag.key}`;
  return (
    <div className="termset-field">
      <div className="termset-head">
        <span className="termset-label">{flag.label}</span>
        {flag.scope === "server" ? <span className="termset-chip termset-chip-server">Machine-wide</span> : null}
        {!isGlobal && chosen !== INHERIT ? <span className="termset-chip">Custom</span> : null}
      </div>
      <div className="termset-seg" role="radiogroup" aria-label={flag.label}>
        {choicesFor(flag, data.inherited, isGlobal).map((c) => {
          const id = `${name}-${c.key === INHERIT ? "inherit" : c.key}`;
          return (
            <label className="termset-seg-opt" key={id} htmlFor={id}>
              <input id={id} type="radio" name={name} checked={c.key === chosen} onChange={() => onPick(flag.key, c.key)} />
              <span className="termset-seg-face">{c.label}</span>
            </label>
          );
        })}
      </div>
      <p className="termset-help">{flag.help}</p>
      {flag.danger ? <p className="termset-danger">⚠ {flag.danger}</p> : null}
      <p className="termset-effect">{effectText(flag.effect)}</p>
    </div>
  );
}

function CatalogSection({ title, note, rows, data, catalogValue, onPatch }) {
  return (
    <section className="termset-cat">
      <h3 className="termset-cat-title">{title}</h3>
      <p className="termset-cat-note">{note}</p>
      {rows.length === 0 ? (
        <p className="termset-cat-empty">No options match.</p>
      ) : (
        <ul className="termset-cat-list">
          {rows.map((row) => (
            <CatalogRow key={row.scope + "/" + row.name} row={row} data={data} catalogValue={catalogValue} onPatch={onPatch} />
          ))}
        </ul>
      )}
    </section>
  );
}

function CatalogRow({ row, data, catalogValue, onPatch }) {
  const stored = data.values && Object.prototype.hasOwnProperty.call(data.values, row.name)
    ? data.values[row.name] : null;
  const inherited = inheritedValueFor(row.name, data.inherited, catalogValue[row.name]);
  const [draft, setDraft] = useState(null); // null = not editing
  const [busy, setBusy] = useState(false);

  async function submit(value) {
    setBusy(true);
    const ok = await onPatch(row.name, value);
    setBusy(false);
    if (ok) setDraft(null);
  }

  const isArray = row.kind === "array";
  const overridden = stored !== null;

  return (
    <li className={"termset-row" + (overridden ? " overridden" : "")}>
      <div className="termset-row-head">
        <code className="termset-row-name">{row.name}</code>
        {row.scope === "window" ? <span className="termset-chip-scope" title="A window option; a PiCode terminal is one window, so it is per-terminal.">window</span> : null}
        {overridden ? <span className="termset-chip">Custom</span> : null}
      </div>

      {isArray ? (
        <p className="termset-row-note">A list ({row.name}[0], [1], …) — editable in tmux.conf; shown here so it is not hidden.</p>
      ) : row.kind === "bool" ? (
        <div className="termset-seg termset-seg-sm" role="radiogroup" aria-label={row.name}>
          {[[INHERIT, `Inherit (${inherited === "on" ? "On" : "Off"})`], ["on", "On"], ["off", "Off"]].map(([val, label]) => {
            const id = `cat-${row.scope}-${row.name}-${val === INHERIT ? "inherit" : val}`;
            const chosen = stored === null ? INHERIT : stored;
            return (
              <label className="termset-seg-opt" key={id} htmlFor={id}>
                <input id={id} type="radio" name={`cat-${row.scope}-${row.name}`} disabled={busy}
                  checked={val === chosen} onChange={() => submit(val)} />
                <span className="termset-seg-face">{label}</span>
              </label>
            );
          })}
        </div>
      ) : (
        <div className="termset-row-edit">
          <input
            className="dlg-input termset-row-input"
            value={draft ?? (stored ?? "")}
            placeholder={`Inherit: ${inherited === "" ? "(empty)" : inherited}`}
            disabled={busy}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter" && draft !== null) submit(draft); }}
            aria-label={row.name}
          />
          <button type="button" className="btn btn-sm" disabled={busy || draft === null} onClick={() => submit(draft)}>Apply</button>
          {overridden ? (
            <button type="button" className="btn btn-ghost btn-sm" disabled={busy} title="Back to inherited" onClick={() => submit(null)}>Reset</button>
          ) : null}
        </div>
      )}

      {row.danger ? <p className="termset-danger">⚠ {row.danger}</p> : null}
    </li>
  );
}

function PageSkeleton() {
  return (
    <div className="termset-page" aria-hidden="true">
      <div className="termset-field">
        <div className="skel-line w-40" />
        <div className="termset-seg termset-seg-skel" />
        <div className="skel-line w-90 termset-skel-help" />
      </div>
      <div className="skel-line w-70 termset-skel-help" />
      <div className="skel-line w-80 termset-skel-help" />
    </div>
  );
}
