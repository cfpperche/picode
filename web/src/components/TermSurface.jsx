import ShellTerm from "./ShellTerm.jsx";

export default function TermSurface({ term, error }) {
  if (!term && !error) return null;
  return (
    <section className="term-surface" aria-label={term ? term.name : "Terminal"}>
      {error ? (
        <p className="file-pane-msg">
          {error}{" "}
          <a href="#/system">Open System</a>
        </p>
      ) : (
        <ShellTerm agentId={term.id} session={term.session} active />
      )}
    </section>
  );
}
