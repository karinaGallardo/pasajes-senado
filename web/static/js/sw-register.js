if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker
      .register("/sw.js")
      .then((registration) => {
        console.log("SW registrado exitosamente:", registration.scope);

        // Pedir permiso y suscribir
        if ("Notification" in window) {
          Notification.requestPermission().then((permission) => {
            if (permission === "granted") {
              const vapidKey = document.querySelector('meta[name="vapid-public-key"]')?.content;
              if (vapidKey) {
                subscribeUser(registration, vapidKey);
              }
            }
          });
        }
      })
      .catch((error) => {
        console.log("Registro del SW falló:", error);
      });
  });
}

function subscribeUser(registration, publicKey) {
  const applicationServerKey = urlB64ToUint8Array(publicKey);
  registration.pushManager
    .subscribe({
      userVisibleOnly: true,
      applicationServerKey: applicationServerKey,
    })
    .then((subscription) => {
      console.log("Usuario suscrito a Push:", subscription);
      // Enviar suscripción al servidor
      saveSubscriptionOnServer(subscription);
    })
    .catch((err) => {
      console.log("Fallo al suscribir al usuario: ", err);
    });
}

function saveSubscriptionOnServer(subscription) {
  fetch("/api/notifications/subscribe", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(subscription),
  })
    .then((response) => {
      if (!response.ok) throw new Error("Error al guardar suscripción");
      console.log("Suscripción guardada en el servidor");
    })
    .catch((err) => console.log("Error enviando suscripción:", err));
}

function urlB64ToUint8Array(base64String) {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/\-/g, "+").replace(/_/g, "/");
  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);
  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}
