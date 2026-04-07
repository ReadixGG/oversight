package network

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"oversight-server/internal/protocol"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	sendBufferSize = 256
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev; restrict in production
	},
}

// Client represents a single WebSocket connection.
type Client struct {
	ID   uint64
	Hub  *Hub
	Conn *websocket.Conn
	Send chan []byte

	// Game state
	MatchID   string
	PlayerID  int
	Team      int
	Class     int
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		Hub:  hub,
		Conn: conn,
		Send: make(chan []byte, sendBufferSize),
	}

	hub.Register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("Client %d read error: %v", c.ID, err)
			}
			break
		}

		// Handle ping/pong at network level
		var msg protocol.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Type == protocol.MsgPing {
			c.sendPong()
			continue
		}

		if msg.Type == protocol.MsgHandshake {
			c.sendHandshakeOK()
			continue
		}

		if c.Hub.OnMessage != nil {
			c.Hub.OnMessage(c, message)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) SendJSON(msg protocol.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.Send <- data:
	default:
	}
}

func (c *Client) sendPong() {
	c.SendJSON(protocol.Message{
		Type:      protocol.MsgPong,
		Data:      map[string]interface{}{},
		Timestamp: time.Now().UnixMilli(),
	})
}

func (c *Client) sendHandshakeOK() {
	c.SendJSON(protocol.Message{
		Type: protocol.MsgHandshakeOK,
		Data: map[string]interface{}{
			"player_id": c.ID,
		},
		Timestamp: time.Now().UnixMilli(),
	})
}
