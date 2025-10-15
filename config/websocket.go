package config

import (
	"github.com/IkingariSolorzano/omma-be/websocket"
)

// Global WebSocket hub instance
var WSHub *websocket.Hub

// InitializeWebSocketHub initializes the global WebSocket hub
func InitializeWebSocketHub() {
	WSHub = websocket.NewHub()
	go WSHub.Run()
}
