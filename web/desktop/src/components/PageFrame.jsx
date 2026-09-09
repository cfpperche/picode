import { IconAgent, IconBack } from "./Icons.jsx";

// One page geometry for every route: .settings-wrap is fluid up to 1240px
// (the Agent CLIs standard, ADR-0103). Pages do not pick their own width.
export default function PageFrame({ id, title, context, children, hidden, embedded, className = "" }) {
  if (embedded) return <section id={id} hidden={hidden} aria-label={title}>{context && <p className="settings-ctx">{context}</p>}{children}</section>;
  return (
    <section id={id} className={"pane-view" + (className ? " " + className : "")} hidden={hidden}>
      <div className="settings-wrap">
        <header className="settings-head">
          <a href="#/" className="btn btn-ghost btn-sm">
            <IconBack />
            Back
          </a>
          <h2>{title}</h2>
        </header>
        <div className="settings-card">
          {context ? (
            <p className="settings-ctx" title={context}>
              <IconAgent />
              <span>{context}</span>
            </p>
          ) : null}
          {children}
        </div>
      </div>
    </section>
  );
}
