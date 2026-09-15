import { useState } from "react";
import { IconTerminal } from "./Icons.jsx";
import { terminalDisplayCli, terminalCliFaviconUrls, terminalCliLabel, terminalCliMark } from "@picode/shared/domain/terminalCli.js";

// A loaded favicon is the identity, same as ProviderFace: the image fills
// the slot with no chip behind it. The boxed badge is only for fallback
// marks (and the unused labeled variant).
export default function TerminalCliBadge({ term, showLabel = false, decorative = false }) {
  const cli = terminalDisplayCli(term);
  const label = cli ? terminalCliLabel(cli) : "Terminal";
  const favicons = cli ? terminalCliFaviconUrls(cli) : [];
  const [failedCount, setFailedCount] = useState(0);
  const favicon = failedCount < favicons.length ? favicons[failedCount] : "";
  const showFavicon = Boolean(favicon);
  const failNext = () => setFailedCount((n) => n + 1);
  const named = decorative ? { alt: "", "aria-hidden": true } : { alt: "", title: label, "aria-label": label };

  if (showFavicon && !showLabel) {
    return (
      <img
        className="ws-face term-cli-face"
        src={favicon}
        onError={failNext}
        {...named}
      />
    );
  }

  const badgeClass =
    "term-cli-badge" +
    (showFavicon ? " has-favicon" : cli ? " cli-" + cli : " is-shell");
  return (
    <span className={badgeClass} title={decorative ? undefined : label} aria-label={decorative ? undefined : label} aria-hidden={decorative ? true : undefined}>
      {showFavicon ? (
        <img className="term-cli-favicon" src={favicon} alt="" onError={failNext} />
      ) : cli ? (
        <span className="term-cli-glyph" aria-hidden="true">{terminalCliMark(cli)}</span>
      ) : <IconTerminal size={15} />}
      {showLabel ? <span className="term-cli-label">{label}</span> : null}
    </span>
  );
}
