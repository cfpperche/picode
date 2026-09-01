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

/* Web Push (ADR-0047). The server sends {title, body, hash, tag}; a tap
   lands on `hash` in an open PiCode window or opens the mobile shell
   there. `tag` collapses repeats of the same subject (one agent asking
   twice is one notification). */
self.addEventListener("push", (e) => {
  let data = {};
  try { data = e.data ? e.data.json() : {}; } catch { data = { title: "PiCode", body: e.data ? e.data.text() : "" }; }
  const title = data.title || "PiCode";
  e.waitUntil(self.registration.showNotification(title, {
    body: data.body || "",
    tag: data.tag || undefined,
    renotify: !!data.tag,
    icon: "/icon-192.png",
    badge: "/icon-192.png",
    data: { hash: data.hash || "#/" },
  }));
});

self.addEventListener("notificationclick", (e) => {
  e.notification.close();
  const hash = (e.notification.data && e.notification.data.hash) || "#/";
  e.waitUntil(self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((list) => {
    const win = list.find((c) => "focus" in c);
    if (win) {
      win.postMessage({ type: "navigate", hash });
      return win.focus();
    }
    return self.clients.openWindow("/?mobile=1" + hash);
  }));
});
