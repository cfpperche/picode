import { supportsCliProviders } from "@picode/shared/domain/cliProviders.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import CliCredentials from "./CliCredentials.jsx";

// One providers surface for all nine agent CLIs (ADR-0169). pi's own editor is
// gone, so this file is only the pane's route contract: which CLI, whether the
// URL asked for the Add door, and the custom-endpoint sub-route pi owns (which
// CliCredentials renders, because that page needs the catalog the pane fetched).
export default function CliProviders({ hidden, cli = "pi", add = false, invalid = false, scoped = false, custom = "", customId = "", onCatalogChange }) {
  const supported = supportsCliProviders(cli);
  const blocked = invalid || scoped || !supported;
  return <section id="cli-providers-view" className="cli-providers-pane" hidden={hidden}>
    {blocked ? <div className="cli-notice" role="status"><span>{invalid ? "This provider link is invalid." : scoped ? "Accounts stay on this machine." : "Providers for " + terminalCliLabel(cli) + " are in development — coming soon."}</span></div>
      : hidden ? null : <CliCredentials hidden={false} cli={cli} add={add} custom={custom} customId={customId} onCatalogChange={onCatalogChange} />}
  </section>;
}
