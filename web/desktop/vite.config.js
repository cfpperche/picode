import { applicationConfig } from "../tools/vite-config.mjs";

// The desktop app's own bundle (ADR-0122): the composition the Windows
// shell loads. It renders the browser app's App with the shell chrome
// enabled — nothing here is served to a browser.
export default applicationConfig("desktop", 5176);
