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
          <div className="settings-head-text">
            <h2>{title}</h2>
            {context ? <p className="settings-head-ctx" title={context}>{context}</p> : null}
          </div>
        </header>
        <div className="settings-card">{children}</div>
      </div>
    </section>
  );
}
