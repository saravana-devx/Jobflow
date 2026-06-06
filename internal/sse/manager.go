// File: internal/sse/manager.go
package sse

import "sync"

type ClientManager struct {
	clients map[string]chan string
	mu      sync.RWMutex
}

func NewClientManager() *ClientManager {
	// clients field is nil by default so we have to add make statement to initialize the map before we can add any clients to it
	return &ClientManager{
		clients: make(map[string]chan string),
	}
}

func (m *ClientManager) AddClient(userID string) chan string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan string, 10)
	m.clients[userID] = ch
	return ch
}

func (m *ClientManager) RemoveClient(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ch, ok := m.clients[userID]; ok {
		close(ch)
		delete(m.clients, userID)
	}
}

func (m *ClientManager) SendToClient(userID string, message string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if ch, ok := m.clients[userID]; ok {
		select {
		case ch <- message:
		default:
		}
	}
}
