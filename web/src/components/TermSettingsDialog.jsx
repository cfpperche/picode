import { useCallback, useEffect, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { api } from "../lib/api.js";
import { toastError } from "../lib/toast.js";
import { INHERIT, selectedKey, choicesFor, effectText, isOverridden, withChoice } from "../lib/termSettings.js";

// Terminal behaviour, global or for one terminal (ADR-0024). `scope` is null
// for the defaults every terminal inherits, or { id, name } for one terminal.
//
// Behaviour only. Font, colours and cursor live in Preferences because they
// belong to the browser looking at the terminal, while these belong to the
// terminal itself and are the same on every device that opens it.
export default function TermSettingsDialog({ open, scope, onClose }) {
  const isGlobal = !scope;
  const path = isGlobal ? "/api/terminals/settings" : `/api/terminals/${scope.id}/settings`;
  const [data, setData] = useState(null);

  useEffect(() => {
    if (!open) {
      setData(null);
      return;
    }
    let alive = true;
    api(path)
      .then((d) => { if (alive) setData(d); })
      .catch((e) => { if (alive) { toastError(e); onClose(); } });
    return () => { alive = false; };
  }, [open, path, onClose]);

  const pick = useCallback((flagKey, value) => {
    setData((prev) => {
      if (!prev) return prev;
      // Move the selection now and reconcile when the server answers. A
      // round trip is short, but not short enough that a segmented control
      // should sit on the old value while it happens.
      const optimistic = { ...prev, values: withChoice(prev.values, flagKey, value) };
      api(path, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        // null clears the field, which is what makes it inherit again.
        body: JSON.stringify({ [flagKey]: value }),
      })
        .then((fresh) => setData(fresh))
        .catch((e) => { toastError(e); setData(prev); });
      return optimistic;
    });
  }, [path]);

  const title = isGlobal ? "Terminal defaults" : scope.name;
  const subtitle = isGlobal
    ? "What every terminal starts with. A terminal that sets its own keeps it; the rest follow along."
    : "Only what this terminal changes. Everything else follows the defaults as they change.";

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-termset" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">{subtitle}</Dialog.Description>

          {!data ? <TermSettingsSkeleton /> : (
            <div className="termset-fields">
              {(data.flags || []).map((flag) => (
                <FlagField
                  key={flag.key}
                  flag={flag}
                  values={data.values}
                  inherited={data.inherited}
                  isGlobal={isGlobal}
                  onPick={pick}
                />
              ))}
              {(data.flags || []).length === 0 ? (
                <p className="termset-help">Nothing to change here yet.</p>
              ) : null}
            </div>
          )}

          <p className="termset-foot">
            Font, colours and cursor are in <a href="#/preferences/terminal">Preferences</a>.
            Those are remembered by this browser; the settings here travel with the
            terminal, so every device that opens it sees the same.
          </p>

          <div className="dlg-actions">
            <button type="button" className="btn btn-primary btn-sm" onClick={onClose}>Close</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

// Native radio inputs, styled as one control. The browser gives arrow-key
// movement, roving focus and the right announcement for free — a homemade
// widget would have to reimplement all three to stand still.
function FlagField({ flag, values, inherited, isGlobal, onPick }) {
  const chosen = selectedKey(values, flag.key);
  const name = `termset-${flag.key}`;
  return (
    <div className="termset-field">
      <div className="termset-head">
        <span className="termset-label">{flag.label}</span>
        {!isGlobal && isOverridden(values, flag.key) ? (
          <span className="termset-chip">Custom</span>
        ) : null}
      </div>
      <div className="termset-seg">
        {choicesFor(flag, inherited, isGlobal).map((c) => {
          const id = `${name}-${c.key === INHERIT ? "inherit" : c.key}`;
          return (
            <label className="termset-seg-opt" key={id} htmlFor={id}>
              <input
                id={id}
                type="radio"
                name={name}
                checked={c.key === chosen}
                onChange={() => onPick(flag.key, c.key)}
              />
              <span className="termset-seg-face">{c.label}</span>
            </label>
          );
        })}
      </div>
      <p className="termset-help">{flag.help}</p>
      <p className="termset-effect">{effectText(flag.effect)}</p>
    </div>
  );
}

// The shape of one field, so the panel does not jump when the answer lands.
function TermSettingsSkeleton() {
  return (
    <div className="termset-fields" aria-hidden="true">
      <div className="termset-field">
        <div className="skel-line w-40" />
        <div className="termset-seg termset-seg-skel" />
        <div className="skel-line w-90 termset-skel-help" />
        <div className="skel-line w-70 termset-skel-help" />
      </div>
    </div>
  );
}
