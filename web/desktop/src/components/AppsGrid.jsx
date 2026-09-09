import { appTile } from "../lib/nativeApps.js";
import AppIcon from "./AppIcon.jsx";

// The Apps sidebar tab (ADR-0036): a phone-style grid of app tiles drawn
// entirely from manifests — no app code runs until a tile is opened.
// nativeApps is this shell's registry of native surfaces (ADR-0109): a
// native app it did not compile in stays on the grid, dimmed and honest.
export default function AppsGrid({ apps, nativeApps, onOpen }) {
  const list = apps || [];
  return (
    <div className="side-section">
      {list.length === 0 ? (
        <p className="side-empty pins-empty">No apps yet. Apps extend PiCode with new surfaces; the first ones arrive in a coming release.</p>
      ) : (
        <div className="app-grid" role="list">
          {list.map((a) => {
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
  );
}
