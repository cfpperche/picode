// What a sidebar list shows: its rows, its empty line, or — before the boot
// has read the fleet — a skeleton. Deciding "empty" from a list that has not
// been read yet showed "No workspaces yet." for seconds on every cold load.
export function sidebarBody(loaded, count) {
  if (count > 0) return "list";
  return loaded ? "empty" : "skeleton";
}
