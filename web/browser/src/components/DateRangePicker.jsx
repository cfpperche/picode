import { rangeLabel } from "@picode/shared/domain/dashboardStats.js";

const OPTIONS = ["today", "7d", "30d", "all"];

// Native radio segmented control, same mechanics as .termset-seg /
// .create-seg elsewhere in this app: the input covers its own segment so
// the hit area is the whole label, not a 1px corner.
// name keeps two pickers on screen apart: radios that share a name are one
// group across the whole document, so the Outcomes page passes its own.
export default function DateRangePicker({ value, onChange, name = "dash-range" }) {
  return (
    <div className="dash-range" role="radiogroup" aria-label="Date range">
      {OPTIONS.map((opt) => (
        <label key={opt} className="dash-range-opt">
          <input
            type="radio"
            name={name}
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
