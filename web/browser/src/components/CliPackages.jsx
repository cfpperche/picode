import Packages from "./Packages.jsx";

// One pane, every CLI (ADR-0176). The route names a CLI, the target it is
// looking at, and — when a link asked for one — the package whose configuration
// page it wants; the pane asks the engine what that CLI is and draws only what
// the answer declares. There is no list of CLIs here any more: the report
// answers for every driver the engine has and refuses the rest by naming them.
export default function CliPackages({ hidden, route, catalog, onPackageUpdates, describe = false }) {
  return <section id="cli-packages-view" hidden={hidden}>
    {route.invalid ?
      <div className="cli-notice" role="status"><span>This package link is invalid.</span></div>
      : !hidden ? <Packages
        key={[route.id, route.workspaceId, route.agentId, route.scope, route.pkg || "", describe ? "describe" : ""].join(":")}
        route={route} catalog={catalog} describe={describe} onPackageUpdates={onPackageUpdates}
      /> : null}
  </section>;
}
