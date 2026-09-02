// Providers a form should offer: the ones with credentials (signedIn),
// plus whatever is already selected so an edit never loses its value.
// With nothing signed in, everything is offered — the person is about
// to sign in somewhere and the list is the map.
export function usableProviders(catalog, current) {
  const all = (catalog && catalog.providers) || [];
  const signed = all.filter((p) => p.signedIn);
  if (!signed.length) return all;
  if (current && !signed.some((p) => p.id === current)) {
    const cur = all.find((p) => p.id === current);
    return cur ? [...signed, cur] : signed;
  }
  return signed;
}
