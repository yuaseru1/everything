const name = "everything-v13";
const files = ["/"];
self.addEventListener("install", (e) => {
  self.skipWaiting();
  e.waitUntil(caches.open(name).then((c) => c.addAll(files)));
});
self.addEventListener("activate", (e) => {
  e.waitUntil(self.clients.claim());
  e.waitUntil(
    caches
      .keys()
      .then((ks) =>
        Promise.all(ks.filter((k) => k !== name).map((k) => caches.delete(k))),
      ),
  );
});
self.addEventListener("fetch", (e) => {
  if (e.request.method !== "GET") return;
  const res = caches.open(name).then((c) => {
    return caches.match(e.request).then((response) => {
      const network = fetch(e.request).then(function (response) {
        if (e.request.method === "GET" && !e.request.url.includes("/file"))
          c.put(e.request, response.clone());
        return response;
      });
      if (response) return response;
      return network;
    });
  });
  if (
    e.request.mode === "navigate" ||
    e.request.headers.get("accept").includes("text/html")
  ) {
    e.respondWith(res.catch(() => caches.match("/")));
  } else {
    e.respondWith(res);
  }
});
