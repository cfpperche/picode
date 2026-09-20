import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { rankVisits, splitAddress } from "../lib/addressHistory.js";
import { IconEnter } from "./Icons.jsx";

// The address bar, with the pages you have already been to under it.
//
// Both panes render this: the desktop one (`onOpen` drives the real WebView2)
// and the one with no desktop shell, which can only frame this machine's
// servers and so passes `localOnly` — the same list, filtered to what that
// pane can open. One component, because two address bars drifted apart once
// already (the wrapper remembered one placeholder, the shell another).
//
// The dropdown is a floating layer in the shared vocabulary
// (`role="listbox"` + `data-state`): a WebView2 paints over any HTML in its
// region, so the app parks the native view while this list is open — the same
// rule the ⋮ menu follows. Lose the role or the state and the list becomes
// invisible over a live page.
export default function WebTabAddress({ value, onChange, onOpen, inputRef, localOnly = false, ariaLabel = "Address" }) {
  // The parent keeps a ref to the field: it refreshes the draft from the
  // page's own location only while the field is NOT focused, and only the
  // element can answer that.
  const ownRef = useRef(null);
  const fieldRef = inputRef || ownRef;
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const [visits, setVisits] = useState([]);
  const [loaded, setLoaded] = useState(false);

  const draft = value || "";
  const rows = useMemo(
    () => rankVisits(visits, { query: draft, skipUrl: draft, localOnly }),
    [visits, draft, localOnly],
  );

  // Fetch on open and on every keystroke, debounced while typing. The list is
  // disposable: a failed read shows the same honest line as an empty history,
  // and the next open retries.
  useEffect(() => {
    if (!open) return undefined;
    const q = draft.trim();
    const t = setTimeout(() => {
      fetch(`/api/browser/history?limit=100${q ? "&q=" + encodeURIComponent(q) : ""}`)
        .then((r) => (r.ok ? r.json() : null))
        .then((d) => {
          setVisits(d && Array.isArray(d.visits) ? d.visits : []);
          setLoaded(true);
        })
        .catch(() => {
          setVisits([]);
          setLoaded(true);
        });
    }, q ? 140 : 0);
    return () => clearTimeout(t);
  }, [open, draft]);

  useEffect(() => {
    setActive(rows.length ? 0 : -1);
  }, [rows.length, draft]);

  const close = useCallback(() => {
    setOpen(false);
    setActive(-1);
  }, []);

  const pick = useCallback(
    (url) => {
      close();
      onOpen(url);
    },
    [close, onOpen],
  );

  const onKeyDown = (e) => {
    if (e.key === "Escape") {
      if (open) {
        e.preventDefault();
        e.stopPropagation();
        close();
      }
      return;
    }
    if (e.key === "Enter") {
      e.preventDefault();
      if (open && active >= 0 && rows[active]) pick(rows[active].url);
      else onOpen(draft);
      return;
    }
    if (e.key === "Tab") {
      close();
      return;
    }
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if (!open) {
        setOpen(true);
        return;
      }
      if (!rows.length) return;
      setActive((cur) => {
        const step = e.key === "ArrowDown" ? 1 : -1;
        return (cur + step + rows.length) % rows.length;
      });
    }
  };

  return (
    <div className="web-tab-address">
      <div className="web-tab-urlbar">
        <input
          ref={fieldRef}
          value={draft}
          onChange={(e) => {
            onChange(e.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onBlur={close}
          onKeyDown={onKeyDown}
          placeholder="Enter a URL"
          spellCheck={false}
          aria-label={ariaLabel}
          aria-autocomplete="list"
          aria-expanded={open}
        />
        <button type="button" className="web-tab-go" title="Open (Enter)" aria-label="Open" onClick={() => pick(draft)}>
          <IconEnter />
        </button>
      </div>
      <ul
        className="web-tab-suggest"
        role="listbox"
        data-state={open ? "open" : "closed"}
        aria-label="Pages you have opened"
        hidden={!open}
      >
        {rows.length === 0 ? (
          <li className="web-tab-suggest-empty" role="presentation">
            {loaded ? (draft.trim() ? "No match in your history." : "No pages here yet — open one and it shows up.") : "Reading your history…"}
          </li>
        ) : (
          rows.map((v, i) => {
            const { host, rest } = splitAddress(v.url);
            return (
              <li
                key={v.url}
                role="option"
                aria-selected={i === active}
                className={"web-tab-suggest-row" + (i === active ? " on" : "")}
                title={v.url}
                onMouseDown={(e) => e.preventDefault()}
                onMouseEnter={() => setActive(i)}
                onClick={() => pick(v.url)}
              >
                <span className="host">{host}</span>
                {rest ? <span className="rest">{rest}</span> : null}
                {v.title ? <span className="title">{v.title}</span> : null}
              </li>
            );
          })
        )}
      </ul>
    </div>
  );
}
