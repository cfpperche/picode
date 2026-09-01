import { createRoot } from "react-dom/client";
import { pickShell } from "./lib/shell.js";
import { overlayAudit } from "./lib/overlayAudit.js";
import { consoleEgg } from "./lib/consoleEgg.js";
import DesktopApp from "./desktop/App.jsx";
import MobileApp from "./mobile/App.jsx";
import "./index.css";

window.__picodeOverlayAudit = overlayAudit;
consoleEgg();

// No StrictMode: xterm + agent websockets must not double-mount.
const App = pickShell() === "mobile" ? MobileApp : DesktopApp;
createRoot(document.getElementById("root")).render(<App />);

// The service worker needs a secure context: HTTPS, or localhost (dev).
if ((location.protocol === "https:" || location.hostname === "localhost") && "serviceWorker" in navigator) {
  navigator.serviceWorker.register("/sw.js").then((reg) => reg.update()).catch(() => {});
  // A push notification tap (ADR-0047) asks the open window to navigate.
  navigator.serviceWorker.addEventListener("message", (e) => {
    if (e.data && e.data.type === "navigate" && typeof e.data.hash === "string") location.hash = e.data.hash;
  });
}
