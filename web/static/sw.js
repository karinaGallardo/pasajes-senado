const CACHE_NAME = 'pasajes-cache-v1';
const ASSETS = [
  '/dashboard',
  '/static/css/app.css',
  '/static/js/app.js',
  '/static/img/logo_senado.webp'
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      // cache.addAll(ASSETS); // Si queremos offline opcional
    })
  );
});

self.addEventListener('fetch', (event) => {
  // Estrategia: Network first, fallback a cache
  // event.respondWith(fetch(event.request).catch(() => caches.match(event.request)));
});

// Manejo de Notificaciones PUSH
self.addEventListener('push', (event) => {
  const data = event.data ? event.data.json() : {};
  const title = data.title || 'Sistema de Pasajes';
  const options = {
    body: data.message || 'Nueva notificación del sistema',
    icon: '/static/img/android-chrome-192x192.png',
    badge: '/static/img/favicon-32x32.png',
    data: {
      url: data.url || '/dashboard'
    }
  };

  event.waitUntil(
    self.registration.showNotification(title, options)
  );
});

// Al hacer clic en la notificación
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  event.waitUntil(
    clients.openWindow(event.notification.data.url)
  );
});
