import { useState } from "react";
import { appTile } from "../lib/nativeApps.js";
import { visibleApps } from "../lib/appsGrid.js";
import AppIcon from "./AppIcon.jsx";
import { IconSearch } from "./Icons.jsx";

// The Apps sidebar tab (ADR-0036): a phone-style grid of app tiles drawn
// entirely from manifests — no app code runs until a tile is opened.
// nativeApps is this shell's registry of native surfaces (ADR-0109): a
// native app it did not compile in stays on the grid, dimmed and honest.
// Header and search follow the Pins pattern (same classes as the other
// sidebar tabs); the list sorts alphabetically and filters as you type.
// Apps are first-party built-ins, so the header carries no add action.
export default function AppsGrid({ apps, nativeApps, onOpen }) {
  const [q, setQ] = useState("");
  const shown = visibleApps(apps || [], q);
  const searching = !!q.trim();
  return (
    <div className="side-section">
      <div className="pins-head">
        <span className="pins-title">Apps</span>
      </div>
      <label className="pins-search">
        <IconSearch />
        <input
          type="search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search apps"
          aria-label="Search apps"
          onKeyDown={(e) => { if (e.key === "Escape") setQ(""); }}
        />
      </label>
      <div className="side-scroll">
      {shown.length === 0 ? (
        <p className="side-empty pins-empty">
          {searching
            ? "No apps match."
            : "No apps yet. Apps extend PiCode with new surfaces; the first ones arrive in a coming release."}
        </p>
      ) : (
        <div className="app-grid" role="list">
          {shown.map((a) => {
            const tile = appTile(a, nativeApps);
            const ok = tile.ok;
            return (
              <button
                key={a.id}
                type="button"
                role="listitem"
                className={"app-tile" + (ok ? "" : " app-tile-unsupported")}
                title={tile.title}
                onClick={() => { if (ok && onOpen) onOpen(a.id); }}
              >
                <span className="app-tile-face">
                  <AppIcon name={a.icon} label={a.name} size={20} />
                  {a.badge.count > 0 ? <span className="app-tile-badge">{a.badge.count > 99 ? "99+" : a.badge.count}</span> : a.badge.dot ? <span className="app-tile-dot" /> : null}
                </span>
                <span className="app-tile-name">{a.name}</span>
              </button>
            );
          })}
        </div>
      )}
      </div>
    </div>
  );
}
