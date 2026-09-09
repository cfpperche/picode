import { matchesListSearch } from "@picode/shared/domain/listSearch.js";

// User menu model — mirrors the mobile v2 More screen groups and copy
// (docs/benchmarks/2026-09-07-mobile-v2.md). Tools holds Agent CLIs,
// Automations, llama.cpp and Integrations (native providers/packages/
// settings live under Agent CLIs). Mobile-only sections (Apps,
// Notifications) have no desktop route and stay out of the menu.
export const MENU_SECTIONS = [
  ["clis", "Agent CLIs", "Launches, sessions and CLI configuration"],
  ["automations", "Automations", "Scheduled and triggered work"],
  ["providers", "Providers", "Accounts, keys, usage"],
  ["llama", "llama.cpp", "Models and server connection"],
  ["integrations", "Integrations", "Connectors and event delivery"],
  ["packages", "Packages", "Skills, extensions, updates"],
  ["preferences", "Preferences", "Theme, notifications, backup"],
  ["devices", "Devices", "Who is connected"],
  ["system", "System", "Version, host, paths"],
];

export const MENU_GROUPS = [
  ["Tools", ["clis", "automations", "llama", "integrations"]],
  ["PiCode", ["preferences", "devices", "system"]],
];

export const MENU_ACTIONS = [
  { id: "whats-new", title: "What’s new", sub: "Release highlights" },
  { id: "share", title: "Open on phone", sub: "Pair with a QR code" },
  { id: "docs", title: "Documentation", sub: "Guides and reference" },
];

export function menuGroups(query) {
  const groups = MENU_GROUPS
    .map(([title, ids]) => ({
      title,
      rows: ids.map(id => MENU_SECTIONS.find(row => row[0] === id)).filter(row => matchesListSearch(query, ...row)),
    }))
    .filter(group => group.rows.length);
  if (query.trim() && matchesListSearch(query, "Pi settings", "model thinking prompt")) groups.unshift({ title: "Agent CLIs", rows: [["settings", "Pi settings", "Model, thinking, tools and keys"]] });
  if (query.trim() && matchesListSearch(query, "Packages", "skills extensions updates")) groups.unshift({ title: "Agent CLIs", rows: [["packages", "Packages", "Pi skills, extensions and updates"]] });
  if (query.trim() && matchesListSearch(query, "Providers", "accounts keys usage login")) groups.unshift({ title: "Agent CLIs", rows: [["providers", "Providers", "Pi accounts, keys and usage"]] });
  return groups;
}

export function menuActions(query) {
  return MENU_ACTIONS.filter(row => matchesListSearch(query, row.title, row.sub, row.id));
}

export function menuHasResults(query) {
  return menuGroups(query).length > 0 || menuActions(query).length > 0;
}
