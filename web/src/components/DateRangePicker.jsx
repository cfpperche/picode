import { rangeLabel } from "../lib/dashboardStats.js";

const OPTIONS = ["today", "7d", "30d", "all"];

// Native radio segmented control, same mechanics as .termset-seg /
// .create-seg elsewhere in this app: the input covers its own segment so
// the hit area is the whole label, not a 1px corner.
export default function DateRangePicker({ value, onChange }) {
  return (
    <div className="dash-range" role="radiogroup" aria-label="Date range">
      {OPTIONS.map((opt) => (
        <label key={opt} className="dash-range-opt">
          <input
            type="radio"
            name="dash-range"
            value={opt}
            checked={value === opt}
            onChange={() => onChange(opt)}
          />
          <span className="dash-range-face">{rangeLabel(opt)}</span>
        </label>
      ))}
    </div>
  );
}
