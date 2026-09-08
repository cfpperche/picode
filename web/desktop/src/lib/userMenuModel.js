import { matchesListSearch } from "@picode/shared/domain/listSearch.js";

// User menu model — mirrors the mobile v2 More screen groups and copy
// (docs/benchmarks/2026-09-07-mobile-v2.md). Mobile-only sections (Apps,
// llama.cpp, Notifications) have no desktop route and stay out of the menu.
export const MENU_SECTIONS = [
  ["clis", "Agent CLIs", "Pi settings, launches and sessions"],
  ["automations", "Automations", "Scheduled and triggered work"],
  ["providers", "Providers", "Accounts, keys, usage"],
  ["integrations", "Integrations", "Connectors and event delivery"],
  ["packages", "Packages", "Skills, extensions, updates"],
  ["preferences", "Preferences", "Theme, notifications, backup"],
  ["devices", "Devices", "Who is connected"],
  ["system", "System", "Version, host, paths"],
];

export const MENU_GROUPS = [
  ["Tools", ["clis", "automations"]],
  ["Agents and connections", ["providers", "integrations", "packages"]],
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
  return groups;
}

export function menuActions(query) {
  return MENU_ACTIONS.filter(row => matchesListSearch(query, row.title, row.sub, row.id));
}

export function menuHasResults(query) {
  return menuGroups(query).length > 0 || menuActions(query).length > 0;
}
