// What the grant's domain field means, said back to the person typing it.
//
// This is a description, not a second matcher: the rule the daemon and the
// shell enforce lives in internal/browser/domains.go and
// desktop-shell/src/origins.rs — written twice on purpose, tested against the
// same table. A hint here that disagreed with them would cost a wrong line of
// text, never a wrong grant. What it must never do is stay quiet about an
// entry that matches nothing: that was the `*` bug (accepted by the field, no
// host matched it, nothing said so).

// The owner's "any site": every host, still only over http/https.
export const ANY_SITE = "*";

// parseDomainField mirrors browserGrantSchema's transform (the schema stays the
// validator; this is the same reading so the hint describes what will be saved).
export function parseDomainField(text) {
  return String(text || "")
    .split(",")
    .map((raw) => {
      let host = String(raw).trim().toLowerCase().replace(/^https?:\/\//, "").split("/")[0];
      host = host.startsWith("[") ? host.slice(1).split("]")[0] : host.split(":")[0];
      return host;
    })
    .filter(Boolean);
}

// describeDomainEntry reads one entry the way the matchers do, in words.
export function describeDomainEntry(entry) {
  const want = String(entry || "").trim().toLowerCase();
  if (!want) return null;
  if (want === ANY_SITE) {
    return {
      kind: "any",
      label: "Any site",
      covers: "any http or https address, on any host",
      miss: "file:, data: and other schemes are never allowed",
    };
  }
  const starTail = want.startsWith("*.") ? want.slice(2) : null;
  const dotTail = want.startsWith(".") ? want.slice(1) : null;
  const suffix = starTail !== null ? starTail : dotTail;
  if (suffix && !/^\.+$/.test(suffix)) {
    const tld = suffix.includes(".") ? "" : ".";
    return {
      kind: "suffix",
      label: tld ? `Every ${tld}${suffix} site` : `${suffix} and its subdomains`,
      covers: tld ? `any host ending in ${tld}${suffix}` : `${suffix} itself and anything under it, like docs.${suffix}`,
      miss: tld ? `a name that merely ends the same, like not${tld}${suffix}` : `a name that merely ends the same, like not${suffix}`,
    };
  }
  if (want.startsWith("*") || want.startsWith(".")) {
    // `*example.com`, `*.`, a lone `.`: the matchers compare these literally
    // (or drop them), and no host is named that — entries that open nothing.
    return {
      kind: "dead",
      label: `${want} matches nothing`,
      covers: "",
      miss: `the matchers read this literally — write ${ANY_SITE} for any site, or *.example.com for that host and its subdomains`,
    };
  }
  return {
    kind: "host",
    label: `${want} only`,
    covers: `${want} on any port`,
    miss: `its subdomains — write *.${want} to include docs.${want}`,
  };
}

// describeDomainField is the field's whole hint: one description per entry, in
// the order they were typed.
export function describeDomainField(text) {
  return parseDomainField(text)
    .map((entry) => {
      const desc = describeDomainEntry(entry);
      return desc ? { entry, ...desc } : null;
    })
    .filter(Boolean);
}
