import { createRoot } from "react-dom/client";
import App from "./App.jsx";
import "./index.css";

// No StrictMode: xterm + agent websockets must not double-mount.
createRoot(document.getElementById("root")).render(<App />);

