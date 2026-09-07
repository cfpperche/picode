// Session-only drafts. Navigation can unmount a conversation without losing
// text or attachment data; a reload deliberately starts a new browser session.
export function createAgentDrafts() {
  const drafts = new Map();
  const listeners = new Set();
  const empty = Object.freeze({ text: "", kind: "prompt", images: [], error: "", sending: false, revision: 0 });
  const read = id => drafts.get(id) || empty;
  const emit = () => { for (const listener of listeners) listener(); };
  function patch(id, change, edited = true) {
    if (!id) return;
    const current = read(id);
    drafts.set(id, { ...current, ...change, revision: current.revision + Number(edited) });
    emit();
  }
  return {
    read,
    subscribe(listener) { listeners.add(listener); return () => listeners.delete(listener); },
    update(id, change) { patch(id, change); },
    async submit(id, send, text) {
      const current = read(id);
      if (!id || current.sending) return false;
      // Voice and slash commands may supply text before the next React render.
      if (text != null && text !== current.text) patch(id, { text });
      const submitted = read(id);
      if (!submitted.text.trim() && !submitted.images.length) return false;
      patch(id, { sending: true, error: "" }, false);
      try {
        const result = await send(submitted.text, submitted.images, submitted.kind);
        if (!result?.accepted) throw new Error(result?.error || "Message was not accepted.");
        const latest = read(id);
        // New typing or attachments belong to the next message, even when the
        // previous request finishes after leaving and returning to this route.
        patch(id, {
          sending: false,
          error: "",
          ...(latest.revision === submitted.revision ? { text: "", images: [] } : {}),
        }, false);
        return true;
      } catch (error) {
        patch(id, { sending: false, error: error?.message || "Message could not be sent." }, false);
        return false;
      }
    },
  };
}

export const agentDrafts = createAgentDrafts();

export function submissionKind(kind, { streaming, waiting } = {}) {
  const selected = kind || "prompt";
  return (streaming || waiting) && selected !== "steer" && selected !== "follow_up" ? "follow_up" : selected;
}

export function isSubmitKey(event) {
  return event.key === "Enter" && !!(event.ctrlKey || event.metaKey) && !event.shiftKey && !event.isComposing && event.keyCode !== 229;
}
