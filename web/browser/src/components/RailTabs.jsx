import { parseRoute } from "../lib/routes.js";
import { aggregateBadge } from "@picode/shared/contracts/appPrimitives.js";
import { IconFolders, IconAgent, IconTerminal, IconGrid, IconPin, IconCli } from "./Icons.jsx";

// The sidebar rail's view tabs — the same nav the sidebar brand row renders
// in the browser. The shell's merged top row renders this too, sized by the
// --shell-nav-w variable the sidebar publishes, so both stay in lockstep
// (ADR-0122). `tight` compresses the cells when the sidebar is narrow.
export default function RailTabs({ tab, selectTab, apps, pkgUpdates, tight = false, onOpenClis }) {
  const appBadge = aggregateBadge(apps);
  return (
    <nav className={"brand-tabs" + (tight ? " brand-tabs-tight" : "")} aria-label="Sidebar">
      <div className="brand-tablist" role="tablist" aria-label="Sidebar views">
        <button type="button" role="tab" className="brand-tab" aria-selected={tab === "workspaces"} title="Workspaces" aria-label="Workspaces" onClick={() => selectTab("workspaces")}><IconFolders size={16} /></button>
        <button type="button" role="tab" className="brand-tab" aria-selected={tab === "agents"} title="Agents" aria-label="Agents" onClick={() => selectTab("agents")}><IconAgent size={16} /></button>
        <button type="button" role="tab" className="brand-tab" aria-selected={tab === "terms"} title="Terminals" aria-label="Terminals" onClick={() => selectTab("terms")}><IconTerminal size={16} /></button>
        <button type="button" role="tab" className="brand-tab" aria-selected={tab === "apps"} title="Apps" aria-label="Apps" onClick={() => selectTab("apps")}>
          <IconGrid size={16} />
          {appBadge.count > 0 ? <span className="brand-tab-badge">{appBadge.count > 99 ? "99+" : appBadge.count}</span> : appBadge.dot ? <span className="brand-tab-dot" /> : null}
        </button>
        <button type="button" role="tab" className="brand-tab" aria-selected={tab === "pins"} title="Pins" aria-label="Pins" onClick={() => selectTab("pins")}><IconPin size={16} /></button>
      </div>
      <button type="button" className="brand-tab brand-clis" title="Agent CLIs" aria-label="Agent CLIs" aria-current={parseRoute() === "clis" ? "page" : undefined} onClick={() => onOpenClis && onOpenClis()}>
        <IconCli size={16} />
        {pkgUpdates?.length ? <span className="brand-tab-dot" aria-label="Package updates available" /> : null}
      </button>
    </nav>
  );
}
