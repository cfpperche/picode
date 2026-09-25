import { cliProvidersHash } from "@picode/shared/domain/cliProviders.js";
import { cliPackagesLocation, cliPackagesHash } from "@picode/shared/domain/cliPackages.js";
import { cliSkillsHash } from "@picode/shared/domain/cliSkills.js";
import { cliConnectorsLocation, cliConnectorsHash } from "@picode/shared/domain/integrations.js";
import { lazy, useState } from "react";
import { cliSettingsLocation, cliSettingsHash } from "@picode/shared/domain/cliSettings.js";
import ScreenHeader from "../components/ScreenHeader.jsx";
const Devices = lazy(() => import("../components/Devices.jsx"));
const Settings = lazy(() => import("../components/Settings.jsx"));
const AgentClis = lazy(() => import("../components/AgentClis.jsx"));
const Automations = lazy(() => import("../components/Automations.jsx"));
const System = lazy(() => import("../components/System.jsx"));
const LlamaPanel = lazy(() => import("../components/LlamaPanel.jsx"));
const Integrations = lazy(() => import("../components/Integrations.jsx"));
import InstallButton from "../components/InstallButton.jsx";
import PushPrefs from "../components/PushPrefs.jsx";
import AppsGrid from "../components/AppsGrid.jsx";
import { IconChevronRight, IconMonitor, IconQR, IconSparkles, IconPlus } from "../components/Icons.jsx";
import { setShell } from "@picode/shared/client/shell.js";
import { MORE_TITLES, moreGroups, moreActions, moreHasResults } from "../lib/moreMenuModel.js";
import PinsList from "./PinsList.jsx";
import SnippetsList from "./SnippetsList.jsx";
import OutcomesList from "./OutcomesList.jsx";
import HistoryList from "./HistoryList.jsx";
import "../styles/mobile-lists.css";

// Mobile-owned settings, loaded only when their section opens.
export default function More({ fleetReady = true, section, apps, catalog, clis = [], clisState = "ok", system, version, themeMode, onTheme, last, onRefreshCatalog, onCatalogChange, onShare, onWhatsNew, whatsNewUnread, onBack, onAgentConfig, workspaces = [], freeAgents = [], legacyAgentId = "" }) {
  // The CLI panes follow the last agent opened, as desktop follows the
  // selected one (ADR-0179). An agent with no cli is a Pi agent (agentIsPi);
  // with no agent yet there is no CLI to pick, so the catalog opens.
  const lastCli = last?.agent ? String(last.agent.cli || "").trim() || "pi" : "";
  const [query, setQuery] = useState("");
  if (!section) {
    const groups = moreGroups(query);
    const actions = moreActions(query).map(row => {
      if (row.id === "updates") return { ...row, Icon: IconSparkles, action: onWhatsNew, unread: whatsNewUnread };
      if (row.id === "pair") return { ...row, Icon: IconQR, action: onShare };
      return { ...row, Icon: IconMonitor, action: () => setShell("desktop") };
    });
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
              <a className="m-row-main" href={!lastCli && ["pi-providers", "pi-packages", "pi-skills", "pi-settings", "connectors"].includes(id) ? "#/clis" : id === "pi-providers" ? cliProvidersHash(lastCli) : id === "pi-packages" ? cliPackagesHash(lastCli, { workspaceId: last?.workspace?.id, agentId: last?.agent?.id || legacyAgentId }) : id === "pi-skills" ? cliSkillsHash(lastCli, { workspaceId: last?.workspace?.id, agentId: last?.agent?.id || legacyAgentId }) : id === "pi-settings" ? cliSettingsHash(lastCli, { workspaceId: last?.workspace?.id, agentId: last?.agent?.id || legacyAgentId }) : id === "connectors" ? cliConnectorsHash(lastCli, { workspaceId: last?.workspace?.id, agentId: last?.agent?.id || legacyAgentId }) : id === "integrations" ? "#/integrations/webhooks" : "#/more/" + id}>
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
        {!moreHasResults(query) ? <div className="m-list-empty" role="status"><p>No matching tools or settings.</p><button type="button" className="btn btn-sm" onClick={() => setQuery("")}>Clear search</button></div> : null}
        {!query.trim() ? <><div className="m-install"><InstallButton /></div>{version ? <p className="m-version">PiCode {version}</p> : null}</> : null}
      </div>
    );
  }
  const agent = last && last.agent;
  const workspace = last && last.workspace;
  return (
    <div className="m-screen m-more-page">
      <ScreenHeader title={MORE_TITLES[section] || "More"} onBack={section === "clis" && (cliSettingsLocation(location.hash) || cliPackagesLocation(location.hash) || cliConnectorsLocation(location.hash)) ? () => { location.hash = "#/clis"; } : onBack}
        right={section === "pins" || section === "snippets" ? <button type="button" className="m-head-btn" aria-label={section === "pins" ? "New pin" : "New snippet"} onClick={() => { location.hash = section === "pins" ? "#/pins/new" : "#/snippets/new"; }}><IconPlus size={18} /></button> : null} />
      {section === "pins" ? <PinsList onOpen={(id) => { location.hash = "#/pins/" + encodeURIComponent(id); }} onNew={() => { location.hash = "#/pins/new"; }} /> : null}
      {section === "snippets" ? <SnippetsList onOpen={(id) => { location.hash = "#/snippets/" + encodeURIComponent(id); }} onNew={() => { location.hash = "#/snippets/new"; }} /> : null}
      {section === "outcomes" ? <OutcomesList /> : null}
      {section === "history" ? <HistoryList workspaces={workspaces} /> : null}
      {section === "apps" ? <AppsGrid apps={apps} onOpen={(id) => { location.hash = "#/app/" + encodeURIComponent(id); }} /> : null}
      {section === "devices" ? <Devices hidden={false} /> : null}
      {section === "clis" ? <AgentClis catalog={catalog} onCatalogChange={onCatalogChange} legacyContextReady={fleetReady} legacyPackageContext={{ workspaceId: workspace?.id || "", agentId: agent?.id || legacyAgentId || "" }} legacyAgentId={last?.agent?.id || legacyAgentId} onAgentConfig={onAgentConfig} /> : null}
      {section === "automations" ? <Automations hidden={false} catalog={catalog} clis={clis} clisLoaded={clisState === "ok"} system={system} workspaces={workspaces} freeAgents={freeAgents} /> : null}
      {section === "preferences" ? <Settings hidden={false} themeMode={themeMode} onTheme={onTheme} workspaces={workspaces} workspacesLoaded={fleetReady} /> : null}
      {section === "system" ? <System hidden={false} version={version} system={system} clis={clis} clisState={clisState} /> : null}
      {section === "llama" ? <LlamaPanel onRefresh={onRefreshCatalog} /> : null}
      {section === "integrations" ? <Integrations hidden={false} /> : null}
      {section === "notifications" ? <section className="settings-wrap"><div className="settings-card"><PushPrefs /></div></section> : null}
    </div>
  );
}
