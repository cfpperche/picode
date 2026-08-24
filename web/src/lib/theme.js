const KEY = "picode-theme";

export function readThemeMode() {
  return localStorage.getItem(KEY) || "system";
}

export function resolvedTheme(mode) {
  if (mode === "light" || mode === "dark") return mode;
  return matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function applyTheme(mode) {
  const r = resolvedTheme(mode);
  document.documentElement.dataset.theme = r;
  document.documentElement.style.colorScheme = r;
}

export function persistTheme(mode) {
  localStorage.setItem(KEY, mode);
  applyTheme(mode);
}
