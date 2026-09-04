import { useState } from "react";
import { IconTerminal } from "./Icons.jsx";
import { terminalCli, terminalCliFaviconUrl, terminalCliLabel, terminalCliMark } from "../lib/terminalCli.js";

export default function TerminalCliBadge({ term, showLabel = false }) {
  const cli = terminalCli(term);
  const label = cli ? terminalCliLabel(cli) : "Terminal";
  const favicon = cli ? terminalCliFaviconUrl(cli) : "";
  const [failedFavicon, setFailedFavicon] = useState("");
  const showFavicon = favicon && failedFavicon !== favicon;
  return (
    <span className={"term-cli-badge" + (cli ? " cli-" + cli : " is-shell")} title={label} aria-label={label}>
      {showFavicon ? (
        <img className="term-cli-favicon" src={favicon} alt="" onError={() => setFailedFavicon(favicon)} />
      ) : cli ? (
        <span className="term-cli-glyph" aria-hidden="true">{terminalCliMark(cli)}</span>
      ) : <IconTerminal size={15} />}
      {showLabel ? <span className="term-cli-label">{label}</span> : null}
    </span>
  );
}
