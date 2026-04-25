package handler

import (
	"log"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/Salman-kp/tripneo/bus-service/ws"
)

// WebsocketUpgradeMiddleware checks if the request is a websocket upgrade
func WebsocketUpgradeMiddleware(c fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// HandleWebSocket handles the actual websocket connection flow
var HandleWebSocket = websocket.New(func(c *websocket.Conn) {
	// e.g. /ws?userId=123
	userID := c.Query("userId")
	if userID == "" {
		log.Println("WS: No userId provided")
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
