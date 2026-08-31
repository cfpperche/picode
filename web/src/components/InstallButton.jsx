import { useEffect, useState } from "react";
import { canInstall, isStandalone, onInstallChange, promptInstall } from "../lib/install.js";
import { IconDownload } from "./Icons.jsx";

function isIOS() {
  return /iPhone|iPad|iPod/.test(navigator.userAgent)
    || (navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1);
}

export default function InstallButton({ className = "btn btn-primary" }) {
  const [ready, setReady] = useState(canInstall);
  const [coach, setCoach] = useState(false);

  useEffect(() => onInstallChange(setReady), []);

  if (isStandalone()) return null;

  async function onClick() {
    if (isIOS()) {
      setCoach(true);
      return;
    }
    const r = await promptInstall();
    if (!r.ok && r.reason === "unavailable") setCoach(true);
  }

  return (
    <div>
      <button type="button" className={className} onClick={onClick}>
        <IconDownload /> {ready ? "Install app" : "Add to Home Screen"}
      </button>
      {coach && isIOS() && (
        <div className="a2hs-coach" onClick={() => setCoach(false)}>
          <div className="a2hs-card" onClick={(e) => e.stopPropagation()}>
            <p>Safari only. Tap the <b>Share</b> icon in the toolbar, then <b>Add to Home Screen</b>.</p>
            <button type="button" className="btn btn-sm" onClick={() => setCoach(false)}>OK</button>
          </div>
          <div className="a2hs-arrow" aria-hidden="true">
            <span>Share</span>
            <svg width="22" height="36" viewBox="0 0 22 36" fill="none" stroke="currentColor" strokeWidth="2"><path d="M11 2v28M4 23l7 9 7-9"/></svg>
          </div>
        </div>
      )}
    </div>
  );
}
