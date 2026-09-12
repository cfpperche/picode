import { boot } from "@picode/browser/src/bootstrap.jsx";
import "./index.css";

// The shell's composition (ADR-0122): the browser app's boot with the
// shell chrome on. The app bar, the window controls and the drag regions
// belong to the shell itself; this entry only says the app lives in one.
boot(document.getElementById("root"), { shellChrome: true });
