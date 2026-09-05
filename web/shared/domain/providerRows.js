// Pure helpers for the providers roster. Everything here is a formatting or
// filtering decision the view should not re-derive, and every one of them
// has one rule in common: a number we did not fetch is never rendered as a
// number (ADR-0031).

// STATE_* are the three things a quota row can honestly be.
export const STATE_LIVE = "live";
export const STATE_STALE = "stale";
export const STATE_NONE = "none";

// FRESH_SEC is how long a fetched window still counts as live. The refresher
// runs every 5 minutes, so anything inside that is what the server last saw.
export const FRESH_SEC = 5 * 60;

// usageKey addresses one roster row in the summary payload. The active slot
// of a provider with no vault rows has an empty account id, same as the API.
export function usageKey(providerId, accountId) {
  const aid = accountId && accountId !== "live" ? accountId : "";
  return String(providerId || "") + " " + aid;
}

// indexUsage turns the summary array into a lookup the row can hit directly.
export function indexUsage(entries) {
  const out = new Map();
  for (const e of entries || []) out.set(usageKey(e.provider, e.accountId), e);
  return out;
}

// quotaState says how much trust the row's number deserves. A window we
// cannot render — neither a percentage nor an amount — counts as nothing,
// so the row says "no plan windows" instead of showing an empty gauge.
export function quotaState(entry) {
  if (!entry || entry.status === "unknown" || !entry.status) return STATE_NONE;
  if (entry.status !== "ok") return STATE_NONE;
  if (barWindows(entry).length === 0 && moneyWindows(entry).length === 0) return STATE_NONE;
  return (entry.ageSec || 0) <= FRESH_SEC ? STATE_LIVE : STATE_STALE;
}

// quotaNote is the one line a row shows when it has no bar to show. It is
// the vendor's reason or our own "not fetched yet" — never a zero.
export function quotaNote(entry) {
  if (!entry || !entry.status || entry.status === "unknown") return "not checked";
  if (entry.status === "auth_required") return "sign in again";
  if (entry.status === "error") return entry.error || "couldn't load";
  if (entry.status === "unsupported") return "no plan windows";
  if (barWindows(entry).length === 0 && moneyWindows(entry).length === 0) return "no plan windows";
  return "";
}

// formatAge is "4m" / "2h" / "3d". Under a minute reads "now" — the number
// is what the server just fetched.
export function formatAge(sec) {
  const n = Number(sec);
  if (!Number.isFinite(n) || n < 60) return "now";
  if (n < 3600) return Math.floor(n / 60) + "m";
  if (n < 86400) return Math.floor(n / 3600) + "h";
  return Math.floor(n / 86400) + "d";
}

// barWindows are the windows worth a bar on a roster row: percentage ones
// only. Money left has no denominator, so it stays in the dialog rather
// than pretending to be a gauge.
export function barWindows(entry, max = 2) {
  const all = (entry && entry.windows) || [];
  return all.filter((w) => w.usedPercent != null).slice(0, max);
}

// moneyWindows are the ones with an amount left and no ceiling — prepaid
// credits, extra usage. They are real readings and belong on the row, as
// text: a balance with no denominator cannot honestly be a bar.
export function moneyWindows(entry, max = 1) {
  const all = (entry && entry.windows) || [];
  return all.filter((w) => w.usedPercent == null && w.remaining != null).slice(0, max);
}

// sourceLabel names where pi would read this provider's credential. Only an
// environment variable is worth saying out loud: it is the one source the
// user cannot change from this page.
export function sourceLabel(provider) {
  if (!provider) return "";
  if (provider.source === "environment") return provider.envVar || "environment";
  return "";
}

// identityLine is what the vendor says this login is. The label is the
// user's alias and is rendered separately, so this never repeats it.
export function identityLine(account, entry) {
  const email = (account && account.email) || (entry && entry.email) || "";
  const plan = (entry && entry.plan) || (account && account.plan) || "";
  return [email, plan].filter(Boolean).join(" · ");
}

// blastRadius is the sentence Sign out has to be able to say. Counts come
// from the catalog; zero dependents means the plain warning is enough.
export function blastRadius(provider) {
  const a = (provider && provider.agents) || 0;
  const t = (provider && provider.automations) || 0;
  const parts = [];
  if (a) parts.push(a === 1 ? "1 agent" : a + " agents");
  if (t) parts.push(t === 1 ? "1 automation" : t + " automations");
  if (!parts.length) return "";
  const verb = a + t === 1 ? " uses " : " use ";
  return parts.join(" and ") + verb + "this provider.";
}

// matchesQuery filters the roster. It searches the provider id, every
// account label and every known email, so "work" finds the row you named
// Work and "@company" finds the account you did not.
export function matchesQuery(provider, query) {
  const q = String(query || "").trim().toLowerCase();
  if (!q) return true;
  if (String(provider.id || "").toLowerCase().includes(q)) return true;
  for (const a of provider.accounts || []) {
    if (String(a.label || "").toLowerCase().includes(q)) return true;
    if (String(a.email || "").toLowerCase().includes(q)) return true;
    if (String(a.plan || "").toLowerCase().includes(q)) return true;
  }
  return false;
}

// spendByProvider maps the dashboard's own 7-day aggregate onto the roster.
// It is our number, computed from session files, not a vendor's.
export function spendByProvider(stats) {
  const out = new Map();
  for (const b of (stats && stats.byProvider) || []) {
    if (!b || !b.provider) continue;
    out.set(b.provider, Number(b.cost) || 0);
  }
  return out;
}

// formatSpend keeps sub-cent noise off the row: real money or nothing.
export function formatSpend(n) {
  const v = Number(n);
  if (!Number.isFinite(v) || v < 0.01) return "";
  return "$" + v.toFixed(2);
}
