import { cliPaneHash } from "@picode/shared/domain/cliLaunch.js";
import { cliSettingsHash, cliSettingsQuery } from "@picode/shared/domain/cliSettings.js";
import { cliPackagesHash } from "@picode/shared/domain/cliPackages.js";
import { cliConnectorsHash } from "@picode/shared/domain/integrations.js";

const RUN = [
  { id: "launch", label: "Launch" },
  { id: "terminals", label: "Terminals" },
  { id: "sessions", label: "Sessions" },
];
const SETUP = [
  { id: "providers", label: "Providers" },
  { id: "settings", label: "Settings" },
  // "Keyboard", not "Keys": the pane next door configures providers, where
  // "keys" means credentials (owner picked the name, 2026-09-12).
  { id: "keyboard", label: "Keyboard" },
  { id: "packages", label: "Packages" },
  { id: "connectors", label: "Connectors" },
];

export function cliSetupHref(cli, pane, ctx = {}, workspace = "") {
  // The layer rides along: this href is written by the pane's "carry the
  // selected agent" rewrite, and dropping the layer there made a layer pill
  // click look like it did nothing (2026-09-12).
  if (pane === "settings") return cliSettingsHash(cli, { agentId: ctx.agentId || "", focus: ctx.focus || "", layer: ctx.layer || "" });
  // The keyboard map is machine-wide, but the link keeps the settings context:
  // going there and back must not move the reader to another agent or layer.
  if (pane === "keyboard") return cliPaneHash(cli, "keyboard") + cliSettingsQuery({ agentId: ctx.agentId || "", layer: ctx.layer || "" });
  if (pane === "packages") return cliPackagesHash(cli, { workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "", scope: ctx.scope || "user" });
  if (pane === "connectors") return cliConnectorsHash(cli, { workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "", scope: ctx.scope || "user" });
  return cliPaneHash(cli, pane, pane === "sessions" ? workspace : "");
}

function Tab({ cli, pane, workspace, ctx, item, extra }) {
  const selected = pane === item.id;
  return (
    <a
      href={cliSetupHref(cli, item.id, ctx, workspace)}
      role="tab"
      aria-selected={selected}
      aria-current={selected ? "page" : undefined}
    >{item.label}{extra}</a>
  );
}

export default function CliPaneTabs({ cli, pane = "launch", panes = null, workspace = "", workspaceId = "", agentId = "", scope = "user", focus = "", layer = "", actions = null, hasPackageUpdates = false }) {
  const ctx = { workspaceId, agentId, scope, focus, layer };
  // panes comes from cliPanes(cli): a CLI with no adapter shows Launch and
  // Terminals only, and the setup group (all Pi-side editors) is absent.
  const allowed = panes ? new Set(panes) : null;
  const run = RUN.filter((item) => !allowed || allowed.has(item.id));
  const setup = SETUP.filter((item) => !allowed || allowed.has(item.id));
  return (
    <div className="cli-pane-bar">
      <nav className="cli-pane-tabs" role="tablist" aria-label="CLI sections">
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
      </nav>
      {actions ? <div className="cli-pane-actions" data-align-row>{actions}</div> : null}
    </div>
  );
}
