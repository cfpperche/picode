import OverflowTabs from "./OverflowTabs.jsx";
import { cliModelsHash, cliPaneHash } from "@picode/shared/domain/cliLaunch.js";
import { MEMORY_WORDS, MODEL_WORDS, PACKAGE_WORDS, SKILL_WORDS } from "@picode/shared/domain/scopes.js";
import { cliSettingsHash, cliSettingsQuery } from "@picode/shared/domain/cliSettings.js";
import { cliPackagesHash } from "@picode/shared/domain/cliPackages.js";
import { cliMemoryHash } from "@picode/shared/domain/cliNative.js";
import { cliConnectorsHash } from "@picode/shared/domain/integrations.js";
import { cliSkillsHash } from "@picode/shared/domain/cliSkills.js";

const RUN = [
  { id: "launch", label: "Launch" },
  { id: "terminals", label: "Terminals" },
  { id: "sessions", label: "Sessions" },
];
const SETUP = [
  { id: "providers", label: "Providers" },
  // What the CLI can reach, and which of it it may use (ADR-0181). Only CLIs
  // PiCode can ask carry it — cliPanes decides.
  { id: "models", label: "Models" },
  { id: "settings", label: "Settings" },
  // "Keyboard", not "Keys": the pane next door configures providers, where
  // "keys" means credentials (owner picked the name, 2026-09-12).
  { id: "keyboard", label: "Keyboard" },
  // What the CLI has remembered between sessions (ADR-0163). Every CLI has
  // the tab; the ones with no native memory answer in one line.
  { id: "memory", label: "Memory" },
  { id: "packages", label: "Packages" },
  // Standalone Agent Skills each CLI loads, and from where (ADR-0196).
  { id: "skills", label: "Skills" },
  { id: "connectors", label: "Connectors" },
];

export function cliSetupHref(cli, pane, ctx = {}, workspace = "") {
  // The layer rides along: this href is written by the pane's "carry the
  // selected agent" rewrite, and dropping the layer there made a layer pill
  // click look like it did nothing (2026-09-12).
  if (pane === "settings") return cliSettingsHash(cli, { workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "", focus: ctx.focus || "", layer: ctx.layer || "" });
  // The keyboard map is machine-wide, but the link keeps the settings context:
  // going there and back must not move the reader to another agent, workspace or layer.
  if (pane === "keyboard") return cliPaneHash(cli, "keyboard") + cliSettingsQuery({ workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "", layer: ctx.layer || "" });
  if (pane === "models") return cliModelsHash(cli, { workspaceId: ctx.workspaceId || "", layer: MODEL_WORDS[ctx.scope] || "" });
  if (pane === "memory") return cliMemoryHash(cli, { workspaceId: ctx.workspaceId || "", scope: MEMORY_WORDS[ctx.scope] || "" });
  if (pane === "packages") return cliPackagesHash(cli, { workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "", scope: PACKAGE_WORDS[ctx.scope] || "user" });
  if (pane === "skills") return cliSkillsHash(cli, { workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "", scope: SKILL_WORDS[ctx.scope] || "" });
  if (pane === "connectors") return cliConnectorsHash(cli, { workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "", scope: PACKAGE_WORDS[ctx.scope] || "user" });
  return cliPaneHash(cli, pane, pane === "sessions" ? workspace : "");
}

function Tab({ cli, pane, workspace, ctx, item, extra }) {
  const selected = pane === item.id;
  return (
    <a
      data-tab={item.id}
      href={cliSetupHref(cli, item.id, ctx, workspace)}
      role="tab"
      aria-selected={selected}
      aria-current={selected ? "page" : undefined}
    >{item.label}{extra}</a>
  );
}

export default function CliPaneTabs({ cli, pane = "launch", panes = null, workspace = "", workspaceId = "", agentId = "", scope = "", focus = "", layer = "", actions = null, hasPackageUpdates = false }) {
  const ctx = { workspaceId, agentId, scope, focus, layer };
  // panes comes from cliPanes(cli). Every CLI carries the setup group; the
  // ones without a native editor render it as an in-development placeholder
  // (owner, 2026-09-15), so the strip is the same shape everywhere.
  const allowed = panes ? new Set(panes) : null;
  const run = RUN.filter((item) => !allowed || allowed.has(item.id));
  const setup = SETUP.filter((item) => !allowed || allowed.has(item.id));
  // Nine tabs and the pane's actions share one bar, so at a narrow width
  // the strip overflows: OverflowTabs gives it the editor strip's arrows,
  // fades and "All sections" list, and keeps the selected tab in view.
  const items = [...run, ...setup].map((item) => ({ id: item.id, label: item.label, href: cliSetupHref(cli, item.id, ctx, workspace) }));
  return (
    <div className="cli-pane-bar">
      <OverflowTabs className="cli-pane-tabs" role="tablist" label="CLI sections" listLabel="All sections" items={items} selectedId={pane}>
        {run.map((item) => <Tab key={item.id} cli={cli} pane={pane} workspace={workspace} ctx={ctx} item={item} />)}
        {run.length && setup.length ? <span className="cli-pane-split" aria-hidden="true" /> : null}
        {setup.map((item) => (
          <Tab
            key={item.id}
            cli={cli}
            pane={pane}
            workspace={workspace}
            ctx={ctx}
            item={item}
            extra={item.id === "packages" && hasPackageUpdates ? <span aria-label="Package updates available"> •</span> : null}
          />
        ))}
      </OverflowTabs>
      {actions ? <div className="cli-pane-actions" data-align-row>{actions}</div> : null}
    </div>
  );
}
