import { matchesListSearch } from "./mobileListSearch.js";

// More-screen model — mirrors the desktop user menu groups and copy
// (docs/benchmarks/2026-09-07-mobile-v2.md). Pins, Apps, llama.cpp and
// Notifications are mobile-only; Providers/Packages stay search-only
// because those routes live under Agent CLIs.
export const MORE_SECTIONS = [
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

export const MORE_TITLES = { ...Object.fromEntries(MORE_SECTIONS.map(([id, t]) => [id, t])), mcps: "MCP servers" };

export const MORE_GROUPS = [
  ["Tools", ["pins", "clis", "automations", "apps", "llama", "integrations"]],
  ["PiCode", ["preferences", "notifications", "devices", "system"]],
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
  if (query.trim() && matchesListSearch(query, "Pi settings", "model thinking prompt")) groups.unshift({ title: "Agent CLIs", rows: [["pi-settings", "Pi settings", "Model, thinking, tools and keys"]] });
  if (query.trim() && matchesListSearch(query, "Packages", "skills extensions updates")) groups.unshift({ title: "Agent CLIs", rows: [["pi-packages", "Packages", "Pi skills, extensions and updates"]] });
  if (query.trim() && matchesListSearch(query, "Providers", "accounts keys usage login")) groups.unshift({ title: "Agent CLIs", rows: [["pi-providers", "Providers", "Pi accounts, keys and usage"]] });
  return groups;
}

export function moreActions(query) {
  return MORE_ACTIONS.filter(row => matchesListSearch(query, row.title, row.sub, row.id));
}

export function moreHasResults(query) {
  return moreGroups(query).length > 0 || moreActions(query).length > 0;
}
