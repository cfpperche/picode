import { IconFlask, IconInbox, IconGrid } from "./Icons.jsx";

// Manifest icon names → components (ADR-0036). Icons are host-owned: an
// app names one from this map and an unknown name falls back to a letter
// tile from the app's own name (the providerLetter pattern).
const APP_ICONS = {
  flask: IconFlask,
  inbox: IconInbox,
  grid: IconGrid,
};

export default function AppIcon({ name, label, size = 16 }) {
  const Icon = APP_ICONS[name];
  if (Icon) return <Icon size={size} />;
  const letter = String(label || "?").trim().charAt(0).toUpperCase() || "?";
  return <span className="app-icon-letter" style={{ fontSize: Math.round(size * 0.72) + "px" }} aria-hidden="true">{letter}</span>;
}
