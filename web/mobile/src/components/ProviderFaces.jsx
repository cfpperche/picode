import { useState } from "react";
import { faceSlice, providerFaviconUrl, providerId, providerLetter } from "@picode/shared/domain/providerIcon.js";
import { agentIsPi } from "@picode/shared/domain/managedPrincipal.js";
import { terminalCliFaviconUrls, terminalCliLabel, terminalCliMark } from "@picode/shared/domain/terminalCli.js";

// A guest agent has no provider identity — its face is the CLI it runs
// (ADR-0160): the same favicon the terminal rows wear, falling back to the
// vendor mark. Pi agents keep the provider face below.
function GuestCliFace({ cli, name }) {
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
  if (agent && !agentIsPi(agent) && agent.cli) {
    return <GuestCliFace cli={agent.cli} name={agent.name} />;
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

export default function ProviderFaces({ agents }) {
  const list = agents || [];
  if (!list.length) return <span className="side-empty-hint">— empty</span>;
  const { shown, extra } = faceSlice(list);
  return (
    <span className="ws-faces">
      {shown.map((ag) => <ProviderFace key={ag.id} agent={ag} />)}
      {extra ? <span className="ws-face ws-face-more" title={extra + " more"}>+{extra}</span> : null}
    </span>
  );
}
