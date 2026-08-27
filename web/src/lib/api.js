export async function api(path, opts) {
  const res = await fetch(path, opts);
  if (!res.ok) {
    let msg = res.statusText;
    try { msg = (await res.json()).error || msg; } catch { /* keep statusText */ }
    throw new Error(msg);
  }
  if (res.status === 204) return null;
  return res.json();
}

export function humanizeError(msg) {
  if (/no such file or directory|not a directory/i.test(msg)) {
    return "That folder doesn't exist — check the path and try again.";
  }
  if (/already exists/i.test(msg)) {
    return "This project folder is already added.";
  }
  if (/name is required/i.test(msg)) {
    return "Give the workspace a name.";
  }
  if (/can't find pane|can't find session|tmux send-keys/i.test(msg)) {
    return "Couldn't reach the terminal. Open Terminal and try again.";
  }
  if (/already compacted/i.test(msg)) {
    return "Nothing left to compact.";
  }
  if (/CreditsError|Insufficient balance|out of credit|out of extra usage/i.test(msg)) {
    return "This provider is out of credit. Add billing or switch provider.";
  }
  if (/\b401\b|Unauthorized|invalid api key|authentication/i.test(msg)) {
    return "Not authorized. Sign in again or check this provider's key.";
  }
  if (/overloaded|\b529\b/i.test(msg)) {
    return "The provider is overloaded. Wait a moment and try again.";
  }
  if (/rate.?limit|\b429\b/i.test(msg)) {
    return "Rate limited. Wait and retry.";
  }
  if (/dynamic client (auth(?:orization)? )?registration/i.test(msg)) {
    return "This server cannot Sign in from PiCode. It needs its own app login.";
  }
  const compact = msg.match(/^rpc: compact failed:\s*(.*)/i);
  if (compact) return compact[1];
  return msg;
}

export function wsURL(path) {
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${location.host}${path}`;
}
