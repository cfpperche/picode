export default function CliTabs({ view, packagesHref = "#/clis/packages/pi", hasPackageUpdates = false }) {
  return <nav className="cli-tabs" aria-label="Agent CLIs">
    <a href="#/clis" aria-current={!["terminals", "sessions", "settings", "packages"].includes(view) ? "page" : undefined}>CLIs</a>
    <a href="#/clis/terminals" aria-current={view === "terminals" ? "page" : undefined}>Terminals</a>
    <a href="#/clis/sessions" aria-current={view === "sessions" ? "page" : undefined}>Sessions</a>
    <a href="#/clis/settings/pi" aria-current={view === "settings" ? "page" : undefined}>Settings</a>
    <a href={packagesHref} aria-current={view === "packages" ? "page" : undefined}>Packages{hasPackageUpdates ? <span aria-label="Package updates available"> •</span> : null}</a>
  </nav>;
}
