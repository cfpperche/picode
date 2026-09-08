// Which folders the window counts (ADR-0097). Same native-radio mechanics
// as DateRangePicker, so the two controls in the dashboard head share one
// rhythm instead of inventing a second segmented control.
//
// "This machine" is the default: the v1 dashboard counted every session
// wherever it ran, and defaulting to the narrower scope would silently drop
// rows the surface has always shown.
const OPTIONS = [
  { id: "machine", label: "This machine" },
  { id: "picode", label: "PiCode" },
];

export default function ScopePicker({ value, onChange }) {
  return (
    <div className="dash-range dash-scope" role="radiogroup" aria-label="Scope">
      {OPTIONS.map((opt) => (
        <label key={opt.id} className="dash-range-opt">
          <input
            type="radio"
            name="dash-scope"
            value={opt.id}
            checked={value === opt.id}
            onChange={() => onChange(opt.id)}
          />
          <span className="dash-range-face">{opt.label}</span>
        </label>
      ))}
    </div>
  );
}
