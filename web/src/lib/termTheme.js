const KEY = "picode-term-theme";

export function readTermTheme() {
  return localStorage.getItem(KEY) === "light" ? "light" : "dark";
}

export function xtermTheme(mode) {
  if (mode === "light") {
    return {
      background: "#ffffff",
      foreground: "#16181d",
      cursor: "#2f6fed",
      selectionBackground: "#c9d7f5",
    };
  }
  return {
    background: "#0e0e11",
    foreground: "#ececf1",
    cursor: "#7c8cf8",
    selectionBackground: "#33467c",
  };
}

export function applyTermTheme(mode) {
  const v = mode === "light" ? "light" : "dark";
  document.documentElement.dataset.termTheme = v;
}

export function persistTermTheme(mode) {
  const v = mode === "light" ? "light" : "dark";
  localStorage.setItem(KEY, v);
  applyTermTheme(v);
  window.dispatchEvent(new Event("picode-term-theme"));
  return v;
}
