import { useEffect, useRef } from "react";
import { IconPlus, IconX } from "./Icons.jsx";

const MAX_CHECKS = 8;

// The rules editor of the integration declaration (ADR-0182), shared by a
// workspace's Settings dialog and Preferences → Landing work: fast-forward,
// its blocked warning (the runner lands fast-forward only), and up to eight
// check commands. `checksHelp` is the one line that differs between the two.
export default function LandingRulesFields({ draft, setDraft, idPrefix, checksHelp }) {
  const focusLast = useRef(false);
  const listRef = useRef(null);
  const checks = draft.checks;

  // A new check row takes the cursor, so Add check → type is one motion.
  useEffect(() => {
    if (!focusLast.current || !listRef.current) return;
    focusLast.current = false;
    const inputs = listRef.current.querySelectorAll("input");
    inputs[inputs.length - 1]?.focus();
  }, [checks.length]);

  function setCheck(i, v) {
    setDraft((d) => ({ ...d, checks: d.checks.map((c, j) => (j === i ? v : c)) }));
  }

  return (
    <div className="wsset-own">
      <label className="dlg-choice wsset-choice">
        <input type="checkbox" checked={draft.ffOnly} aria-describedby={!draft.ffOnly ? idPrefix + "-ff-warn" : undefined} onChange={(e) => setDraft((d) => ({ ...d, ffOnly: e.target.checked }))} />
        <span>Only land a branch that is up to date with the target <span className="wsset-muted">(fast-forward)</span></span>
      </label>
      {!draft.ffOnly ? <p id={idPrefix + "-ff-warn"} className="wsset-effect is-blocked">PiCode lands fast-forward only: with this off, authorized branches stay blocked.</p> : null}
      <div className="wsset-checks">
        <span className="wsset-sublabel">Checks that must pass first</span>
        {checksHelp ? <p className="wsset-help wsset-checks-help">{checksHelp}</p> : null}
        {checks.length === 0 ? (
          <p className="wsset-empty">No checks: an authorized branch lands right away.</p>
        ) : (
          <ol className="wsset-check-list" ref={listRef}>
            {checks.map((c, i) => (
              <li key={i} className="wsset-check">
                <input
                  className="dlg-input wsset-check-input"
                  value={c}
                  autoComplete="off"
                  spellCheck={false}
                  placeholder={i === 0 ? "e.g. make ci" : "Another command"}
                  aria-label={"Check " + (i + 1)}
                  onChange={(e) => setCheck(i, e.target.value)}
                />
                <button type="button" className="ws-icon-btn wsset-check-remove" aria-label={"Remove check " + (i + 1)} title="Remove" onClick={() => setDraft((d) => ({ ...d, checks: d.checks.filter((_, j) => j !== i) }))}>
                  <IconX size={13} />
                </button>
              </li>
            ))}
          </ol>
        )}
        {checks.length < MAX_CHECKS ? (
          <button type="button" className="btn btn-ghost btn-sm wsset-add" onClick={() => { focusLast.current = true; setDraft((d) => ({ ...d, checks: [...d.checks, ""] })); }}>
            <IconPlus size={13} /> Add check
          </button>
        ) : null}
      </div>
    </div>
  );
}
