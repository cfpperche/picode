import { IconBack } from "./Icons.jsx";

export default function PageFrame({ id, title, context, children, hidden, wide }) {
  return (
    <section id={id} className="pane-view" hidden={hidden}>
      <div className={"settings-wrap" + (wide ? " wide" : "")}>
        <header className="settings-head">
          <a href="#/" className="btn btn-ghost btn-sm">
            <IconBack />
            Back
          </a>
          <h2>{title}</h2>
        </header>
        <div className="settings-card">
          {context ? <p className="settings-ctx" title={context}>{context}</p> : null}
          {children}
        </div>
      </div>
    </section>
  );
}
