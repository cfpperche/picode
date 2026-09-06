// Collapsed workspace header shows one face strip for everything living in
// the workspace — managed agents first, then terminals (agent CLI favicons,
// shell marks). Pure merge so order and identity stay testable; slicing to
// the strip cap is faceSlice's job (shared/domain/providerIcon.js).
export function collapseFaceItems(agents, terms) {
  return [
    ...(agents || []).filter((a) => a && a.id).map((ag) => ({ kind: "agent", id: "a:" + ag.id, ag })),
    ...(terms || []).filter((t) => t && t.id).map((t) => ({ kind: "term", id: "t:" + t.id, term: t })),
  ];
}
