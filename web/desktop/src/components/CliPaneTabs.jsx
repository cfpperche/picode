import { cliPaneHash } from "@picode/shared/domain/cliLaunch.js";

const PANES = [
  { id: "launch", label: "Launch" },
  { id: "terminals", label: "Terminals" },
  { id: "sessions", label: "Sessions" },
];

export default function CliPaneTabs({ cli, pane = "launch", workspace = "", actions = null }) {
  return (
    <div className="cli-pane-bar">
      <nav className="cli-pane-tabs" role="tablist" aria-label="CLI sections">
        {PANES.map((item) => {
          const selected = pane === item.id;
          return (
            <a
              key={item.id}
              href={cliPaneHash(cli, item.id, item.id === "sessions" ? workspace : "")}
              role="tab"
              aria-selected={selected}
              aria-current={selected ? "page" : undefined}
            >{item.label}</a>
          );
        })}
      </nav>
      {actions ? <div className="cli-pane-actions" data-align-row>{actions}</div> : null}
    </div>
  );
}
