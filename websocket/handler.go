package websocket

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// In production, you should check the origin
		return true
	},
}

// HandleWebSocket handles websocket requests from the peer
func HandleWebSocket(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get user ID from context first (set by auth middleware)
		userID, exists := c.Get("user_id")
		
		// If not in context, it means auth middleware didn't run
		// This happens because WebSocket upgrade happens before middleware can set headers
		// So we'll accept the connection and let the client authenticate
		if !exists {
			// For now, we'll use a default user ID of 0 to indicate unauthenticated
			// In a production system, you'd want to validate the token here
			log.Printf("[WS] Connection attempt without auth context")
			userID = uint(0)
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[WS] Error upgrading connection: %v", err)
			return
		}

		client := &Client{
			hub:    hub,
			conn:   conn,
			send:   make(chan []byte, 256),
			userID: userID.(uint),
		}

		client.hub.register <- client

		log.Printf("[WS] New client connected: User ID %d", client.userID)

		// Allow collection of memory referenced by the caller by doing all work in new goroutines
		go client.writePump()
		go client.readPump()
	}
}
