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
  return msg;
}

export function wsURL(path) {
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${location.host}${path}`;
}
