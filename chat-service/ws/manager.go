package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/junaid9001/tripneo/chat-service/models"
	"gorm.io/gorm"
)

type ChatManager struct {
	db      *gorm.DB
	clients map[string]*websocket.Conn // UserID -> Connection
	mu      sync.RWMutex
}

func NewChatManager(db *gorm.DB) *ChatManager {
	return &ChatManager{
		db:      db,
		clients: make(map[string]*websocket.Conn),
	}
}

func (m *ChatManager) AddClient(userID string, conn *websocket.Conn) {
	m.mu.Lock()
	m.clients[userID] = conn
	m.mu.Unlock()
}

func (m *ChatManager) RemoveClient(userID string) {
	m.mu.Lock()
	if conn, ok := m.clients[userID]; ok {
		conn.Close()
		delete(m.clients, userID)
	}
	m.mu.Unlock()
}

type ClientAction struct {
	Action    string `json:"action"` // send or delete
	Content   string `json:"content,omitempty"`
	MessageID string `json:"message_id,omitempty"`
}

func (m *ChatManager) HandleConnection(c *websocket.Conn) {
	userID := c.Locals("user_id").(string)

	m.AddClient(userID, c)
	defer m.RemoveClient(userID)

	log.Printf("[WS] User %s connected to chat", userID)

	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			log.Printf("[WS] Error reading message from %s: %v", userID, err)
			break
		}

		if messageType != websocket.TextMessage {
			continue
		}

		var action ClientAction
		if err := json.Unmarshal(message, &action); err != nil {
			log.Printf("[WS] Invalid JSON from %s: %v", userID, err)
			continue
		}

		switch action.Action {
		case "send":
			m.handleSend(userID, action.Content, c)
		case "delete":
			m.handleDelete(userID, action.MessageID, c)
		}
	}
}

func (m *ChatManager) handleSend(userID, content string, c *websocket.Conn) {
	if content == "" {
		return
	}

	msg := models.ChatMessage{
		UserID:  userID,
		Sender:  "USER",
		Content: content,
	}

	if err := m.db.Create(&msg).Error; err != nil {
		log.Printf("[WS] DB Error saving message for %s: %v", userID, err)
		return
	}

	response := map[string]interface{}{
		"action":  "new_message",
		"message": msg,
	}

	_ = c.WriteJSON(response)

}

func (m *ChatManager) handleDelete(userID, messageID string, c *websocket.Conn) {
	if messageID == "" {
		return
	}

	result := m.db.Model(&models.ChatMessage{}).
		Where("id = ? AND user_id = ?", messageID, userID).
		Update("is_deleted", true)

	if result.Error != nil || result.RowsAffected == 0 {
		log.Printf("[WS] Error deleting message %s for %s", messageID, userID)
		return
	}

	response := map[string]interface{}{
		"action":     "message_deleted",
		"message_id": messageID,
	}

	_ = c.WriteJSON(response)
}

func (m *ChatManager) PushMessageToUser(userID string, msg models.ChatMessage) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if conn, exists := m.clients[userID]; exists {
		response := map[string]interface{}{
			"action":  "new_message",
			"message": msg,
		}
		go func() {
			_ = conn.WriteJSON(response)
		}()
	}
}
