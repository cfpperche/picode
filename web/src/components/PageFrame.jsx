import { IconBack } from "./Icons.jsx";

export default function PageFrame({ id, title, children, hidden }) {
  return (
    <section id={id} className="pane-view" hidden={hidden}>
      <div className="settings-wrap">
        <header className="settings-head">
          <a href="#/" className="btn btn-ghost btn-sm">
            <IconBack />
            Back
          </a>
          <h2>{title}</h2>
        </header>
        <div className="settings-card">{children}</div>
      </div>
    </section>
  );
}
