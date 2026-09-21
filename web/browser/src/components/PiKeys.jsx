import { useEffect, useMemo, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { isReservedChord } from "@picode/shared/domain/browserChord.js";
import { effectiveKeys, isOverride, matchKeys, platformAlternates, fromEvent } from "@picode/shared/domain/piKey.js";
import { formatChord } from "../lib/appKeys.js";
import { KEYBOARD_CLIS, pickupLine } from "@picode/shared/domain/cliKeys.js";

// The pane's own CLI. Today only Pi ships an editor, so this constant is the
// only thing that would move when a guest's adapter lands (P2): the report
// carries the rest — the file, the host's platform, the catalog.
const CLI = KEYBOARD_CLIS[0];

// The keyboard map of the agent CLI this pane belongs to. Pi is the only CLI
// whose map PiCode can write today; the report carries the file it writes, the
// host's platform (nine of pi's actions bind differently on Windows and WSL)
// and the effective defaults for this machine, so nothing here guesses.
const PLATFORM_LABEL = { windows: "Windows", wsl: "WSL", linux: "Linux", darwin: "macOS" };

export default function PiKeys({ disabled = false }) {
  const [rep, setRep] = useState(null);
  const [err, setErr] = useState("");
  const [q, setQ] = useState("");
  const [facet, setFacet] = useState("all");
  const [keyFilter, setKeyFilter] = useState("");
  const [listen, setListen] = useState("");
  const [findKey, setFindKey] = useState(false);
  const [busy, setBusy] = useState(false);

  function load() {
    setErr("");
    api("/api/cli-keys?cli=" + CLI.id).then(setRep).catch((e) => { setRep(null); setErr(e.message || "Can't load keys."); });
  }

  useEffect(() => { load(); }, []);

  const platform = rep?.platform || "";
  const user = rep?.user || {};
  const actions = rep?.actions || [];

  // chord -> the actions that answer to it. Pi's contexts overlap on purpose
  // (52 of its 89 actions share a chord), so this is reported, never "fixed".
  const index = useMemo(() => {
    const map = new Map();
    for (const action of actions) {
      for (const chord of new Set(effectiveKeys(action, user, platform))) {
        if (!map.has(chord)) map.set(chord, []);
        map.get(chord).push(action);
      }
    }
    return map;
  }, [actions, user, platform]);

  const counts = useMemo(() => {
    let changed = 0, shared = 0, off = 0;
    for (const action of actions) {
      const keys = effectiveKeys(action, user, platform);
      if (isOverride(action, user)) changed++;
      if (!keys.length) off++;
      if (keys.some((chord) => (index.get(chord) || []).length > 1)) shared++;
    }
    return { all: actions.length, changed, shared, off };
  }, [actions, user, platform, index]);

  // Two listeners can own the keyboard: the row that is capturing a chord and
  // the search box's "find by key". Escape is the way out of both and can
  // never be recorded — it is the key every other control in the app exits on.
  useEffect(() => {
    if ((!listen && !findKey) || disabled) return undefined;
    function onKey(ev) {
      if (ev.key === "Escape") {
        ev.preventDefault();
        setListen("");
        setFindKey(false);
        return;
      }
      const chord = fromEvent(ev);
      if (!chord) return;
      ev.preventDefault();
      ev.stopPropagation();
      if (findKey) {
        setKeyFilter(chord);
        setFindKey(false);
        return;
      }
      const action = actions.find((a) => a.id === listen);
      if (!action) return;
      const keys = effectiveKeys(action, user, platform);
      setListen("");
      if (keys.includes(chord)) return;
      save(listen, keys.concat(chord));
    }
    window.addEventListener("keydown", onKey, true);
    return () => window.removeEventListener("keydown", onKey, true);
  }, [listen, findKey, rep, disabled]);

  // The write is one reversible row, so the row updates before the round trip
  // and the server's report replaces it (a refusal reverts it and says why).
  async function put(body, nextUser, done) {
    if (disabled) return;
    const before = rep;
    setRep({ ...rep, user: nextUser });
    try {
      const next = await api("/api/cli-keys", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ cli: CLI.id, ...body }),
      });
      setRep(next);
      toast.ok(done);
    } catch (e) {
      setRep(before);
      toastError(e);
    }
  }

  function save(action, keys) {
    put({ action, keys }, { ...user, [action]: keys }, "Saved. Pi applies it on /reload or the next run.");
  }

  function reset(action) {
    const next = { ...user };
    delete next[action];
    put({ action, reset: true }, next, "Back to Pi's default.");
  }

  async function resetAll() {
    const ok = await askConfirm({
      title: "Reset every key?",
      message: "Every action this pane changed goes back to Pi's default. Keys PiCode does not know about stay in the file.",
      confirmLabel: "Reset all",
      danger: true,
    });
    if (!ok) return;
    setBusy(true);
    const next = {};
    for (const [id, keys] of Object.entries(user)) {
      if (actions.some((a) => a.id === id)) continue;
      next[id] = keys;
    }
    try {
      await put({ resetAll: true }, next, "Every changed key is back to Pi's default.");
    } finally { setBusy(false); }
  }

  if (err) {
    return (
      <section className="settings-section">
        <h3>Keyboard</h3>
        <p className="settings-desc">
          Can't read {rep?.file || "the key map"}.{" "}
          <button type="button" className="btn btn-ghost btn-sm" onClick={load}>Retry</button>
        </p>
      </section>
    );
  }
  if (!rep) {
    return (
      <section className="settings-section">
        <h3>Keyboard</h3>
        <p className="settings-desc">Loading the key map…</p>
        <div className="key-skel" aria-hidden="true" />
        <div className="key-skel" aria-hidden="true" />
        <div className="key-skel" aria-hidden="true" />
      </section>
    );
  }

  const shown = actions.filter((a) => {
    if (!matchKeys(a, user, q, platform)) return false;
    if (keyFilter && !effectiveKeys(a, user, platform).includes(keyFilter)) return false;
    if (facet === "changed") return isOverride(a, user);
    if (facet === "off") return effectiveKeys(a, user, platform).length === 0;
    if (facet === "shared") return effectiveKeys(a, user, platform).some((c) => (index.get(c) || []).length > 1);
    return true;
  });
  const groups = [];
  for (const a of shown) {
    const last = groups[groups.length - 1];
    if (!last || last.name !== a.group) groups.push({ name: a.group, rows: [a] });
    else last.rows.push(a);
  }
  const facets = [
    { id: "all", label: "All", count: counts.all },
    { id: "changed", label: "Changed", count: counts.changed },
    { id: "shared", label: "Shared", count: counts.shared },
    { id: "off", label: "Off", count: counts.off },
  ].filter((f) => f.id === "all" || f.count > 0);

  return (
    <section className="settings-section key-pane">
      <h3>Keyboard</h3>
      <p className="settings-desc">
        {/* The period rides inside the code box: that box has padding, so a
            period after </code> sits one padding-width from the path and reads
            as a stray mark (visual review, 2026-09-21). A path that is not
            there yet carries its own sentence instead. */}
        This machine. Writes{" "}
        {rep.exists
          ? <code>{rep.file}.</code>
          : <><code>{rep.file}</code> (not created yet).</>}{" "}
        {pickupLine(rep.cli || CLI.id)}
      </p>

      <div className="key-bar" data-align-row data-align-wrap>
        <form className="key-filter" noValidate onSubmit={(e) => e.preventDefault()}>
          <input
            id="keys-filter"
            type="search"
            aria-label="Filter actions by name, group or key"
            // Short on purpose: the pane is ~474px wide at a 1024 window, and a
            // longer hint is cut at that width (visual review, 2026-09-21). The
            // aria-label carries what the field actually matches.
            placeholder="Filter keys"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </form>
        <button
          type="button"
          className={"key-find" + (findKey ? " is-on" : "")}
          aria-pressed={findKey}
          onClick={() => setFindKey((v) => !v)}
          title="Filter to the actions that answer to a chord"
        >{findKey ? "Press a key…" : "Find by key"}</button>
        <div className="key-facets" role="radiogroup" aria-label="Filter by state">
          {facets.map((f) => (
            <button
              key={f.id}
              type="button"
              role="radio"
              aria-checked={facet === f.id}
              className="key-facet"
              onClick={() => setFacet(f.id)}
            >{f.label}<span className="key-facet-n">{f.count}</span></button>
          ))}
        </div>
        {/* Outside the radiogroup: this one is not a state you pick among the
            facets, it is the key filter saying which chord is on. */}
        {keyFilter ? (
          <button
            type="button"
            className="key-facet is-chord"
            aria-label={"Clear the key filter (" + formatChord(keyFilter) + ")"}
            onClick={() => setKeyFilter("")}
          >{formatChord(keyFilter)}<span className="key-facet-x" aria-hidden="true">×</span></button>
        ) : null}
        <span className="key-count" role="status">
          {shown.length < actions.length ? shown.length + " of " + actions.length + " actions" : actions.length + " actions"}
        </span>
        {counts.changed ? (
          <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={resetAll}>
            {busy ? "Resetting…" : "Reset all"}
          </button>
        ) : null}
      </div>

      {shown.length === 0 ? (
        <p className="settings-desc">
          {keyFilter ? "No action answers to " + formatChord(keyFilter) + ". " : q ? "No action matches “" + q + "”. " : "Nothing here. "}
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={() => { setQ(""); setKeyFilter(""); setFacet("all"); }}
          >Show all</button>
        </p>
      ) : groups.map((group) => (
        <div key={group.name} className="key-group">
          <h4 className="settings-h">{group.name}<span className="key-group-n">{group.rows.length}</span></h4>
          {group.rows.map((a) => {
            const keys = effectiveKeys(a, user, platform);
            const changed = isOverride(a, user);
            const waiting = listen === a.id;
            const reserved = keys.filter(isReservedChord);
            const others = [];
            for (const chord of keys) {
              for (const other of index.get(chord) || []) {
                if (other.id !== a.id && !others.some((o) => o.id === other.id)) others.push(other);
              }
            }
            const alternates = platformAlternates(a, platform);
            return (
              <div key={a.id} className={"key-row" + (changed ? " is-changed" : "")} data-key-action={a.id}>
                <div className="key-name">
                  <span className="key-label">{a.label}</span>
                  {alternates.length ? (
                    <p className="key-alt">
                      {alternates.map((al) => (
                        (PLATFORM_LABEL[al.platform] || al.platform) + ": " + (al.keys.length ? al.keys.map(formatChord).join(" or ") : "unbound")
                      )).join(" · ")}
                    </p>
                  ) : null}
                </div>
                <div className="key-keys" data-align-row data-align-wrap>
                  {keys.map((k) => (
                    <span key={k} className="key-chip">
                      <kbd>{formatChord(k)}</kbd>
                      <button
                        type="button"
                        className="key-x"
                        aria-label={"Remove " + formatChord(k)}
                        disabled={disabled || waiting}
                        onClick={() => save(a.id, keys.filter((x) => x !== k))}
                      >×</button>
                    </span>
                  ))}
                  {waiting ? <span className="key-chip is-listen">Press a key…</span> : null}
                  {!keys.length && !waiting ? (
                    <span className="key-none">{changed ? "Off" : "Unbound"}</span>
                  ) : null}
                </div>
                {/* The note sits BETWEEN the keycaps and the row's actions: the
                    grid's flexible column, so the space a row with no note
                    leaves is the only air in the row (owner's screenshot,
                    2026-09-21). */}
                {waiting ? (
                  <p className="key-note is-live" aria-live="polite">
                    Press the chord to add. Esc cancels · a plain letter needs Ctrl, Alt or Cmd.
                  </p>
                ) : null}
                {!waiting && reserved.length ? (
                  <p className="key-note is-warn">
                    The browser keeps {reserved.map(formatChord).join(" and ")} — Pi never receives {reserved.length > 1 ? "them" : "it"} in a tab.
                  </p>
                ) : null}
                {!waiting && others.length ? (
                  <p className="key-note">
                    Also on {others.slice(0, 3).map((o) => o.label).join(", ")}{others.length > 3 ? " and " + (others.length - 3) + " more" : ""}.{" "}
                    <button
                      type="button"
                      className="key-show"
                      aria-label={"Show every action that shares a key with " + a.label}
                      onClick={() => setKeyFilter(keys.find((k) => (index.get(k) || []).length > 1) || "")}
                    >Show</button>
                  </p>
                ) : null}
                {/* Reset first, Add key last: the column is right-aligned, so
                    the action every row has keeps the slot the reader has
                    learned. With Add first, the rows that also offer Reset
                    shifted it ~58px left (visual review, 2026-09-21). */}
                <div className="key-actions" data-align-row>
                  {changed && !waiting ? (
                    <button type="button" className="key-act" disabled={disabled} onClick={() => reset(a.id)}>Reset</button>
                  ) : null}
                  {waiting ? (
                    <button type="button" className="key-act" onClick={() => setListen("")}>Cancel</button>
                  ) : (
                    <button type="button" className="key-act" disabled={disabled} onClick={() => setListen(a.id)}>Add key</button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      ))}
    </section>
  );
}
