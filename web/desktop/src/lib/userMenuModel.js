import { matchesListSearch } from "@picode/shared/domain/listSearch.js";

// User menu model — mirrors the mobile v2 More screen groups and copy
// (docs/benchmarks/2026-09-07-mobile-v2.md). Mobile-only sections (Apps,
// llama.cpp, Notifications) have no desktop route and stay out of the menu.
export const MENU_SECTIONS = [
  ["clis", "Agent CLIs", "Launch settings and terminals"],
  ["automations", "Automations", "Scheduled and triggered work"],
  ["providers", "Providers", "Accounts, keys, usage"],
  ["settings", "Settings", "Pi: model, thinking, prompt"],
  ["integrations", "Integrations", "Connectors and event delivery"],
  ["packages", "Packages", "Skills, extensions, updates"],
  ["preferences", "Preferences", "Theme, notifications, backup"],
  ["devices", "Devices", "Who is connected"],
  ["system", "System", "Version, host, paths"],
];

export const MENU_GROUPS = [
  ["Tools", ["clis", "automations"]],
  ["Agents and connections", ["providers", "settings", "integrations", "packages"]],
  ["PiCode", ["preferences", "devices", "system"]],
];

export const MENU_ACTIONS = [
  { id: "whats-new", title: "What’s new", sub: "Release highlights" },
  { id: "share", title: "Open on phone", sub: "Pair with a QR code" },
  { id: "docs", title: "Documentation", sub: "Guides and reference" },
];

export function menuGroups(query) {
  return MENU_GROUPS
    .map(([title, ids]) => ({
      title,
      rows: ids.map(id => MENU_SECTIONS.find(row => row[0] === id)).filter(row => matchesListSearch(query, ...row)),
    }))
    .filter(group => group.rows.length);
}

export function menuActions(query) {
  return MENU_ACTIONS.filter(row => matchesListSearch(query, row.title, row.sub, row.id));
}

export function menuHasResults(query) {
  return menuGroups(query).length > 0 || menuActions(query).length > 0;
}
