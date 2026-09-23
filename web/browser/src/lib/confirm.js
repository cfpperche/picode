// feedback (optional) is the removal dialog's question (ADR-0194):
// { taxonomy } from the cleanup preview. With it, a confirm resolves
// { ...choices, feedback: { draft, shown, stopAsking } }.
export function askConfirm({ title, message, confirmLabel, danger, choices, feedback } = {}) {
  return new Promise((resolve) => {
    window.dispatchEvent(new CustomEvent("picode-confirm", {
      detail: {
        title: title || "Confirm",
        message: message || "",
        confirmLabel: confirmLabel || "Continue",
        danger: !!danger,
        choices: Array.isArray(choices) ? choices : [],
        feedback: feedback && feedback.taxonomy ? feedback : null,
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
