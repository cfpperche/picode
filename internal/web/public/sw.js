/* Cache hashed Vite assets only. index.html is always network (new release = new JS). */
const CACHE = "picode-assets-v1";

self.addEventListener("install", () => self.skipWaiting());

self.addEventListener("activate", (e) => {
  e.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))
    ).then(() => self.clients.claim())
  );
});

self.addEventListener("fetch", (e) => {
  const u = new URL(e.request.url);
  if (u.origin !== location.origin) return;
  if (u.pathname.startsWith("/api/") || u.pathname.startsWith("/ws/")) return;
  if (e.request.method !== "GET") return;
  if (!u.pathname.startsWith("/assets/")) {
    e.respondWith(fetch(e.request, { cache: "no-store" }));
    return;
  }
  e.respondWith(
    caches.open(CACHE).then((c) =>
      c.match(e.request).then((hit) => {
        if (hit) return hit;
        return fetch(e.request).then((res) => {
          if (res.ok) c.put(e.request, res.clone());
          return res;
        });
      })
    )
  );
});
