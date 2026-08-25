import { createRoot } from "react-dom/client";
import { pickShell } from "./lib/shell.js";
import { overlayAudit } from "./lib/overlayAudit.js";
import DesktopApp from "./desktop/App.jsx";
import MobileApp from "./mobile/App.jsx";
import "./index.css";

window.__picodeOverlayAudit = overlayAudit;

// No StrictMode: xterm + agent websockets must not double-mount.
const App = pickShell() === "mobile" ? MobileApp : DesktopApp;
createRoot(document.getElementById("root")).render(<App />);

if (location.protocol === "https:" && "serviceWorker" in navigator) {
  navigator.serviceWorker.register("/sw.js").then((reg) => reg.update()).catch(() => {});
}
