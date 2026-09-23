import { matchesListSearch } from "./mobileListSearch.js";

// More-screen model — mirrors the desktop user menu groups and copy
// (docs/benchmarks/2026-09-07-mobile-v2.md). Pins, Apps, llama.cpp and
// Notifications are mobile-only; Providers/Packages stay search-only
// because those routes live under Agent CLIs.
export const MORE_SECTIONS = [
  ["missions", "Missions", "Objectives, evidence and agent handoffs"],
  ["pins", "Pins", "Notes, files and reminders"],
  ["snippets", "Snippets", "Reusable prompts and commands"],
  ["outcomes", "Outcomes", "How removed agents ended"],
  ["automations", "Automations", "Scheduled and triggered work"],
  ["clis", "Agent CLIs", "Launches, sessions and CLI configuration"],
  ["apps", "Apps", "Docker and other tools"],
  ["notifications", "Notifications", "Push when an agent needs you"],
  ["llama", "llama.cpp", "Models and server connection"],
  ["providers", "Providers", "Accounts, keys, usage"],
  ["preferences", "Preferences", "Theme, notifications, backup"],
  ["connectors", "Connectors", "MCP servers and tools"],
  ["integrations", "Webhooks", "Signed event delivery"],
  ["packages", "Packages", "Plugins, extensions, updates"],
  ["devices", "Devices", "Who is connected"],
  ["system", "System", "Version, host, paths"],
];

export const MORE_TITLES = { ...Object.fromEntries(MORE_SECTIONS.map(([id, t]) => [id, t])), mcps: "MCP servers" };

export const MORE_GROUPS = [
  ["Tools", ["missions", "pins", "snippets", "outcomes", "clis", "automations", "apps", "llama"]],
  ["PiCode", ["preferences", "notifications", "devices", "system", "integrations"]],
];

export const MORE_ACTIONS = [
  { id: "updates", title: "What’s new", sub: "Release highlights" },
  { id: "pair", title: "Open on another phone", sub: "Pair with a QR code" },
  { id: "desktop", title: "Desktop layout", sub: "Open the desktop workspace" },
];

export function moreGroups(query) {
  const groups = MORE_GROUPS
    .map(([title, ids]) => ({
      title,
      rows: ids.map(id => MORE_SECTIONS.find(row => row[0] === id)).filter(row => matchesListSearch(query, title, ...row)),
    }))
    .filter(group => group.rows.length);
  // The CLI-pane shortcuts share one "Agent CLIs" group; one group per
  // match repeated the heading over adjacent rows.
  if (query.trim()) {
    const cliRows = [
      ["pi-settings", "CLI settings", "The last agent's CLI configuration", "pi model thinking prompt"],
      ["pi-packages", "Packages", "Plugins, extensions and updates", "plugins extensions updates"],
      ["pi-skills", "Skills", "Agent Skills each CLI loads", "skills SKILL.md agent skills"],
      ["pi-providers", "Providers", "Accounts, keys and usage", "accounts keys usage login"],
      ["connectors", "Connectors", "MCP servers and tools", "MCP servers tools"],
    ].filter(([, title, , keys]) => matchesListSearch(query, title, keys)).map(([id, title, sub]) => [id, title, sub]);
    if (cliRows.length) groups.unshift({ title: "Agent CLIs", rows: cliRows });
  }
  return groups;
}

export function moreActions(query) {
  return MORE_ACTIONS.filter(row => matchesListSearch(query, row.title, row.sub, row.id));
}

export function moreHasResults(query) {
  return moreGroups(query).length > 0 || moreActions(query).length > 0;
}
