export default function CliTabs({ view }) {
  return <nav className="cli-tabs" aria-label="Agent CLIs">
    <a href="#/clis" aria-current={!["terminals", "sessions", "settings"].includes(view) ? "page" : undefined}>CLIs</a>
    <a href="#/clis/terminals" aria-current={view === "terminals" ? "page" : undefined}>Terminals</a>
    <a href="#/clis/sessions" aria-current={view === "sessions" ? "page" : undefined}>Sessions</a>
    <a href="#/clis/settings/pi" aria-current={view === "settings" ? "page" : undefined}>Settings</a>
  </nav>;
}
