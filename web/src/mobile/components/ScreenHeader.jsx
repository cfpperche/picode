import { IconBack } from "../../components/Icons.jsx";

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
        <span className="m-head-name">{title}</span>
        {sub ? <span className="m-head-sub">{sub}</span> : null}
      </div>
      <div className="m-head-right">{right || null}</div>
    </header>
  );
}
