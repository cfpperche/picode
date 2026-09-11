import { cliPaneHash } from "@picode/shared/domain/cliLaunch.js";

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

function Tab({ cli, pane, workspace, item, extra }) {
  const selected = pane === item.id;
  return (
    <a
      href={cliPaneHash(cli, item.id, item.id === "sessions" ? workspace : "")}
      role="tab"
      aria-selected={selected}
      aria-current={selected ? "page" : undefined}
    >{item.label}{extra}</a>
  );
}

export default function CliPaneTabs({ cli, pane = "launch", workspace = "", actions = null, hasPackageUpdates = false }) {
  return (
    <div className="cli-pane-bar">
      <nav className="cli-pane-tabs" role="tablist" aria-label="CLI sections">
        {RUN.map((item) => <Tab key={item.id} cli={cli} pane={pane} workspace={workspace} item={item} />)}
        <span className="cli-pane-split" aria-hidden="true" />
        {SETUP.map((item) => (
          <Tab
            key={item.id}
            cli={cli}
            pane={pane}
            workspace={workspace}
            item={item}
            extra={item.id === "packages" && hasPackageUpdates ? <span aria-label="Package updates available"> •</span> : null}
          />
        ))}
      </nav>
      {actions ? <div className="cli-pane-actions" data-align-row>{actions}</div> : null}
    </div>
  );
}
