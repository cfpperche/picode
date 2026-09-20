import { useState } from "react";
import { faceSlice, providerFaviconUrl, providerId, providerLetter } from "@picode/shared/domain/providerIcon.js";
import { agentIsPi } from "@picode/shared/domain/managedPrincipal.js";
import { terminalDisplayCli, terminalCliFaviconUrls, terminalCliLabel, terminalCliMark } from "@picode/shared/domain/terminalCli.js";
import { collapseFaceItems } from "../lib/collapseFaces.js";

// Every agent wears its CLI's favicon (ADR-0160) — pi included, the same
// art the terminal rows wear — falling back to the vendor mark. The
// provider face below stays for explicit provider ids.
function CliAgentFace({ cli, name }) {
  const favicons = terminalCliFaviconUrls(cli);
  const [failed, setFailed] = useState(0);
  const src = failed < favicons.length ? favicons[failed] : "";
  const title = name || terminalCliLabel(cli);
  if (src) {
    return <img className="ws-face" src={src} alt="" title={title} onError={() => setFailed((n) => n + 1)} />;
  }
  return <span className="ws-face" title={title}>{terminalCliMark(cli)}</span>;
}

export function ProviderFace({ agent, id }) {
  if (!id && agent) {
    return <CliAgentFace cli={agentIsPi(agent) ? "pi" : agent.cli} name={agent.name} />;
  }
  const pid = (id || providerId(agent)).toLowerCase();
  const src = providerFaviconUrl(pid);
  const letter = providerLetter(pid || (agent && agent.name));
  const [fail, setFail] = useState(false);
  const title = pid || (agent && agent.name) || "agent";
  if (!src || fail) {
    return <span className="ws-face" title={title}>{letter}</span>;
  }
  return <img className="ws-face" src={src} alt="" title={title} onError={() => setFail(true)} />;
}

// Terminal in the collapsed strip: the CLI's own favicon when one loads,
// otherwise the vendor mark (π, Cl, Cx, G) — plain shells wear ">_".
// Deliberately the plain ws-face, no term-cli-face override: in the strip
// agents and terminals are one visual family (same 16px plate, ring and
// contained art); the full-bleed look stays a row-identity treatment.
export function TermFace({ term }) {
  const cli = terminalDisplayCli(term);
  const label = terminalCliLabel(cli);
  const title = (term.name || "Terminal") + (cli ? " — " + label : "");
  const favicons = terminalCliFaviconUrls(cli);
  const [failed, setFailed] = useState(0);
  const src = failed < favicons.length ? favicons[failed] : "";
  if (src) {
    return (
      <img
        className="ws-face" src={src} alt="" title={title}
        onError={() => setFailed((n) => n + 1)}
      />
    );
  }
  return <span className="ws-face" title={title}>{terminalCliMark(cli)}</span>;
}

export default function ProviderFaces({ agents, terms }) {
  const items = collapseFaceItems(agents, terms);
  if (!items.length) return <span className="side-empty-hint">— empty</span>;
  const { shown, extra } = faceSlice(items);
  return (
    <span className="ws-faces">
      {shown.map((it) => it.kind === "agent"
        ? <ProviderFace key={it.id} agent={it.ag} />
        : <TermFace key={it.id} term={it.term} />)}
      {extra ? <span className="ws-face ws-face-more" title={extra + " more"}>+{extra}</span> : null}
    </span>
  );
}
