import { cliProvidersLocation, cliProvidersHash } from "@picode/shared/domain/cliProviders.js";
import { cliPackagesLocation, cliPackagesHash } from "@picode/shared/domain/cliPackages.js";
import { lazy, useState } from "react";
import { cliSettingsLocation, cliSettingsHash } from "@picode/shared/domain/cliSettings.js";
import ScreenHeader from "../components/ScreenHeader.jsx";
const Devices = lazy(() => import("../components/Devices.jsx"));
const Settings = lazy(() => import("../components/Settings.jsx"));
const AgentClis = lazy(() => import("../components/AgentClis.jsx"));
const Automations = lazy(() => import("../components/Automations.jsx"));
const System = lazy(() => import("../components/System.jsx"));
const LlamaPanel = lazy(() => import("../components/LlamaPanel.jsx"));
const Mcps = lazy(() => import("../components/Mcps.jsx"));
const Integrations = lazy(() => import("../components/Integrations.jsx"));
import InstallButton from "../components/InstallButton.jsx";
import PushPrefs from "../components/PushPrefs.jsx";
import AppsGrid from "../components/AppsGrid.jsx";
import { IconChevronRight, IconMonitor, IconQR, IconSparkles, IconPlus } from "../components/Icons.jsx";
import { setShell } from "@picode/shared/client/shell.js";
import { matchesListSearch } from "../lib/mobileListSearch.js";
import PinsList from "./PinsList.jsx";
import "../styles/mobile-lists.css";

const SECTIONS = [
  ["pins", "Pins", "Notes, files and reminders"],
  ["automations", "Automations", "Scheduled and triggered work"],
  ["clis", "Agent CLIs", "Launches, sessions and CLI configuration"],
  ["apps", "Apps", "Docker and other tools"],
  ["notifications", "Notifications", "Push when an agent needs you"],
  ["llama", "llama.cpp", "Models and server connection"],
  ["providers", "Providers", "Accounts, keys, usage"],
  ["preferences", "Preferences", "Theme, notifications, backup"],
  ["integrations", "Integrations", "Connectors and event delivery"],
  ["packages", "Packages", "Skills, extensions, updates"],
  ["devices", "Devices", "Who is connected"],
  ["system", "System", "Version, host, paths"],
];

const TITLES = { ...Object.fromEntries(SECTIONS.map(([id, t]) => [id, t])), mcps: "MCP servers" };
const GROUPS = [
  ["Tools", ["pins", "clis", "automations", "apps", "llama"]],
  ["Agents and connections", ["integrations"]],
  ["PiCode", ["preferences", "notifications", "devices", "system"]],
];

// Mobile-owned settings, loaded only when their section opens.
export default function More({ fleetReady = true, section, apps, catalog, system, version, themeMode, onTheme, last, onRefreshCatalog, onCatalogChange, onShare, onWhatsNew, whatsNewUnread, onBack, onAgentConfig, workspaces = [], freeAgents = [], legacyAgentId = "" }) {
  const [query, setQuery] = useState("");
  if (!section) {
    const groups = GROUPS.map(([title, ids]) => ({ title, rows: ids.map(id => SECTIONS.find(row => row[0] === id)).filter(row => matchesListSearch(query, title, ...row)) })).filter(group => group.rows.length);
    if (query.trim() && matchesListSearch(query, "Pi settings", "model thinking prompt")) groups.unshift({ title: "Agent CLIs", rows: [["pi-settings", "Pi settings", "Model, thinking, tools and keys"]] });
    if (query.trim() && matchesListSearch(query, "Packages", "skills extensions updates")) groups.unshift({ title: "Agent CLIs", rows: [["pi-packages", "Packages", "Pi skills, extensions and updates"]] });
    if (query.trim() && matchesListSearch(query, "Providers", "accounts keys usage login")) groups.unshift({ title: "Agent CLIs", rows: [["pi-providers", "Providers", "Pi accounts, keys and usage"]] });
    const actions = [
      { id: "updates", title: "What’s new", sub: "Release highlights", Icon: IconSparkles, action: onWhatsNew, unread: whatsNewUnread },
      { id: "pair", title: "Open on another phone", sub: "Pair with a QR code", Icon: IconQR, action: onShare },
      { id: "desktop", title: "Desktop layout", sub: "Open the desktop workspace", Icon: IconMonitor, action: () => setShell("desktop") },
    ].filter(row => matchesListSearch(query, row.title, row.sub, row.id));
    return (
      <div className="m-screen m-v2-lists m-more-v2" aria-label="More">
        <div className="m-screen-head m-list-head">
          <input type="search" className="dlg-input m-list-search" aria-label="Search tools and settings" placeholder="Search tools and settings" value={query} onChange={event => setQuery(event.target.value)} />
        </div>
        {groups.map(group => <section className="m-section m-more-group" key={group.title} aria-label={group.title}>
          <h2 className="m-section-label">{group.title}</h2>
          <ul className="m-list m-menu m-group-list">
          {group.rows.map(([id, title, sub]) => (
            <li key={id} className="m-row">
              <a className="m-row-main" href={id === "pi-providers" ? cliProvidersHash() : id === "pi-packages" ? cliPackagesHash("pi", { agentId: last?.agent?.id || legacyAgentId }) : id === "pi-settings" ? cliSettingsHash("pi", { agentId: last?.agent?.id || legacyAgentId }) : "#/more/" + id}>
                <span className="m-row-text">
                  <span className="m-row-title">{title}</span>
                  <span className="m-row-sub">{sub}</span>
                </span>
                <IconChevronRight size={16} className="m-row-chev" />
              </a>
            </li>
          ))}
          </ul>
        </section>)}
        {actions.length ? <section className="m-section m-more-group" aria-label="Continue">
          <h2 className="m-section-label">Continue</h2>
          <ul className="m-list m-menu m-group-list">{actions.map(({ id, title, sub, Icon, action, unread }) => <li key={id} className={"m-row" + (unread ? " is-unread" : "")}>
            <button type="button" className="m-row-main" onClick={action}>
              <span className="m-row-face"><Icon size={18} /></span>
              <span className="m-row-text"><span className="m-row-title">{title}</span><span className="m-row-sub">{sub}</span></span>
              <IconChevronRight size={16} className="m-row-chev" />
            </button>
          </li>)}</ul>
        </section> : null}
        {!groups.length && !actions.length ? <div className="m-list-empty" role="status"><p>No matching tools or settings.</p><button type="button" className="btn btn-sm" onClick={() => setQuery("")}>Clear search</button></div> : null}
        {!query.trim() ? <><div className="m-install"><InstallButton /></div>{version ? <p className="m-version">PiCode {version}</p> : null}</> : null}
      </div>
    );
  }
  const agent = last && last.agent;
  const workspace = last && last.workspace;
  const agentName = agent ? (agent.name && agent.name !== "default" ? agent.name : (workspace ? workspace.name : "")) : "";
  return (
    <div className="m-screen m-more-page">
      <ScreenHeader title={TITLES[section] || "More"} onBack={section === "clis" && (cliSettingsLocation(location.hash) || cliPackagesLocation(location.hash) || cliProvidersLocation(location.hash)) ? () => { location.hash = "#/clis"; } : onBack}
        right={section === "pins" ? <button type="button" className="m-head-btn" aria-label="New pin" onClick={() => { location.hash = "#/pins/new"; }}><IconPlus size={18} /></button> : null} />
      {section === "pins" ? <PinsList onOpen={(id) => { location.hash = "#/pins/" + encodeURIComponent(id); }} onNew={() => { location.hash = "#/pins/new"; }} /> : null}
      {section === "apps" ? <AppsGrid apps={apps} onOpen={(id) => { location.hash = "#/app/" + encodeURIComponent(id); }} /> : null}
      {section === "devices" ? <Devices hidden={false} /> : null}
      {section === "clis" ? <AgentClis catalog={catalog} onCatalogChange={onCatalogChange} legacyContextReady={fleetReady} legacyPackageContext={{ workspaceId: workspace?.id || "", agentId: agent?.id || legacyAgentId || "" }} legacyAgentId={last?.agent?.id || legacyAgentId} onAgentConfig={onAgentConfig} /> : null}
      {section === "automations" ? <Automations hidden={false} catalog={catalog} system={system} workspaces={workspaces} freeAgents={freeAgents} /> : null}
      {section === "preferences" ? <Settings hidden={false} themeMode={themeMode} onTheme={onTheme} /> : null}
      {section === "system" ? <System hidden={false} version={version} system={system} /> : null}
      {section === "llama" ? <LlamaPanel onRefresh={onRefreshCatalog} /> : null}
      {section === "integrations" ? <Integrations hidden={false}
        workspaceId={workspace?.id || ""} workspaceName={workspace?.name || ""} workspacePath={workspace?.path || ""}
        agentId={agent?.id || ""} agentName={agentName} agentWorkPath={agent?.workPath || ""} agentRunning={!!(agent && agent.mode && agent.mode !== "stopped")} /> : null}
      {section === "mcps" ? (
        <Mcps hidden={false} workspaceId={workspace ? workspace.id : ""} workspaceName={workspace ? workspace.name : ""} workspacePath={workspace ? workspace.path : ""}
          agentId={agent ? agent.id : ""} agentName={agentName} agentWorkPath={agent ? agent.workPath || "" : ""} agentRunning={!!(agent && agent.mode && agent.mode !== "stopped")} />
      ) : null}
      {section === "notifications" ? <section className="settings-wrap"><div className="settings-card"><PushPrefs /></div></section> : null}
    </div>
  );
}
