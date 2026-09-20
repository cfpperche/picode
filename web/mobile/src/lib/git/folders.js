// The phone's folder grouping for the Inspector's Changes list: one section
// per top-level directory, each header summing every file under it, the
// section's files listed flat (their own subfolder rides the row's small
// line, like GitFileRow prints it). A deep tree without a collapsible
// widget — the honest shape for a screen this wide.

export function folderSections(changes) {
  const byDir = new Map();
  for (const c of changes || []) {
    if (!c || !c.path) continue;
    const parts = c.path.split("/");
    const dir = parts.length > 1 ? parts[0] : "";
    let section = byDir.get(dir);
    if (!section) {
      section = { dir, name: dir, files: [], add: 0, del: 0 };
      byDir.set(dir, section);
    }
    section.files.push(c);
    section.add += Number(c.add) || 0;
    section.del += Number(c.del) || 0;
  }
  const sections = [...byDir.values()];
  const byName = (a, b) => (a.dir || "").localeCompare(b.dir || "", undefined, { sensitivity: "base", numeric: true });
  sections.sort(byName);
  for (const section of sections) {
    section.files.sort((a, b) => a.path.slice(a.path.lastIndexOf("/") + 1).localeCompare(b.path.slice(b.path.lastIndexOf("/") + 1), undefined, { sensitivity: "base", numeric: true }));
  }
  return sections;
}
