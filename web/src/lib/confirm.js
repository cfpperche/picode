export function askConfirm({ title, message, confirmLabel, danger } = {}) {
  return new Promise((resolve) => {
    window.dispatchEvent(new CustomEvent("picode-confirm", {
      detail: {
        title: title || "Confirm",
        message: message || "",
        confirmLabel: confirmLabel || "Continue",
        danger: !!danger,
        resolve,
      },
    }));
  });
}
