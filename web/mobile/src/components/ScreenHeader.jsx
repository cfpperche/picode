import { IconBack } from "./Icons.jsx";

// Pushed-screen header: Back on the left (deterministic — the parent
// tab), the title in the middle, one optional control on the right.
export default function ScreenHeader({ title, sub, onBack, right }) {
  return (
    <header className="m-head">
      {onBack ? (
        <button type="button" className="m-head-back" onClick={onBack} aria-label="Back">
          <IconBack size={18} />
        </button>
      ) : <span className="m-head-spacer" />}
      <div className="m-head-title">
        <h1 className="m-head-name" title={typeof title === "string" ? title : undefined}>{title}</h1>
        {sub ? <span className="m-head-sub">{sub}</span> : null}
      </div>
      <div className="m-head-right" data-align-row>{right || null}</div>
    </header>
  );
}
