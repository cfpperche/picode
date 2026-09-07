// Word-wise list matcher shared by the app list searches (mobile More screen,
// desktop user menu). Case- and accent-insensitive; every query word must
// appear somewhere in the joined values.
const normalize = value => String(value || "").normalize("NFKD").replace(/\p{M}/gu, "").toLocaleLowerCase();

export function matchesListSearch(query, ...values) {
  const words = normalize(query).trim().split(/\s+/).filter(Boolean);
  const text = values.map(normalize).join(" ");
  return words.every(word => text.includes(word));
}
