import { useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { skillSourceSchema } from "@picode/shared/contracts/schemas.js";
import {
  SKILLSSH_NOTE, catalogEmptyLine, catalogInstalled, catalogOrigin, installedSkillKeys, skillCatalogPath, sourceStateLine,
} from "@picode/shared/domain/cliSkills.js";
import { toast } from "../lib/toast.js";

// A license names itself on the card only when it is short (MIT, Apache-2.0);
// a sentence such as "Complete terms in LICENSE.txt" stays in the tooltip of
// the preview, not as a chip that reads "LICENSE".
function shortLicense(text = "") {
  const t = text.trim();
  return /^[A-Za-z0-9.+-]{1,20}$/.test(t) && !/^license$/i.test(t) ? t : "";
}

// The Skills Marketplace (ADR-0196 slice 5): one catalog for every CLI, from
// the built-in sources, the person's own and — only when switched on —
// skills.sh. A card describes; Install opens the same preview, scan and
// consent as a typed source (AddSkillDialog). PiCode never vouches: a card
// names where the skill comes from, nothing more. Pattern: the Packages
// marketplace cards (docs/benchmarks/2026-09-23-skills-marketplace.md).
export default function SkillsMarketplace({ report, onInstall }) {
  const [q, setQ] = useState("");
  const [asked, setAsked] = useState("");
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [reload, setReload] = useState(0);
  const [adding, setAdding] = useState("");
  const [addErr, setAddErr] = useState("");
  const [busy, setBusy] = useState("");

  // Typing settles before it asks (and before anything reaches skills.sh).
  useEffect(() => {
    const t = setTimeout(() => setAsked(q), 250);
    return () => clearTimeout(t);
  }, [q]);

  useEffect(() => {
    let active = true;
    api(skillCatalogPath(asked)).then((next) => {
      if (!active) return;
      setData(next);
      setError(null);
    }).catch((err) => { if (active) setError(err); });
    return () => { active = false; };
  }, [asked, reload]);

  // A source's read ends (skills.catalog) or the sources change (settings).
  useEffect(() => subscribeFeed((event) => {
    if (event.type === "skills.catalog") setReload((n) => n + 1);
    if (event.type === "setting.updated" && String(event.data?.key || "").startsWith("skills.")) setReload((n) => n + 1);
  }), []);

  const installed = installedSkillKeys(report);
  const items = data?.items || [];
  const sources = data?.sources || [];
  const reading = sources.some((s) => s.reading);

  const addSource = async (e) => {
    e.preventDefault();
    const parsed = skillSourceSchema.safeParse({ source: adding });
    if (!parsed.success) { setAddErr(parsed.error.issues[0].message); return; }
    setBusy("add"); setAddErr("");
    try {
      const res = await api("/api/skills/sources", { method: "POST", body: JSON.stringify({ input: parsed.data.source }) });
      toast.ok(res.input + " added.");
      setAdding("");
      setReload((n) => n + 1);
    } catch (x) {
      setAddErr(x.message);
    } finally {
      setBusy("");
    }
  };

  const removeSource = async (input) => {
    setBusy(input);
    try {
      await api("/api/skills/sources", { method: "DELETE", body: JSON.stringify({ input }) });
      toast.ok(input + " removed.");
      setReload((n) => n + 1);
    } catch (x) {
      toast.error(x.message);
    } finally {
      setBusy("");
    }
  };

  const setSkillsSH = async (on) => {
    setData((d) => (d ? { ...d, skillssh: on } : d));
    try {
      await api("/api/skills/skillssh", { method: "PUT", body: JSON.stringify({ on }) });
      setReload((n) => n + 1);
    } catch (x) {
      setData((d) => (d ? { ...d, skillssh: !on } : d));
      toast.error(x.message);
    }
  };

  let body;
  if (error && !data) {
    body = (
      <div className="cli-notice is-error" role="alert">
        <span>{error.message}</span>
        <button type="button" className="btn btn-ghost btn-sm" onClick={() => setReload((n) => n + 1)}>Try again</button>
      </div>
    );
  } else if (!data || (!items.length && reading)) {
    body = (
      <div className="gpkg-grid gpkg-skel" role="status" aria-label="Reading the sources">
        {[0, 1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="gpkg-card" aria-hidden="true">
            <div className="gpkg-card-head"><div className="skel-line w-50" /></div>
            <div className="skel-line" /><div className="skel-line w-80" />
            <div className="gpkg-card-foot"><div className="skel-line w-40" /></div>
          </div>
        ))}
      </div>
    );
  } else if (!items.length) {
    body = (
      <div className="cli-notice" role="status">
        <span>{catalogEmptyLine(asked, reading)}</span>
        {asked.trim() ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => setQ("")}>Clear the search</button> : null}
      </div>
    );
  } else {
    body = (
      <ul className="gpkg-grid skills-market-grid">
        {items.map((it) => {
          const origin = catalogOrigin(it);
          const has = catalogInstalled(installed, it);
          return (
            <li key={it.id} className="gpkg-card">
              <div className="gpkg-card-head">
                <span className="gpkg-card-name" title={it.name}>{it.name}</span>
                {shortLicense(it.license) ? <span className="pkg-type" title={it.license}>{shortLicense(it.license)}</span> : null}
              </div>
              {it.description
                ? <p className="gpkg-card-desc" title={it.description}>{it.description}</p>
                : <p className="gpkg-card-desc is-muted">skills.sh lists no description; open it to read the skill.</p>}
              <div className="gpkg-card-meta">
                <span className="gpkg-card-src" title={origin.title}>{origin.label}</span>
              </div>
              <div className="gpkg-card-foot">
                {has
                  ? <span className="gpkg-status" title="This CLI already has this skill from this source">Installed</span>
                  : <button type="button" className="btn btn-primary btn-sm" onClick={() => onInstall(it)}>Install</button>}
                {it.url ? <a className="btn btn-ghost btn-sm" href={it.url} target="_blank" rel="noreferrer">Source ↗</a> : null}
              </div>
            </li>
          );
        })}
      </ul>
    );
  }

  return (
    <div className="skills-market">
      <div className="cli-memory-bar" data-align-row>
        <input
          className="cli-memory-search skills-market-search"
          type="search"
          placeholder={data?.skillssh ? "Search the sources and skills.sh" : "Search skills"}
          value={q}
          aria-label="Search skills"
          onChange={(e) => setQ(e.target.value)}
        />
      </div>
      {data?.skillsshError ? <p className="cli-memory-note is-warn">{data.skillsshError}. The sources above still answer.</p> : null}
      {body}

      <details className="skills-sources">
        <summary>Sources · {sources.length}{reading ? " · reading" : ""}</summary>
        <ul className="skills-source-list">
          {sources.map((s) => (
            <li key={s.input} className="skills-source-row">
              <span className="skills-source-name" title={s.input}>{s.input}</span>
              {s.builtin ? <span className="pkg-type">Built in</span> : null}
              <span className={"skills-source-state" + (s.error && !s.count ? " is-warn" : "")}>{sourceStateLine(s)}</span>
              {s.builtin ? null : (
                <button type="button" className="btn btn-ghost btn-sm" disabled={busy === s.input} onClick={() => removeSource(s.input)}>Remove</button>
              )}
            </li>
          ))}
        </ul>
        <form className="skills-source-add" noValidate onSubmit={addSource} data-align-row>
          <input
            className="cred-input"
            autoComplete="off"
            spellCheck={false}
            placeholder="owner/repo or https://site"
            aria-label="Add a source"
            aria-invalid={addErr ? true : undefined}
            value={adding}
            onChange={(e) => { setAddErr(""); setAdding(e.target.value); }}
          />
          <button type="submit" className="btn btn-sm" disabled={busy === "add" || !adding.trim()}>{busy === "add" ? <span className="cred-waiting">Adding</span> : "Add source"}</button>
        </form>
        <p className="form-error" role="alert" hidden={!addErr}>{addErr}</p>
        <label className="skills-sh-row">
          <Switch.Root className="rx-switch" checked={!!data?.skillssh} onCheckedChange={setSkillsSH} aria-label="Also search skills.sh">
            <Switch.Thumb className="rx-switch-thumb" />
          </Switch.Root>
          <span><strong>Also search skills.sh</strong> — {SKILLSSH_NOTE}</span>
        </label>
      </details>
    </div>
  );
}
