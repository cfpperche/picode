export function askConfirm({ title, message, confirmLabel, danger, choices } = {}) {
  return new Promise((resolve) => {
    window.dispatchEvent(new CustomEvent("picode-confirm", {
      detail: {
        title: title || "Confirm",
        message: message || "",
        confirmLabel: confirmLabel || "Continue",
        danger: !!danger,
        choices: Array.isArray(choices) ? choices : [],
        resolve,
      },
    }));
  });
}

export function fmtBytes(n) {
  const x = Number(n) || 0;
  if (x < 1024) return x + " B";
  if (x < 1024 * 1024) {
    const kb = x / 1024;
    return (kb < 10 ? kb.toFixed(1) : Math.round(kb)) + " KB";
  }
  const mb = x / (1024 * 1024);
  return (mb < 10 ? mb.toFixed(1) : Math.round(mb)) + " MB";
}
