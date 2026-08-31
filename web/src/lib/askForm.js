/** Sequential extension_ui_request cards in one turn become one growing form. */

function isUser(it) {
  return it && it.kind === "block" && it.cls === "user";
}

function isAsk(it) {
  return it && it.kind === "ask";
}

function lastUserIndex(items) {
  let n = -1;
  for (let i = 0; i < items.length; i++) {
    if (isUser(items[i])) n = i;
  }
  return n;
}

function stepFromDialog(d, status) {
  return {
    id: d.id,
    method: d.method || "select",
    title: d.title || "",
    message: d.message || "",
    options: d.options || [],
    placeholder: d.placeholder || "",
    prefill: d.prefill || "",
    status: status || "open",
    answer: "",
  };
}

function askToStep(it) {
  return {
    id: it.id,
    method: it.method || "select",
    title: it.title || "",
    message: it.message || "",
    options: it.options || [],
    placeholder: it.placeholder || "",
    prefill: it.prefill || "",
    status: it.status || "open",
    answer: it.answer || "",
  };
}

function stepsOf(it) {
  if (it.steps && it.steps.length) return it.steps.slice();
  return [askToStep(it)];
}

/** Index of the ask card that should grow, or -1. */
export function stitchIndex(items) {
  const list = items || [];
  const from = lastUserIndex(list);
  for (let i = list.length - 1; i > from; i--) {
    const it = list[i];
    if (isAsk(it)) {
      if (it.status === "cancelled" || it.status === "timeout") return -1;
      return i;
    }
    if (it.kind === "block" || it.kind === "tool") return -1;
  }
  return -1;
}

function cardFromDialog(d, status) {
  const step = stepFromDialog(d, status);
  return {
    kind: "ask",
    id: d.id,
    method: step.method,
    title: step.title,
    message: step.message,
    options: step.options,
    placeholder: step.placeholder,
    prefill: step.prefill,
    timeout: d.timeout || 0,
    status: status || "open",
    answer: "",
    steps: [step],
    ts: Date.now(),
  };
}

function applyDialog(card, d, status) {
  return {
    ...card,
    id: d.id,
    method: d.method || card.method,
    title: d.title || "",
    message: d.message || "",
    options: d.options || [],
    placeholder: d.placeholder || "",
    prefill: d.prefill || "",
    timeout: d.timeout || 0,
    status: status || "open",
  };
}

export function putAsk(items, dialog, status) {
  if (!dialog || !dialog.id) return items || [];
  const nextStatus = status || "open";
  const cur = items || [];
  for (let i = 0; i < cur.length; i++) {
    const it = cur[i];
    if (!isAsk(it)) continue;
    const steps = stepsOf(it);
    const si = steps.findIndex((s) => s.id === dialog.id);
    if (it.id !== dialog.id && si < 0) continue;
    const copy = cur.slice();
    if (si >= 0) {
      steps[si] = { ...steps[si], ...stepFromDialog(dialog, nextStatus), answer: steps[si].answer };
      copy[i] = { ...applyDialog(it, dialog, nextStatus), steps };
    } else {
      copy[i] = { ...applyDialog(it, dialog, nextStatus), steps };
    }
    return copy;
  }
  const at = stitchIndex(cur);
  if (at >= 0) {
    const it = cur[at];
    const steps = stepsOf(it);
    steps.push(stepFromDialog(dialog, nextStatus));
    const copy = cur.slice();
    copy[at] = { ...applyDialog(it, dialog, nextStatus), steps };
    return copy;
  }
  return [...cur, cardFromDialog(dialog, nextStatus)];
}

export function answerAsk(items, id, answer, cancelled) {
  const status = cancelled ? "cancelled" : "answered";
  const text = cancelled ? "Cancelled" : (answer || "Answered");
  return (items || []).map((it) => {
    if (!isAsk(it) || it.status !== "open") return it;
    const steps = stepsOf(it);
    let hit = it.id === id;
    const nextSteps = steps.map((s) => {
      if (s.id !== id || s.status !== "open") return s;
      hit = true;
      return { ...s, status, answer: text };
    });
    if (!hit) return it;
    return { ...it, status, answer: text, steps: nextSteps };
  });
}

export function timeoutAsk(items, id) {
  return (items || []).map((it) => {
    if (!isAsk(it) || it.status !== "open") return it;
    const steps = stepsOf(it);
    let hit = it.id === id;
    const nextSteps = steps.map((s) => {
      if (s.id !== id || s.status !== "open") return s;
      hit = true;
      return { ...s, status: "timeout", answer: "Timed out" };
    });
    if (!hit) return it;
    return { ...it, status: "timeout", answer: "Timed out", steps: nextSteps };
  });
}

export function cancelOpenAsks(items) {
  return (items || []).map((it) => {
    if (!isAsk(it) || it.status !== "open") return it;
    const steps = stepsOf(it).map((s) => (
      s.status === "open" ? { ...s, status: "cancelled", answer: "Cancelled" } : s
    ));
    return { ...it, status: "cancelled", answer: "Cancelled", steps };
  });
}

/** True when the latest ask in this turn is answered (form finished or between steps). */
export function askJustAnswered(items) {
  const list = items || [];
  for (let i = list.length - 1; i >= 0; i--) {
    if (isAsk(list[i])) return list[i].status === "answered";
    if (isUser(list[i]) || list[i].kind === "block" || list[i].kind === "tool") return false;
  }
  return false;
}
