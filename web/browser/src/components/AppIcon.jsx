import { useEffect, useState } from "react";
import { IconCanvas, IconFlask, IconGlobe, IconInbox, IconGrid, IconPackage, IconTmux } from "./Icons.jsx";

// Manifest icon names → components (ADR-0036). Icons are host-owned: an
// app names one from this map and an unknown name falls back to a letter
// tile from the app's own name (the providerLetter pattern).
const APP_ICONS = {
  flask: IconFlask,
  inbox: IconInbox,
  grid: IconGrid,
  box: IconPackage,
  canvas: IconCanvas,
  tmux: IconTmux,
  globe: IconGlobe,
};

export default function AppIcon({ name, label, size = 16, iconUrl }) {
  const [failed, setFailed] = useState(false);
  useEffect(() => { setFailed(false); }, [iconUrl]);
  if (iconUrl && !failed) {
    return (
      <img
        src={iconUrl}
        width={size} height={size}
        alt=""
        className="app-icon-img"
        onError={() => setFailed(true)}
      />
    );
  }
  const Icon = APP_ICONS[name];
  if (Icon) return <Icon size={size} />;
  const letter = String(label || "?").trim().charAt(0).toUpperCase() || "?";
  return <span className="app-icon-letter" style={{ fontSize: Math.round(size * 0.72) + "px" }} aria-hidden="true">{letter}</span>;
}
