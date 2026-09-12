import { createRoot } from "react-dom/client";
import App from "@picode/browser/src/App.jsx";
import "@picode/shared/tokens/theme.css";

// The shell's composition: the browser app with the shell chrome on. The
// app bar, the window controls and the drag regions belong to the shell
// itself (ADR-0122); this entry only tells the app it lives in one.
createRoot(document.getElementById("root")).render(<App shellChrome />);
