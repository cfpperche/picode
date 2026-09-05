export function askPrompt({ title, message, defaultValue, confirmLabel } = {}) {
  return new Promise((resolve) => {
    window.dispatchEvent(new CustomEvent("picode-prompt", {
      detail: {
        title: title || "Name",
        message: message || "",
        defaultValue: defaultValue || "",
        confirmLabel: confirmLabel || "Save",
        resolve,
      },
    }));
  });
}
