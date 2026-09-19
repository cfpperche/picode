// Launchable, installed catalog rows a workspace can bind as a managed
// CLI principal (ADR-0159 Fatia 3). Uninstalled and detect-only CLIs stay
// off the picker; Agent CLIs is the way to install them.

export function catalogForPrincipal(clis) {
  return (clis || []).filter((c) => c && c.id && c.launchable !== false && c.installed);
}
