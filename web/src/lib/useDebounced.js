import { useEffect, useState } from "react";

// The debounced twin of a fast-changing value (a controlled input, say).
export function useDebounced(value, ms = 220) {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return v;
}
