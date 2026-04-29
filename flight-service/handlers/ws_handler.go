package handlers

import (
	"log"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/flight-service/ws"
)

// websocketupgrademiddleware checks if the request is a websocket upgrade
func WebsocketUpgradeMiddleware(c fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// handleWebSocket handles the actual websocket connection flow
var HandleWebSocket = websocket.New(func(c *websocket.Conn) {
	// Extract the secure user ID injected by the Gateway's JwtMiddleware
	userID := c.Headers("X-User-Id")
	if userID == "" {
		log.Println("WS: Unauthenticated access attempt blocked")
		return
	}

	ws.DefaultManager.AddConnection(userID, c)

	var (
		mt  int
		msg []byte
		err error
	)

	for {
		if mt, msg, err = c.ReadMessage(); err != nil {
			log.Println("WS: read error:", err)
			break
		}
		log.Printf("WS: recv from %s: %s (type: %d)", userID, msg, mt)

	}

	ws.DefaultManager.RemoveConnection(userID)
})
