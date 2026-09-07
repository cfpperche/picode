import { supportedApp } from "@picode/shared/contracts/appPrimitives.js";
import AppIcon from "./AppIcon.jsx";
import { IconChevronRight } from "./Icons.jsx";
import "../styles/mobile-settings.css";

// The Apps sidebar tab (ADR-0036): a phone-style grid of app tiles drawn
// entirely from manifests — no app code runs until a tile is opened.
export default function AppsGrid({ apps, onOpen }) {
  const list = apps || [];
  return (
    <div className="side-section">
      {list.length === 0 ? (
        <div className="m-settings-empty"><p>No apps are available.</p><a className="btn btn-sm" href="#/more">Back to More</a></div>
      ) : (
        <div className="app-grid" role="list">
          {list.map((a) => {
            const ok = supportedApp(a);
            return (
              <div key={a.id} role="listitem"><button
                type="button"
                disabled={!ok}
                className={"app-tile" + (ok ? "" : " app-tile-unsupported")}
                title={ok ? a.name : a.name + " needs a newer PiCode (app speaks v" + a.apiVersion + ")"}
                onClick={() => { if (ok && onOpen) onOpen(a.id); }}
              >
                <span className="app-tile-face">
                  <AppIcon name={a.icon} label={a.name} size={20} />
                  {a.badge?.count > 0 ? <span className="app-tile-badge">{a.badge.count > 99 ? "99+" : a.badge.count}</span> : a.badge?.dot ? <span className="app-tile-dot" /> : null}
                </span>
                <span className="app-tile-copy"><span className="app-tile-name">{a.name}</span>{!ok ? <span className="app-tile-note">Update PiCode to open</span> : null}</span>
                {ok ? <IconChevronRight size={15} /> : null}
              </button></div>
            );
          })}
        </div>
      )}
    </div>
  );
}
