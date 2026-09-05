// A labelled radio card for theme-ish choices (app theme, terminal colors).
// Moved out of Settings.jsx when the terminal appearance controls moved to
// the terminal settings page — both surfaces render these.
export default function ThemeCard({ option, label, desc, active, onPick, icon }) {
  return (
    <button
      type="button"
      className="theme-card"
      data-theme-option={option}
      role="radio"
      aria-checked={active}
      onClick={() => onPick(option)}
    >
      {icon}
      <span className="theme-card-name">{label}</span>
      <span className="theme-card-desc">{desc}</span>
    </button>
  );
}
