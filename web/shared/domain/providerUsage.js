export function showUsageButton(p, account) {
  if (!p) return false;
  if (account) {
    return Boolean(account.quotaKind && account.type && account.quotaKind === account.type);
  }
  return Boolean(p.signedIn && p.quotaKind && p.authType === p.quotaKind);
}

export function usagePath(providerId, accountId) {
  const id = encodeURIComponent(providerId || "");
  const aid = accountId && accountId !== "live" ? encodeURIComponent(accountId) : "";
  if (aid) return "/api/providers/" + id + "/accounts/" + aid + "/usage";
  return "/api/providers/" + id + "/usage";
}

export function barTone(used) {
  if (used == null || Number.isNaN(used)) return "ok";
  if (used >= 90) return "bad";
  if (used >= 70) return "warn";
  return "ok";
}

export function formatReset(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  const hh = String(d.getHours()).padStart(2, "0");
  const mi = String(d.getMinutes()).padStart(2, "0");
  return mm + "/" + dd + " " + hh + ":" + mi;
}

export function resetLine(resets) {
  const n = Array.isArray(resets) ? resets.length : 0;
  if (!n) return "";
  const exp = resets[0] && resets[0].expiresAt ? formatReset(resets[0].expiresAt) : "";
  const count = n === 1 ? "1 reset available" : n + " resets available";
  return exp ? count + " · expires " + exp : count;
}

export function formatMoney(n, unit) {
  if (n == null || Number.isNaN(Number(n))) return "";
  const v = Number(n);
  if (unit === "usd") return "$" + v.toFixed(2);
  return unit ? v + " " + unit : String(v);
}

export function activeAccountLine(provider, report) {
  const label = (report && report.accountLabel) || activeLabel(provider);
  const kind = (report && report.authType) || (provider && provider.authType) || "";
  const method = kind === "oauth" ? "account" : kind === "api_key" ? "api key" : "";
  return [label, method].filter(Boolean).join(" · ");
}

function activeLabel(provider) {
  const accs = (provider && provider.accounts) || [];
  const a = accs.find((x) => x.active) || accs[0];
  return (a && a.label) || "Default";
}

export function usageCopy(report) {
  if (!report) return { line: "", action: "" };
  if (report.status === "auth_required") return { line: "Sign in again.", action: "signin" };
  if (report.status === "error") {
    return { line: report.error || "Couldn't load usage.", action: "retry" };
  }
  const has = (report.windows && report.windows.length) || (report.resets && report.resets.length);
  if (report.status === "ok" && !has) {
    return { line: "No usage windows on this plan.", action: "retry" };
  }
  return { line: "", action: "" };
}
