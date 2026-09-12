import { createRoot } from "react-dom/client";
import { overlayAudit } from "@picode/shared/domain/overlayAudit.js";
import "./index.css";
import App from "./App.jsx";
import PairingScreen from "./components/PairingScreen.jsx";
import { consoleEgg } from "./lib/consoleEgg.js";
import { installHashGuard } from "./lib/hashGuard.js";

// The one boot path for both entries (ADR-0122): /browser/ renders it plain,
// /desktop/ renders it with the shell chrome on. No StrictMode: xterm and
// agent websockets must not double-mount.
export function boot(rootEl, { shellChrome = false } = {}) {
  window.__picodeOverlayAudit = overlayAudit;
  consoleEgg();
  installHashGuard();
  createRoot(rootEl).render(<App shellChrome={shellChrome} />);

  // The pairing screen (ADR-0049) lives in its own root: whatever the shell
  // does when every API call answers 401, the way in must still render.
  const pairRoot = document.createElement("div");
  pairRoot.id = "pair";
  document.body.appendChild(pairRoot);
  createRoot(pairRoot).render(<PairingScreen />);

  // The service worker needs a secure context: HTTPS, or localhost (dev).
  if (import.meta.env.PROD && (location.protocol === "https:" || location.hostname === "localhost") && "serviceWorker" in navigator) {
    navigator.serviceWorker.register("/sw.js").then((reg) => reg.update()).catch(() => {});
    // A push notification tap (ADR-0047) asks the open window to navigate.
    navigator.serviceWorker.addEventListener("message", (e) => {
      if (e.data && e.data.type === "navigate" && typeof e.data.hash === "string") location.hash = e.data.hash;
    });
  }
}
