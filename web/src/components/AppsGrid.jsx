import { supportedApp } from "../lib/appPrimitives.js";
import AppIcon from "./AppIcon.jsx";

// The Apps sidebar tab (ADR-0036): a phone-style grid of app tiles drawn
// entirely from manifests — no app code runs until a tile is opened.
export default function AppsGrid({ apps, onOpen }) {
  const list = apps || [];
  return (
    <div className="side-section">
      {list.length === 0 ? (
        <p className="side-empty pins-empty">No apps yet. Apps extend PiCode with new surfaces; the first ones arrive in a coming release.</p>
      ) : (
        <div className="app-grid" role="list">
          {list.map((a) => {
            const ok = supportedApp(a);
            return (
              <button
                key={a.id}
                type="button"
                role="listitem"
                className={"app-tile" + (ok ? "" : " app-tile-unsupported")}
                title={ok ? a.name : a.name + " needs a newer PiCode (app speaks v" + a.apiVersion + ")"}
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
