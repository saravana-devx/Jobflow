package sse

import (
	"sync"

	"github.com/google/uuid"
)

type ClientManager struct {
	clients map[string]map[string]chan string
	mu      sync.RWMutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]map[string]chan string),
	}
}

func (m *ClientManager) AddClient(userID string) (string, chan string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan string, 10)
	connID := uuid.NewString()
	if m.clients[userID] == nil {
		m.clients[userID] = make(map[string]chan string)
	}
	m.clients[userID][connID] = ch
	return connID, ch
}

func (m *ClientManager) RemoveClient(userID, connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	conns, ok := m.clients[userID]
	if !ok {
		return
	}
	if ch, ok := conns[connID]; ok {
		close(ch)
		delete(conns, connID)
	}
	if len(conns) == 0 {
		delete(m.clients, userID)
	}
}

func (m *ClientManager) SendToClient(userID string, message string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ch := range m.clients[userID] {
		select {
		case ch <- message:
		default:
		}
	}
}
