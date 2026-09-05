// One line of a unified diff. Shared by the chat's edit cards and the git
// graph's commit pane so both read the same way.
export default function DiffLine({ h }) {
  return (
    <div className={"diff-line " + h.kind}>
      <span className="diff-gutter">
        {h.kind === "add" ? "+" : h.kind === "del" ? "−" : h.kind === "gap" ? "·" : " "}
      </span>
      <span className="diff-text">{h.text}</span>
    </div>
  );
}
