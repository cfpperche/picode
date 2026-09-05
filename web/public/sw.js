/* ADR-0072: retain the registration/scope and push subscription while each
   application owns its hashed assets. HTML always revalidates on the network. */
const CACHE_PREFIX = "picode-ui-v2-";
const ACTIVE_CACHES = ["launcher", "desktop", "mobile"].map(app => CACHE_PREFIX + app);

self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", event => {
  event.waitUntil(caches.keys().then(keys => Promise.all(keys
    .filter(key => key === "picode-assets-v1" || (key.startsWith("picode-ui-") && !ACTIVE_CACHES.includes(key)))
    .map(key => caches.delete(key)))).then(() => self.clients.claim()));
});

self.addEventListener("fetch", event => {
  const url = new URL(event.request.url);
  if (url.origin !== location.origin || event.request.method !== "GET") return;
  if (url.pathname.startsWith("/api/") || url.pathname.startsWith("/ws/")) return;
  const app = url.pathname.match(/^\/(desktop|mobile)\/assets\//)?.[1]
    || (url.pathname.startsWith("/assets/") ? "launcher" : "");
  if (!app) {
    event.respondWith(fetch(event.request, { cache: "no-store" }));
    return;
  }
  event.respondWith(caches.open(CACHE_PREFIX + app).then(async cache => {
    const hit = await cache.match(event.request);
    if (hit) return hit;
    const response = await fetch(event.request);
    if (response.ok) await cache.put(event.request, response.clone());
    return response;
  }));
});

self.addEventListener("push", event => {
  let data = {};
  try { data = event.data ? event.data.json() : {}; } catch { data = { title: "PiCode", body: event.data ? event.data.text() : "" }; }
  event.waitUntil(self.registration.showNotification(data.title || "PiCode", {
    body: data.body || "", tag: data.tag || undefined, renotify: !!data.tag,
    icon: "/icon-192.png", badge: "/icon-192.png",
    data: { hash: typeof data.hash === "string" && data.hash.startsWith("#/") ? data.hash : "#/" },
  }));
});

self.addEventListener("notificationclick", event => {
  event.notification.close();
  const requested = event.notification.data?.hash;
  const hash = typeof requested === "string" && requested.startsWith("#/") ? requested : "#/";
  event.waitUntil(self.clients.matchAll({ type: "window", includeUncontrolled: true }).then(list => {
    const eligible = list.filter(client => {
      const url = new URL(client.url);
      return url.origin === location.origin && /^\/(?:(desktop|mobile)\/?)?$/.test(url.pathname) && "focus" in client;
    });
    const win = eligible.find(client => new URL(client.url).pathname.startsWith("/mobile")) || eligible[0];
    if (win) {
      win.postMessage({ type: "navigate", hash });
      return win.focus();
    }
    return self.clients.openWindow("/mobile/" + hash);
  }));
});
