package services

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // En producción, ajustar para seguridad
	},
}

// WSNotificationHub maneja las conexiones de WebSocket para notificaciones en tiempo real
type WSNotificationHub struct {
	// Clientes conectados: mapeamos el puntero de la conexión a un booleano
	clients    map[*websocket.Conn]bool
	broadcast  chan interface{}
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

var Hub *WSNotificationHub

func InitHub() {
	Hub = &WSNotificationHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan interface{}),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
	go Hub.run()
}

func (h *WSNotificationHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Println("[Hub] Nuevo administrador conectado vía WebSocket")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
			log.Println("[Hub] Administrador desconectado de WebSocket")

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				err := client.WriteJSON(message)
				if err != nil {
					log.Printf("[Hub] Error enviando mensaje JSON: %v", err)
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *WSNotificationHub) Broadcast(msg interface{}) {
	h.broadcast <- msg
}

func (h *WSNotificationHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Hub] Error al mejorar conexión: %v", err)
		return
	}
	h.register <- conn

	// Mantener la conexión abierta leyendo (aunque no esperamos mensajes del cliente)
	go func() {
		defer func() {
			h.unregister <- conn
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()
}
