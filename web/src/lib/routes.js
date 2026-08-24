// Hash routes. Settings is PiCode-the-product. Pi-facing surfaces
// get their own paths (docs/architecture.md — Application routes).
export const ROUTES = {
  workspace: "/",
  settings: "/settings",
  providers: "/providers",
  mcps: "/mcps",
  devices: "/devices",
};

export function parseRoute(hash) {
  const h = (hash || location.hash || "").replace(/^#/, "") || "/";
  if (h === "/settings") return "settings";
  if (h === "/providers") return "providers";
  if (h === "/mcps") return "mcps";
  if (h === "/devices") return "devices";
  return "workspace";
}

export function go(name) {
  const path = ROUTES[name] || "/";
  location.hash = path === "/" ? "#/" : "#" + path;
}
