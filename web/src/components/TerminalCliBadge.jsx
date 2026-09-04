import { useState } from "react";
import { IconTerminal } from "./Icons.jsx";
import { terminalCli, terminalCliFaviconUrls, terminalCliLabel, terminalCliMark } from "../lib/terminalCli.js";

// A loaded favicon is the identity, same as ProviderFace: the image fills
// the slot with no chip behind it. The boxed badge is only for fallback
// marks (and the unused labeled variant).
export default function TerminalCliBadge({ term, showLabel = false }) {
  const cli = terminalCli(term);
  const label = cli ? terminalCliLabel(cli) : "Terminal";
  const favicons = cli ? terminalCliFaviconUrls(cli) : [];
  const [failedCount, setFailedCount] = useState(0);
  const favicon = failedCount < favicons.length ? favicons[failedCount] : "";
  const showFavicon = Boolean(favicon);
  const failNext = () => setFailedCount((n) => n + 1);

  if (showFavicon && !showLabel) {
    return (
      <img
        className="ws-face term-cli-face"
        src={favicon}
        alt=""
        title={label}
        aria-label={label}
        onError={failNext}
      />
    );
  }

  const badgeClass =
    "term-cli-badge" +
    (showFavicon ? " has-favicon" : cli ? " cli-" + cli : " is-shell");
  return (
    <span className={badgeClass} title={label} aria-label={label}>
      {showFavicon ? (
        <img className="term-cli-favicon" src={favicon} alt="" onError={failNext} />
      ) : cli ? (
        <span className="term-cli-glyph" aria-hidden="true">{terminalCliMark(cli)}</span>
      ) : <IconTerminal size={15} />}
      {showLabel ? <span className="term-cli-label">{label}</span> : null}
    </span>
  );
}
