import { useEffect, useState } from "react";

// The sound twin of bad.component.jsx: every construct here must NOT be
// flagged. Declared-before-use, reads inside callbacks (they run after
// initialization), a shadowing inner block, hoisted names (function and
// var), member properties and object keys that merely carry the name, and
// a for-head `let` that scopes to the loop. Checker input only; excluded
// from every build and the tree scan.
export default function GoodComponent({ items, config }) {
  const earlyVar = legacy; // var hoists: undefined here, never a TDZ read
  var legacy = config.legacy;
  const [showFullUrl, setShowFullUrl] = useState(true);
  const label = showFullUrl ? "full url" : "origin only";
  const conf = { showFullUrl: true }; // object key, not a read
  const viaMember = config.showFullUrl; // member property, not a read
  let total = 0;
  for (let i = 0; i < items.length; i++) total += items[i].weight || 1;
  useEffect(() => {
    const t = setInterval(tick, 800);
    function tick() {
      console.log("polling", showFullUrl, total);
    }
    return () => clearInterval(t);
  }, [items.length, showFullUrl]);
  {
    let showFullUrl = false; // inner binding shadows; this read is its own
    label2(showFullUrl);
  }
  const caption = describe(showFullUrl); // hoisted function below
  function describe(v) {
    return v ? "full url" : "origin only";
  }
  return (
    <div className="tab" title={label} data-caption={caption} data-conf={conf.showFullUrl}>
      <button onClick={() => setShowFullUrl(!showFullUrl)}>flip</button>
      <span>
        {items.length} — {viaMember}
      </span>
    </div>
  );
}

function label2(v) {
  return Boolean(v);
}
