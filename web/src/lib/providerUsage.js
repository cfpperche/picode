export function showUsageButton(p) {
  return Boolean(p && p.signedIn && p.quotaKind && p.authType === p.quotaKind);
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
  if (report.status === "ok" && !(report.windows && report.windows.length)) {
    return { line: "No usage windows on this plan.", action: "retry" };
  }
  return { line: "", action: "" };
}
