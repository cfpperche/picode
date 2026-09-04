import { AlertTriangle, ArrowLeft, Check, LoaderCircle } from "lucide-react";

const STEPS = [
  { phase: "receiving", label: "Receiving" },
  { phase: "processing", label: "Processing" },
  { phase: "restoring", label: "Returning" },
];

function phaseIndex(phase) {
  if (phase === "done") return STEPS.length;
  return Math.max(0, STEPS.findIndex((step) => step.phase === phase));
}

export default function BurstSurface({ burst, agentName, onCancel }) {
  if (!burst) return null;
  const failed = burst.phase === "failed";
  const done = burst.phase === "done";
  const restarting = burst.phase === "restoring" && burst.terminalUnavailable;
  const active = phaseIndex(burst.phase);
  const buttonLabel = restarting ? "Restarting terminal" : failed || done ? "Return to terminal" : "Cancel and return";
  const emptyMessage = failed
    ? (burst.error || "The reply could not finish.")
    : done
      ? "The reply finished without a text response."
      : "The response will appear here while the terminal session continues.";

  return (
    <section className={"term-surface burst-surface burst-" + burst.phase} aria-label="Terminal reply status">
      <div className="burst-card" aria-live="polite" aria-busy={!failed && !done}>
        <div className="burst-kicker">{agentName ? agentName + " · " : ""}Terminal reply</div>

        <ol className="burst-steps" aria-label="Reply progress">
          {STEPS.map((step, i) => {
            const complete = done || (!failed && i < active);
            const current = !done && !failed && i === active;
            return (
              <li key={step.phase} className={(complete ? "is-complete " : "") + (current ? "is-current" : "")}>
                <span className="burst-step-dot" aria-hidden="true">
                  {complete ? <Check size={12} /> : current ? <LoaderCircle size={13} /> : null}
                </span>
                <span>{step.label}</span>
              </li>
            );
          })}
        </ol>

        <div className="burst-status">
          {failed ? <AlertTriangle size={18} aria-hidden="true" /> : done ? <Check size={18} aria-hidden="true" /> : <span className="burst-pulse" aria-hidden="true" />}
          <h2>{failed ? "Reply stopped" : done ? "Reply complete" : (burst.activity || "Processing your reply")}</h2>
        </div>

        {burst.output ? (
          <div className="burst-response" aria-label="Agent response">{burst.output}</div>
        ) : (
          <p className="burst-empty">{emptyMessage}</p>
        )}
        {failed && burst.output ? <p className="burst-error">{burst.error || "The reply could not finish."}</p> : null}

        <button type="button" className="btn btn-sm burst-return" onClick={onCancel} disabled={restarting}>
          <ArrowLeft size={14} />
          {buttonLabel}
        </button>
      </div>
    </section>
  );
}
