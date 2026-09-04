import { useState } from "react";
import { IconTerminal } from "./Icons.jsx";
import { terminalCli, terminalCliFaviconUrls, terminalCliLabel, terminalCliMark } from "../lib/terminalCli.js";

// A loaded favicon renders bare — same treatment as workspace favicons —
// while the boxed badge is reserved for the fallback marks.
export default function TerminalCliBadge({ term, showLabel = false }) {
  const cli = terminalCli(term);
  const label = cli ? terminalCliLabel(cli) : "Terminal";
  const favicons = cli ? terminalCliFaviconUrls(cli) : [];
  const [failedCount, setFailedCount] = useState(0);
  const favicon = failedCount < favicons.length ? favicons[failedCount] : "";
  const showFavicon = Boolean(favicon);
  const badgeClass =
    "term-cli-badge" +
    (showFavicon ? " has-favicon" : "") +
    (cli ? " cli-" + cli : " is-shell");
  return (
    <span className={badgeClass} title={label} aria-label={label}>
      {showFavicon ? (
        <img className="term-cli-favicon" src={favicon} alt="" onError={() => setFailedCount(failedCount + 1)} />
      ) : cli ? (
        <span className="term-cli-glyph" aria-hidden="true">{terminalCliMark(cli)}</span>
      ) : <IconTerminal size={15} />}
      {showLabel ? <span className="term-cli-label">{label}</span> : null}
    </span>
  );
}
