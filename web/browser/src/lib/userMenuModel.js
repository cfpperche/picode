import { matchesListSearch } from "@picode/shared/domain/listSearch.js";

// User menu model — mirrors the mobile v2 More screen groups and copy
// (docs/benchmarks/2026-09-07-mobile-v2.md). Tools holds Automations
// and llama.cpp. Agent CLIs opens from the sidebar header (search still
// finds it, plus Settings/Packages/Providers/Connectors). Webhooks live
// under PiCode. Mobile-only sections (Apps, Notifications) have no
// desktop route and stay out of the menu.
export const MENU_SECTIONS = [
  ["clis", "Agent CLIs", "Launches, sessions and CLI configuration"],
  ["automations", "Automations", "Scheduled and triggered work"],
  ["snippets", "Snippets", "Reusable prompts"],
  ["outcomes", "Outcomes", "How removed agents ended"],
  ["providers", "Providers", "Accounts, keys, usage"],
  ["llama", "llama.cpp", "Models and server connection"],
  ["connectors", "Connectors", "MCP servers and tools"],
  ["integrations", "Webhooks", "Signed event delivery"],
  ["packages", "Packages", "Plugins, extensions, updates"],
  ["preferences", "Preferences", "Theme, notifications, backup"],
  ["browser", "Browser", "Work browser and site access"],
  ["computer", "Computer", "Agents using this computer"],
  ["devices", "Devices", "Who is connected"],
  ["system", "System", "Version, host, paths"],
  ["termset", "Terminal defaults", "Font, colors, cursor"],
];

export const MENU_GROUPS = [
  ["Tools", ["automations", "snippets", "outcomes", "llama"]],
  ["PiCode", ["preferences", "browser", "computer", "devices", "system", "integrations", "termset"]],
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
  if (query.trim() && matchesListSearch(query, "Agent CLIs", "launches sessions CLI configuration", "tmux guard")) {
    const row = ["clis", "Agent CLIs", "Launches, sessions and CLI configuration"];
    const tools = groups.find(g => g.title === "Tools");
    if (tools) tools.rows.unshift(row);
    else groups.unshift({ title: "Tools", rows: [row] });
  }
  // The CLI-pane shortcuts share one "Agent CLIs" group; one group per
  // match repeated the heading over adjacent rows.
  if (query.trim()) {
    const cliRows = [
      ["settings", "CLI settings", "The selected agent's CLI configuration", "pi model thinking prompt"],
      ["packages", "Packages", "Plugins, extensions and updates", "plugins extensions updates"],
      ["skills", "Skills", "Agent Skills each CLI loads", "skills SKILL.md agent skills"],
      ["providers", "Providers", "Accounts, keys and usage", "accounts keys usage login"],
      ["connectors", "Connectors", "MCP servers and tools", "MCP servers tools"],
    ].filter(([, title, , keys]) => matchesListSearch(query, title, keys)).map(([id, title, sub]) => [id, title, sub]);
    if (cliRows.length) groups.unshift({ title: "Agent CLIs", rows: cliRows });
  }
  return groups;
}

export function menuActions(query) {
  return MENU_ACTIONS.filter(row => matchesListSearch(query, row.title, row.sub, row.id));
}

export function menuHasResults(query) {
  return menuGroups(query).length > 0 || menuActions(query).length > 0;
}
