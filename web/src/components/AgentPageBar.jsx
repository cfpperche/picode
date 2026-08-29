import { IconSettings, IconMcp, IconPackage } from "./Icons.jsx";

const PAGES = [
  { id: "settings", label: "Settings", Icon: IconSettings },
  { id: "mcps", label: "MCPs", Icon: IconMcp },
  { id: "packages", label: "Packages", Icon: IconPackage },
];

export default function AgentPageBar({ onGo, pkgUpdates }) {
  if (!onGo) return null;
  const hasPkgUp = !!(pkgUpdates && pkgUpdates.length);
  return (
    <div className="composer-pages" role="toolbar" aria-label="Agent">
      {PAGES.map((p) => (
        <button
          key={p.id}
          type="button"
          className="composer-page"
          aria-label={p.label}
          title={p.label}
          onClick={() => onGo(p.id)}
        >
          <p.Icon />
          {p.id === "packages" && hasPkgUp ? <span className="um-dot composer-page-dot" aria-label="Updates available" /> : null}
        </button>
      ))}
    </div>
  );
}
