package handlers

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow any origin for simplicity (consider restricting in production)
	},
}

type Message struct {
	Channel string `json:"channel"`
	Content string `json:"content"`
}

type WebSocketHandler struct {
	clients   map[*websocket.Conn]bool
	mutex     sync.Mutex
	broadcast chan Message
}

func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan Message),
	}
}

func (h *WebSocketHandler) HandleWebSocket(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return err
	}
	defer conn.Close()

	h.mutex.Lock()
	h.clients[conn] = true
	h.mutex.Unlock()
	log.Println("New WebSocket client connected")

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			h.mutex.Lock()
			delete(h.clients, conn)
			h.mutex.Unlock()
			break
		}
	}
	return nil
}

func (h *WebSocketHandler) HandleMessages() {
	for {
		message := <-h.broadcast
		log.Println("Broadcasting message to channel:", message.Channel)
		h.mutex.Lock()
		for conn := range h.clients {
			// Here you can add logic to filter clients by channel if needed
			go func(c *websocket.Conn, m Message) {
				if err := c.WriteJSON(m); err != nil {
					log.Println("WebSocket write error:", err)
					h.mutex.Lock()
					delete(h.clients, c)
					h.mutex.Unlock()
				}
			}(conn, message)
		}
		h.mutex.Unlock()
	}
}

func (h *WebSocketHandler) StartBroadcasting() {
	go h.HandleMessages()
}

// BroadcastMessage allows other packages to send messages to the broadcast channel
func (h *WebSocketHandler) BroadcastMessage(channel, content string) {
	h.broadcast <- Message{Channel: channel, Content: content}
}
