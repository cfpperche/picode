// ADR-0057: tool-agnostic live preview. A tool may surface what it is
// looking at by emitting `details.preview` alongside its result (persisted,
// so replay renders it) and its partial results (live frames while the tool
// runs). PiCode core knows the shape, never the tool name.
//
//   details.preview := { image: string (data URI or https URL),   // required
//                        url?: string, title?: string, ts?: number }
//
// Frames replace each other — the renderer never accumulates them.
export function previewFromDetails(details) {
  const p = details && typeof details === "object" ? details.preview : null;
  if (!p || typeof p.image !== "string" || !p.image) return null;
  return {
    image: p.image,
    url: typeof p.url === "string" ? p.url : "",
    title: typeof p.title === "string" ? p.title : "",
  };
}
