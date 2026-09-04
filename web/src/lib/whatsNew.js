export const WHATS_NEW_SEEN_KEY = "picode-whats-new-seen";
export const MAX_RELEASES = 3;
export const MAX_HIGHLIGHTS = 9;

// Release tags are deliberately kept to numeric major.minor.patch values.
// Build metadata and a leading v are accepted because /api/version exposes
// both release identities and source-build identities.
export function parseVersion(value) {
  const raw = String(value || "").trim().replace(/^v/i, "").split(/[+-]/, 1)[0];
  const parts = raw.split(".");
  if (parts.length !== 3 || parts.some((p) => !/^\d+$/.test(p))) return null;
  return parts.map((p) => Number(p));
}

export function compareVersions(a, b) {
  const av = parseVersion(a);
  const bv = parseVersion(b);
  if (!av || !bv) return null;
  for (let i = 0; i < 3; i++) {
    if (av[i] !== bv[i]) return av[i] > bv[i] ? 1 : -1;
  }
  return 0;
}

function validEntry(entry) {
  return !!entry && !!parseVersion(entry.version) && Array.isArray(entry.highlights);
}

function sortNewestFirst(entries) {
  return entries.slice().sort((a, b) => compareVersions(b.version, a.version) || 0);
}

// Select published notes at or below the running semver. When seen is set,
// only releases newer than that browser's last acknowledged version remain.
// The limits keep a long-upgraded browser from opening an unbounded wall.
export function selectReleaseNotes(entries, current, seen = "") {
  const currentVersion = parseVersion(current);
  if (!currentVersion) return [];
  const seenVersion = seen ? parseVersion(seen) : null;
  const eligible = (Array.isArray(entries) ? entries : [])
    .filter(validEntry)
    .filter((entry) => {
      const relation = compareVersions(entry.version, current);
      if (relation == null || relation > 0) return false;
      return !seenVersion || compareVersions(entry.version, seen) > 0;
    });
  const releases = sortNewestFirst(eligible).slice(0, MAX_RELEASES);
  let highlights = 0;
  const out = [];
  for (const release of releases) {
    const selected = release.highlights.slice(0, Math.max(0, MAX_HIGHLIGHTS - highlights));
    if (!selected.length) continue;
    out.push({ ...release, highlights: selected });
    highlights += selected.length;
  }
  return out;
}

export function hasUnseenRelease({ release = false, current = "", seen = "", entries = [] } = {}) {
  return !!release && selectReleaseNotes(entries, current, seen).length > 0;
}

export function shouldAutoOpen({
  release = false,
  current = "",
  seen = "",
  entries = [],
  hasProductState = true,
  blocked = false,
} = {}) {
  if (!release || !hasProductState || blocked) return false;
  return hasUnseenRelease({ release, current, seen, entries });
}

export function readSeenVersion(store = typeof localStorage !== "undefined" ? localStorage : null) {
  if (!store) return "";
  try { return store.getItem(WHATS_NEW_SEEN_KEY) || ""; } catch { return ""; }
}

export function writeSeenVersion(version, store = typeof localStorage !== "undefined" ? localStorage : null) {
  if (!store || !parseVersion(version)) return false;
  try {
    store.setItem(WHATS_NEW_SEEN_KEY, String(version));
    return true;
  } catch {
    return false;
  }
}
