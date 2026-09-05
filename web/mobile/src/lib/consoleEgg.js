export function consoleEgg() {
  const accent = "#7c8cf8";
  const muted = "#9b9ba7";
  console.log(
    "%cπ%c PiCode\n%cThe browser is a door, not a cage.\n%cgithub.com/cfpperche/picode",
    "font: 700 36px/1 Georgia, ui-serif, serif; color: " + accent + ";",
    "font: 650 16px/36px -apple-system, 'Segoe UI', sans-serif; color: inherit;",
    "font: 12px/1.7 -apple-system, 'Segoe UI', sans-serif; color: " + muted + ";",
    "font: 11px/1.6 ui-monospace, monospace; color: " + muted + ";"
  );
}
