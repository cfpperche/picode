import { useEffect, useState } from "react";

// Deliberate blank-window reproduction (the class behind main 4f1a68a7):
// both reads of `showFullUrl` sit before its `const` in the same render
// body, so the first render throws "Cannot access 'showFullUrl' before
// initialization" and the app mounts blank. The checker must flag exactly
// these two reads and nothing else in this file — the reads inside the
// effect callback and the onClick arrow run after initialization and are
// exempt. Checker input only; excluded from every build and the tree scan.
export default function BadComponent({ tabId }) {
  const id = tabId.slice(2);
  const label = showFullUrl ? "full url" : "origin only";
  useEffect(() => {
    const t = setInterval(tick, 800);
    function tick() {
      console.log("polling", showFullUrl, id);
    }
    return () => clearInterval(t);
  }, [id, showFullUrl]);
  const [showFullUrl, setShowFullUrl] = useState(true);
  return (
    <div className="tab" title={label}>
      <button onClick={() => setShowFullUrl(!showFullUrl)}>flip</button>
      <span>{id}</span>
    </div>
  );
}
