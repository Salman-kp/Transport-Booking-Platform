package ws

import (
	"log"
	"sync"

	"github.com/gofiber/contrib/v3/websocket"
)

// manager keeps track of active WebSocket connections.
type Manager struct {
	connections map[string]*websocket.Conn // userID -> conn
	mu          sync.RWMutex
}

var DefaultManager = &Manager{
	connections: make(map[string]*websocket.Conn),
}

func (m *Manager) AddConnection(userID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[userID] = conn
	log.Printf("WS: User %s connected", userID)
}

func (m *Manager) RemoveConnection(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conn, ok := m.connections[userID]; ok {

		_ = conn.Close()
		delete(m.connections, userID)
		log.Printf("WS: User %s disconnected", userID)
	}
}

func (m *Manager) SendToUser(userID string, message interface{}) error {
	m.mu.RLock()
	conn, ok := m.connections[userID]
	m.mu.RUnlock()

	if !ok {
		return nil // user is not currently connected
	}

	return conn.WriteJSON(message)
}
