package network

import (
	"log"
	"sync"
)

// Hub manages all connected clients and routes messages.
type Hub struct {
	mu         sync.RWMutex
	clients    map[uint64]*Client
	nextID     uint64

	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan []byte

	// OnMessage is called when a client sends a message.
	OnMessage func(client *Client, data []byte)
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint64]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan []byte, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.nextID++
			client.ID = h.nextID
			h.clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("Client %d connected (total: %d)", client.ID, h.ClientCount())

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("Client %d disconnected (total: %d)", client.ID, h.ClientCount())

		case message := <-h.Broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// Client buffer full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) GetClient(id uint64) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.clients[id]
}

// SendToClients sends a message to specific client IDs.
func (h *Hub) SendToClients(ids []uint64, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, id := range ids {
		if c, ok := h.clients[id]; ok {
			select {
			case c.Send <- data:
			default:
			}
		}
	}
}
