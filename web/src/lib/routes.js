// Hash routes. Preferences is PiCode-the-product. Settings is pi (ADR-0012).
export const ROUTES = {
  workspace: "/",
  preferences: "/preferences",
  settings: "/settings",
  system: "/system",
  providers: "/providers",
  mcps: "/mcps",
  packages: "/packages",
  devices: "/devices",
};

export function parseRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  if (h === "/preferences") return "preferences";
  if (h === "/settings") return "settings";
  if (h === "/system") return "system";
  if (h === "/providers") return "providers";
  if (h === "/mcps") return "mcps";
  if (h === "/packages") return "packages";
  if (h === "/devices") return "devices";
  return "workspace";
}

export function go(name) {
  const path = ROUTES[name] || "/";
  location.hash = path === "/" ? "#/" : "#" + path;
}
