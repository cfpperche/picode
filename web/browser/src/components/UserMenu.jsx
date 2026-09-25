import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { useRef, useState } from "react";
import { IconUser, IconChevronUp, IconSun, IconMonitor, IconMoon, IconPhone, IconChevronRight, IconExternal, IconQR, IconMode, IconSettings, IconDrive, IconProvider, IconMcp, IconPackage, IconClock, IconSparkles, IconCli, IconModel, IconFile, IconGlobe, IconCheck, IconHistory } from "./Icons.jsx";
import { menuGroups, menuActions, menuHasResults } from "../lib/userMenuModel.js";
import { DOCS_BASE } from "../lib/commandDocs.js";
import InstallButton from "./InstallButton.jsx";

const SECTION_ICONS = {
  clis: IconCli,
  automations: IconClock,
  snippets: IconFile,
  outcomes: IconCheck,
  history: IconHistory,
  providers: IconProvider,
  connectors: IconMcp,
  settings: IconSettings,
  termset: IconSettings,
  browser: IconGlobe,
  computer: IconMonitor,
  integrations: IconMcp,
  llama: IconModel,
  packages: IconPackage,
  preferences: IconMode,
  devices: IconPhone,
  system: IconDrive,
};

const ACTION_ICONS = { "whats-new": IconSparkles, share: IconQR, docs: IconExternal };

export default function UserMenu({ host, version, inShell = false, themeMode, onTheme, onNavigate, onShare, onWhatsNew, whatsNewUnread, pkgUpdates, onDocs }) {
  const [query, setQuery] = useState("");
  const contentRef = useRef(null);
  const hasPkgUp = !!(pkgUpdates && pkgUpdates.length);
  const hasNotice = !!whatsNewUnread;
  const searching = !!query.trim();
  const groups = menuGroups(query);
  const actions = menuActions(query);
  const rows = (list) => list.filter(([id]) => (id !== "browser" && id !== "computer") || inShell).map(([id, title, sub]) => {
    const Icon = SECTION_ICONS[id];
    return (
      <DropdownMenu.Item key={id} className="um-item" id={"um-" + id} onSelect={() => onNavigate(id)}>
        {Icon ? <Icon className="um-item-ico" /> : null}
        <span className="um-item-text">
          <span className="um-item-name">{title}{id === "clis" && hasPkgUp ? <span className="um-dot" aria-label="Updates available" /> : null}</span>
          <span className="um-row-sub">{sub}</span>
        </span>
        <IconChevronRight />
      </DropdownMenu.Item>
    );
  });

  // Keep the search keys out of the menu's typeahead; Escape still closes.
  const onSearchKeyDown = (event) => {
    if (event.key === "Escape") return;
    event.stopPropagation();
    if (event.key === "ArrowDown") {
      const first = contentRef.current && contentRef.current.querySelector("[role^='menuitem']");
      if (first) { event.preventDefault(); first.focus(); }
    }
  };

  const renderAction = (action) => {
    const Icon = ACTION_ICONS[action.id];
    if (action.id === "docs") {
      // In-app browser tab when the app can open one (ADR-0169's pane taught
      // the route); the system browser remains the no-shell fallback, where
      // the shell bridge used to catch this very anchor.
      return (
        <DropdownMenu.Item className="um-item" id="um-docs" onSelect={() => (onDocs ? onDocs() : window.open(DOCS_BASE + "/", "_blank", "noopener,noreferrer"))}>
          <Icon className="um-item-ico" />
          <span className="um-item-text">
            <span className="um-item-name">{action.title}</span>
            <span className="um-row-sub">{action.sub}</span>
          </span>
          <IconExternal />
        </DropdownMenu.Item>
      );
    }
    return (
      <DropdownMenu.Item key={action.id} className="um-item" id={"um-" + action.id} onSelect={action.id === "share" ? () => onShare && onShare() : () => onWhatsNew && onWhatsNew()}>
        <Icon className="um-item-ico" />
        <span className="um-item-text">
          <span className="um-item-name">{action.title}{action.id === "whats-new" && whatsNewUnread ? <span className="um-dot" aria-label="New release notes" /> : null}</span>
          <span className="um-row-sub">{action.sub}</span>
        </span>
        <IconChevronRight />
      </DropdownMenu.Item>
    );
  };

  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button className="um-trigger" id="um-trigger" aria-label={hasNotice ? (host || "This machine") + ", updates available" : undefined}>
          <span className="um-avatar" aria-hidden="true"><IconUser />{hasNotice ? <span className="um-dot" /> : null}</span>
          <span className="um-meta">
            <span className="um-name" id="um-name">{host || <span className="um-name-skel" aria-label="Loading the machine name" />}</span>
            <span className="um-sub" id="um-sub">this machine</span>
          </span>
          <IconChevronUp className="um-chev" />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className="um-popover" side="top" align="start" sideOffset={6} collisionPadding={8} ref={contentRef}>
          <div className="um-account">
            <span className="um-avatar" aria-hidden="true"><IconUser /></span>
            <div className="um-account-meta">
              <span className="um-account-name" id="um-name2">{host || "This machine"}</span>
              <span className="um-account-sub">PiCode on this machine</span>
            </div>
          </div>

          <div className="um-search" onKeyDown={onSearchKeyDown}>
            <input
              type="search"
              className="um-search-input"
              aria-label="Search tools and settings"
              placeholder="Search tools and settings"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </div>

          {menuHasResults(query) ? (
            <>
              {groups.map((group) => (
                <div className="um-group" key={group.title} aria-label={group.title}>
                  <div className="um-label">{group.title}</div>
                  {rows(group.rows)}
                </div>
              ))}
              {actions.length ? (
                <div className="um-group" aria-label="Continue">
                  <div className="um-label">Continue</div>
                  {actions.map(renderAction)}
                </div>
              ) : null}
            </>
          ) : (
            <div className="um-empty" role="status">
              <span>No matching tools or settings.</span>
              <button type="button" className="btn btn-sm" onClick={() => setQuery("")}>Clear search</button>
            </div>
          )}

          {!searching ? (
            <>
              <DropdownMenu.Separator className="um-divider" />
              <div className="um-label">Theme</div>
              <div className="um-theme" role="radiogroup" aria-label="Theme">
                <button type="button" role="radio" aria-checked={themeMode === "light"} data-theme-option="light" data-active={themeMode === "light" ? "1" : ""} onClick={() => onTheme("light")}>
                  <IconSun /> Light
                </button>
                <button type="button" role="radio" aria-checked={themeMode === "system"} data-theme-option="system" data-active={themeMode === "system" ? "1" : ""} onClick={() => onTheme("system")}>
                  <IconMonitor /> System
                </button>
                <button type="button" role="radio" aria-checked={themeMode === "dark"} data-theme-option="dark" data-active={themeMode === "dark" ? "1" : ""} onClick={() => onTheme("dark")}>
                  <IconMoon /> Dark
                </button>
              </div>

              <DropdownMenu.Separator className="um-divider" />
              {/* The desktop shell IS the installed app — installing again
                  from inside it makes no sense, so the entry is browser-only. */}
              {!inShell ? <div className="um-install"><InstallButton className="btn btn-primary btn-sm" /></div> : null}
              <div className="um-version">PiCode <span id="um-ver">{version ? "v" + version : ""}</span></div>
            </>
          ) : null}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
