import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { IconUser, IconChevronUp, IconSun, IconMonitor, IconMoon, IconPhone, IconChevronRight, IconExternal, IconQR } from "./Icons.jsx";
import { readShellPref, setShell } from "../lib/shell.js";
import InstallButton from "./InstallButton.jsx";

export default function UserMenu({ host, version, themeMode, onTheme, onNavigate, onShare }) {
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button className="um-trigger" id="um-trigger">
          <span className="um-avatar" aria-hidden="true"><IconUser /></span>
          <span className="um-meta">
            <span className="um-name" id="um-name">{host}</span>
            <span className="um-sub" id="um-sub">this machine</span>
          </span>
          <IconChevronUp className="um-chev" />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className="um-popover" side="top" align="start" sideOffset={6} collisionPadding={8}>
          <div className="um-account">
            <span className="um-avatar" aria-hidden="true"><IconUser /></span>
            <div className="um-account-meta">
              <span className="um-account-name" id="um-name2">{host}</span>
              <span className="um-account-sub">PiCode on this machine</span>
            </div>
          </div>

          <DropdownMenu.Separator className="um-divider" />
          <div className="um-label">Theme</div>
          <div className="um-theme" role="group" aria-label="Theme">
            <button type="button" data-theme-option="light" data-active={themeMode === "light" ? "1" : ""} onClick={() => onTheme("light")}>
              <IconSun /> Light
            </button>
            <button type="button" data-theme-option="system" data-active={themeMode === "system" ? "1" : ""} onClick={() => onTheme("system")}>
              <IconMonitor /> System
            </button>
            <button type="button" data-theme-option="dark" data-active={themeMode === "dark" ? "1" : ""} onClick={() => onTheme("dark")}>
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

          <DropdownMenu.Separator className="um-divider" />
          <DropdownMenu.Item className="um-item" id="um-preferences" onSelect={() => onNavigate("preferences")}>
            <span>Preferences</span>
            <IconChevronRight />
          </DropdownMenu.Item>
          <DropdownMenu.Item className="um-item" id="um-settings" onSelect={() => onNavigate("settings")}>
            <span>Settings</span>
            <IconChevronRight />
          </DropdownMenu.Item>
          <DropdownMenu.Item className="um-item" id="um-system" onSelect={() => onNavigate("system")}>
            <span>System</span>
            <IconChevronRight />
          </DropdownMenu.Item>
          <DropdownMenu.Item className="um-item" id="um-providers" onSelect={() => onNavigate("providers")}>
            <span>Providers</span>
            <IconChevronRight />
          </DropdownMenu.Item>
          <DropdownMenu.Item className="um-item" id="um-mcps" onSelect={() => onNavigate("mcps")}>
            <span>MCPs</span>
            <IconChevronRight />
          </DropdownMenu.Item>
          <DropdownMenu.Item className="um-item" id="um-packages" onSelect={() => onNavigate("packages")}>
            <span>Packages</span>
            <IconChevronRight />
          </DropdownMenu.Item>
          <DropdownMenu.Item className="um-item" id="um-devices" onSelect={() => onNavigate("devices")}>
            <span>Devices</span>
            <IconChevronRight />
          </DropdownMenu.Item>
          <DropdownMenu.Item className="um-item" id="um-share" onSelect={() => onShare && onShare()}>
            <span>Open on phone</span>
            <IconQR />
          </DropdownMenu.Item>
          <div style={{ padding: "8px 10px 10px" }}><InstallButton className="btn btn-primary btn-sm" /></div>
          <DropdownMenu.Item asChild>
            <a className="um-item" id="um-docs" href="https://github.com/cfpperche/picode#readme" target="_blank" rel="noopener noreferrer">
              <span>Documentation</span>
              <IconExternal />
            </a>
          </DropdownMenu.Item>

          <DropdownMenu.Separator className="um-divider" />
          <div className="um-version">PiCode <span id="um-ver">{version ? "v" + version : ""}</span></div>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
