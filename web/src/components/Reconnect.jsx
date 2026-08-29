import { useEffect, useState } from "react";
import PiSpinner from "./PiSpinner.jsx";

export default function Reconnect({ onReload }) {
  const [step, setStep] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setStep((s) => (s < 1 ? 1 : s)), 420);
    return () => clearInterval(t);
  }, []);
  const steps = [
    { id: "lost", label: "Lost connection" },
    { id: "wait", label: "Waiting for the server" },
  ];
  return (
    <div className="pkg-job reconnect-job" role="alertdialog" aria-modal="true" aria-labelledby="reconnect-title">
      <div className="pkg-job-card">
        <h3 id="reconnect-title">Reconnecting</h3>
        <ol className="pkg-job-steps">
          {steps.map((s, i) => {
            const st = i < step ? "done" : i === step ? "run" : "todo";
            return (
              <li key={s.id} className={"pkg-job-step " + st}>
                <span className="pkg-job-mark" aria-hidden="true">
                  {st === "run" ? <PiSpinner title="Working" /> : st === "done" ? "✓" : "○"}
                </span>
                <code>{s.label}</code>
              </li>
            );
          })}
        </ol>
        <div className="pkg-job-actions">
          <button type="button" className="btn btn-primary btn-sm" onClick={onReload}>Reload now</button>
        </div>
      </div>
    </div>
  );
}
