package main

import (
	"log"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"

	"github.com/junaid9001/tripneo/chat-service/config"
	"github.com/junaid9001/tripneo/chat-service/db"
	"github.com/junaid9001/tripneo/chat-service/models"
	"github.com/junaid9001/tripneo/chat-service/ws"
)

func main() {
	cfg := config.LoadConfig()
	database := db.Connect(cfg.DatabaseURL)
	manager := ws.NewChatManager(database)

	app := fiber.New()

	api := app.Group("/api/chat")

	// Middleware to extract X-User-ID appended by the API Gateway
	authMiddleware := func(c fiber.Ctx) error {
		userID := c.Get("X-User-Id")
		if userID == "" {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized: missing gateway identity payload"})
		}
		
		c.Locals("user_id", userID)
		return c.Next()
	}

	// 1. Initial Load: Fetch message history explicitly so we don't have to send thousands of messages over WS on initial connection
	api.Get("/messages", authMiddleware, func(c fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		
		var messages []models.ChatMessage
		// Order by creation, older first
		database.Where("user_id = ? AND is_deleted = false", userID).Order("created_at asc").Find(&messages)
		
		return c.JSON(fiber.Map{
			"messages": messages,
		})
	})

	// 2. WebSocket Route
	api.Use("/ws", func(c fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			return authMiddleware(c)
		}
		return fiber.ErrUpgradeRequired
	})

	api.Get("/ws", websocket.New(manager.HandleConnection))


	// 3. Admin Routes (Simulated for later proper implementation)
	admin := api.Group("/admin")
	
	// E.g. GET /api/chat/admin/users
	// E.g. GET /api/chat/admin/messages/:userId

	// The magic bridge: Post an admin reply and push it to active WS
	admin.Post("/reply/:userId", func(c fiber.Ctx) error {
		// In production, guard this with IsAdmin middleware
		userID := c.Params("userId")
		
		type Request struct {
			Content string `json:"content"`
		}
		var req Request
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "bad request"})
		}

		msg := models.ChatMessage{
			UserID:  userID,
			Sender:  "ADMIN",
			Content: req.Content,
		}

		if err := database.Create(&msg).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to save message"})
		}

		// Push directly to the frontend if they have the socket open!
		manager.PushMessageToUser(userID, msg)

		return c.JSON(fiber.Map{"success": true, "message": msg})
	})

	log.Printf("Chat service running on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
