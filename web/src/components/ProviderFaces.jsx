import { useState } from "react";
import { faceSlice, providerFaviconUrl, providerId, providerLetter } from "../lib/providerIcon.js";

export function ProviderFace({ agent }) {
  const id = providerId(agent);
  const src = providerFaviconUrl(id);
  const letter = providerLetter(id || (agent && agent.name));
  const [fail, setFail] = useState(false);
  const title = id || (agent && agent.name) || "agent";
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
