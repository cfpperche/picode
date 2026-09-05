import { IconAgent } from "./Icons.jsx";

export default function PageFrame({ id, title, context, children, hidden, wide }) {
  return (
    <section id={id} className="pane-view" hidden={hidden} aria-label={title}>
      <div className={"settings-wrap" + (wide ? " wide" : "")}>
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
