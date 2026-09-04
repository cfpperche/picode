import ScreenHeader from "../components/ScreenHeader.jsx";
import Devices from "../../components/Devices.jsx";
import Settings from "../../components/Settings.jsx";
import PiSettings from "../../components/PiSettings.jsx";
import System from "../../components/System.jsx";
import Providers from "../../components/Providers.jsx";
import Mcps from "../../components/Mcps.jsx";
import Packages from "../../components/Packages.jsx";
import InstallButton from "../../components/InstallButton.jsx";
import PushPrefs from "../../components/PushPrefs.jsx";
import AppsGrid from "../../components/AppsGrid.jsx";
import { IconChevronRight, IconMonitor, IconQR, IconSparkles } from "../../components/Icons.jsx";
import { setShell } from "../../lib/shell.js";

const SECTIONS = [
  ["apps", "Apps", "Docker and other tools"],
  ["notifications", "Notifications", "Push when an agent needs you"],
  ["providers", "Providers", "Accounts, keys, usage"],
  ["settings", "Settings", "Pi: model, thinking, prompt"],
  ["preferences", "Preferences", "Theme, notifications, backup"],
  ["mcps", "MCP", "Servers this agent can use"],
  ["packages", "Packages", "Skills, extensions, updates"],
  ["devices", "Devices", "Who is connected"],
  ["system", "System", "Version, host, paths"],
];

const TITLES = Object.fromEntries(SECTIONS.map(([id, t]) => [id, t]));

// The rest of the product, one tap deep. Sections mount the same page
// components the desktop routes do — with real props — under a mobile
// header; their own desktop "Back" link is hidden by mobile.css.
export default function More({ section, apps, catalog, system, version, themeMode, onTheme, last, onRefreshCatalog, onShare, onWhatsNew, whatsNewUnread, onBack }) {
  if (!section) {
    return (
      <div className="m-screen">
        <div className="m-screen-head"><h2 className="m-screen-title">More</h2></div>
        <ul className="m-list m-menu">
          {SECTIONS.map(([id, title, sub]) => (
            <li key={id} className="m-row">
              <a className="m-row-main" href={"#/more/" + id}>
                <span className="m-row-text">
                  <span className="m-row-title">{title}</span>
                  <span className="m-row-sub">{sub}</span>
                </span>
                <IconChevronRight size={16} className="m-row-chev" />
              </a>
            </li>
          ))}
          <li className={"m-row" + (whatsNewUnread ? " is-unread" : "")}>
            <button type="button" className="m-row-main" onClick={onWhatsNew}>
              <span className="m-row-face"><IconSparkles size={18} /></span>
              <span className="m-row-text"><span className="m-row-title">What’s new</span><span className="m-row-sub">Release highlights and improvements</span></span>
              <IconChevronRight size={16} className="m-row-chev" />
            </button>
          </li>
          <li className="m-row">
            <button type="button" className="m-row-main" onClick={onShare}>
              <span className="m-row-face"><IconQR size={18} /></span>
              <span className="m-row-text"><span className="m-row-title">Open on another phone</span><span className="m-row-sub">QR code for this server</span></span>
            </button>
          </li>
          <li className="m-row">
            <button type="button" className="m-row-main" onClick={() => setShell("desktop")}>
              <span className="m-row-face"><IconMonitor size={18} /></span>
              <span className="m-row-text"><span className="m-row-title">Desktop layout</span><span className="m-row-sub">The full workstation shell, on this screen</span></span>
            </button>
          </li>
        </ul>
        <div className="m-install"><InstallButton /></div>
        {version ? <p className="m-version">PiCode {version}</p> : null}
      </div>
    );
  }
  const agent = last && last.agent;
  const workspace = last && last.workspace;
  const agentName = agent ? (agent.name && agent.name !== "default" ? agent.name : (workspace ? workspace.name : "")) : "";
  return (
    <div className="m-screen m-more-page">
      <ScreenHeader title={TITLES[section] || "More"} onBack={onBack} />
      {section === "apps" ? <AppsGrid apps={apps} onOpen={(id) => { location.hash = "#/app/" + encodeURIComponent(id); }} /> : null}
      {section === "devices" ? <Devices hidden={false} /> : null}
      {section === "preferences" ? <Settings hidden={false} themeMode={themeMode} onTheme={onTheme} /> : null}
      {section === "settings" ? <PiSettings hidden={false} agent={agent} workspace={workspace} catalog={catalog} /> : null}
      {section === "system" ? <System hidden={false} version={version} system={system} /> : null}
      {section === "providers" ? <Providers hidden={false} catalog={catalog} onRefresh={onRefreshCatalog} /> : null}
      {section === "mcps" ? (
        <Mcps hidden={false} workspaceId={workspace ? workspace.id : ""} workspaceName={workspace ? workspace.name : ""} workspacePath={workspace ? workspace.path : ""}
          agentId={agent ? agent.id : ""} agentName={agentName} agentWorkPath={agent ? agent.workPath || "" : ""} agentRunning={!!(agent && agent.mode && agent.mode !== "stopped")} />
      ) : null}
      {section === "packages" ? (
        <Packages hidden={false} workspaceId={workspace ? workspace.id : ""} workspaceName={workspace ? workspace.name : ""} workspacePath={workspace ? workspace.path : ""}
          agentId={agent ? agent.id : ""} agentName={agentName} />
      ) : null}
      {section === "notifications" ? <section className="settings-wrap"><div className="settings-card"><PushPrefs /></div></section> : null}
    </div>
  );
}
