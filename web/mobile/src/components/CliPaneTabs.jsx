import { cliPaneHash } from "@picode/shared/domain/cliLaunch.js";
import { cliSettingsHash } from "@picode/shared/domain/cliSettings.js";
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
  { id: "packages", label: "Packages" },
  { id: "connectors", label: "Connectors" },
];

export function cliSetupHref(cli, pane, ctx = {}, workspace = "") {
  if (pane === "settings") return cliSettingsHash(cli, { agentId: ctx.agentId || "", focus: ctx.focus || "" });
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

export default function CliPaneTabs({ cli, pane = "launch", workspace = "", workspaceId = "", agentId = "", scope = "user", focus = "", actions = null, hasPackageUpdates = false }) {
  const ctx = { workspaceId, agentId, scope, focus };
  return (
    <div className="cli-pane-bar">
      <nav className="cli-pane-tabs" role="tablist" aria-label="CLI sections">
        {RUN.map((item) => <Tab key={item.id} cli={cli} pane={pane} workspace={workspace} ctx={ctx} item={item} />)}
        <span className="cli-pane-split" aria-hidden="true" />
        {SETUP.map((item) => (
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
