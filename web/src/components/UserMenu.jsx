import { IconUser, IconChevronUp, IconSun, IconMonitor, IconMoon, IconPhone, IconChevronRight, IconExternal } from "./Icons.jsx";
import { readShellPref, setShell } from "../lib/shell.js";
import InstallButton from "./InstallButton.jsx";

export default function UserMenu({
  open, onToggle, onClose, host, version, themeMode, onTheme, onNavigate,
}) {
  return (
    <div className="usermenu" id="usermenu">
      <button
        className="um-trigger"
        id="um-trigger"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={(e) => { e.stopPropagation(); onToggle(); }}
      >
        <span className="um-avatar" aria-hidden="true"><IconUser /></span>
        <span className="um-meta">
          <span className="um-name" id="um-name">{host}</span>
          <span className="um-sub" id="um-sub">this machine</span>
        </span>
        <IconChevronUp />
      </button>

      <div className="um-popover" id="um-popover" hidden={!open} role="menu" aria-label="User menu">
        <div className="um-account">
          <span className="um-avatar" aria-hidden="true"><IconUser /></span>
          <div className="um-account-meta">
            <span className="um-account-name" id="um-name2">{host}</span>
            <span className="um-account-sub">PiCode on this machine</span>
          </div>
        </div>

        <div className="um-divider" />
        <div className="um-label">Theme</div>
        <div className="um-theme" role="group" aria-label="Theme">
          <button type="button" data-theme-option="light" data-active={themeMode === "light" ? "1" : ""} onClick={() => { onTheme("light"); onClose(); }}>
            <IconSun /> Light
          </button>
          <button type="button" data-theme-option="system" data-active={themeMode === "system" ? "1" : ""} onClick={() => { onTheme("system"); onClose(); }}>
            <IconMonitor /> System
          </button>
          <button type="button" data-theme-option="dark" data-active={themeMode === "dark" ? "1" : ""} onClick={() => { onTheme("dark"); onClose(); }}>
            <IconMoon /> Dark
          </button>
        </div>

        <div className="um-label">Layout</div>
        <div className="um-theme" role="group" aria-label="Layout">
          <button type="button" data-active={readShellPref() === "desktop" ? "1" : ""} onClick={() => setShell("desktop")}>
            <IconMonitor /> Desktop
          </button>
          <button type="button" data-active={readShellPref() === "system" ? "1" : ""} onClick={() => setShell("system")}>
            <IconMonitor /> Auto
          </button>
          <button type="button" data-active={readShellPref() === "mobile" ? "1" : ""} onClick={() => setShell("mobile")}>
            <IconPhone /> Mobile
          </button>
        </div>

        <div className="um-divider" />
        <button type="button" className="um-item" id="um-settings" role="menuitem" onClick={() => onNavigate("settings")}>
          <span>Settings</span>
          <IconChevronRight />
        </button>
        <button type="button" className="um-item" id="um-system" role="menuitem" onClick={() => onNavigate("system")}>
          <span>System</span>
          <IconChevronRight />
        </button>
        <button type="button" className="um-item" id="um-providers" role="menuitem" onClick={() => onNavigate("providers")}>
          <span>Providers</span>
          <IconChevronRight />
        </button>
        <button type="button" className="um-item" id="um-mcps" role="menuitem" onClick={() => onNavigate("mcps")}>
          <span>MCPs</span>
          <IconChevronRight />
        </button>
        <button type="button" className="um-item" id="um-devices" role="menuitem" onClick={() => onNavigate("devices")}>
          <span>Devices</span>
          <IconChevronRight />
        </button>
        <div style={{ padding: "8px 10px 10px" }}><InstallButton className="btn btn-primary btn-sm" /></div>
        <a className="um-item" id="um-docs" href="https://github.com/cfpperche/picode#readme" target="_blank" rel="noopener noreferrer" role="menuitem">
          <span>Documentation</span>
          <IconExternal />
        </a>

        <div className="um-divider" />
        <div className="um-version">PiCode <span id="um-ver">{version ? "v" + version : ""}</span></div>
      </div>
    </div>
  );
}
