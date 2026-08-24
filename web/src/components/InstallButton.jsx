import { useEffect, useState } from "react";
import { canInstall, isStandalone, onInstallChange, promptInstall } from "../lib/install.js";

export default function InstallButton({ className = "btn btn-primary" }) {
  const [ready, setReady] = useState(canInstall);
  const [hint, setHint] = useState("");

  useEffect(() => onInstallChange(setReady), []);

  if (isStandalone()) return null;

  async function onClick() {
    const r = await promptInstall();
    if (r.ok) return;
    if (r.reason === "unavailable") {
      const ios = /iPhone|iPad|iPod/.test(navigator.userAgent);
      setHint(ios
        ? "iPhone: Share → Add to Home Screen (Safari only)."
        : "Chrome will offer Install when this site qualifies. Use HTTPS and visit twice if needed.");
    }
  }

  return (
    <div>
      <button type="button" className={className} onClick={onClick}>
        {ready ? "Install app" : "Add to Home Screen"}
      </button>
      {hint ? <p className="m-empty" style={{ textAlign: "left", padding: "8px 0" }}>{hint}</p> : null}
    </div>
  );
}
