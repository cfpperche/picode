import { measured, REPORTED, PARTIAL, NOT_REPORTED, ESTIMATED } from "@picode/shared/domain/dashboardStats.js";

// What each agent CLI can and cannot tell this dashboard (ADR-0097).
//
// This panel is the reason every other number on the surface can be read at
// face value. Nine vendors record wildly different things — Claude Code
// counts lines changed and never quota, Codex reports quota and never a
// price, Grok writes prompt history and nothing else — and a dashboard that
// silently summed whatever it found would be confidently wrong. So the
// blind spots are rendered as data, next to the numbers they qualify,
// rather than described in a doc nobody opens.
const SIGNALS = [
  { id: "cost", label: "Cost" },
  { id: "tokens", label: "Tokens" },
  { id: "turns", label: "Turns" },
  { id: "tools", label: "Tools" },
  { id: "errors", label: "Errors" },
  { id: "impact", label: "Lines" },
  { id: "timing", label: "Time" },
  { id: "limits", label: "Limits" },
];

const MARKS = {
  [REPORTED]: { glyph: "●", title: "recorded by this CLI" },
  [PARTIAL]: { glyph: "◑", title: "recorded for some sessions only" },
  [NOT_REPORTED]: { glyph: "—", title: "this CLI does not record it" },
  // Cost only (ADR-0185): the CLI prices nothing, PiCode priced its turns.
  [ESTIMATED]: { glyph: "~", title: "not recorded by this CLI — estimated by PiCode at list price" },
};
const UNKNOWN = { glyph: "?", title: "could not be read on this machine" };

export default function CoveragePanel({ coverage }) {
  const rows = (coverage || []).filter((r) => r && r.cli);
  if (rows.length === 0) {
    return <p className="dash-empty">No agent CLI reported this period.</p>;
  }
  return (
    <>
      <table className="dash-coverage">
        <thead>
          <tr>
            <th scope="col" className="dash-coverage-cli">CLI</th>
            {SIGNALS.map((s) => (
              <th key={s.id} scope="col">{s.label}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.cli}>
              <th scope="row" className="dash-coverage-cli">{r.label || r.cli}</th>
              {SIGNALS.map((s) => {
                const state = (r.signals && r.signals[s.id]) || "unavailable";
                const mark = MARKS[state] || UNKNOWN;
                return (
                  <td key={s.id} className={measured(state) ? "is-on" : "is-off"}>
                    <span title={(r.label || r.cli) + " · " + s.label + ": " + mark.title}>{mark.glyph}</span>
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
      {rows.some((r) => r.note) ? (
        <ul className="dash-coverage-notes">
          {rows.filter((r) => r.note).map((r) => (
            <li key={r.cli}><span className="dash-coverage-note-cli">{r.label || r.cli}</span> {r.note}</li>
          ))}
        </ul>
      ) : null}
    </>
  );
}
