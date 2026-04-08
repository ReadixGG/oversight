package game

import (
	"log"
	"sync"
	"time"

	"oversight-server/internal/network"
	"oversight-server/internal/protocol"
)

const (
	playersPerTeam  = 5
	playersPerMatch = playersPerTeam * 2
	minPlayersForMatch = 2
)

type Matchmaker struct {
	hub      *network.Hub
	tickRate int

	mu      sync.Mutex
	queue   []*network.Client
	matches map[string]*Match
}

func NewMatchmaker(hub *network.Hub, tickRate int) *Matchmaker {
	mm := &Matchmaker{
		hub:      hub,
		tickRate: tickRate,
		queue:    make([]*network.Client, 0),
		matches:  make(map[string]*Match),
	}

	hub.OnMessage = mm.handleMessage
	return mm
}

func (mm *Matchmaker) Run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mm.tryMakeMatch()
	}
}

func (mm *Matchmaker) handleMessage(client *network.Client, data []byte) {
	msgType := protocol.DecodeMessageType(data)
	payload := protocol.DecodePayload(data)

	switch msgType {
	case protocol.MsgFindMatch:
		mm.addToQueue(client)
	case protocol.MsgCancelSearch:
		mm.removeFromQueue(client)
	case protocol.MsgSelectClass:
		classType := protocol.DecodeSelectClass(payload)
		client.Class = classType
	default:
		if client.MatchID != "" {
			mm.mu.Lock()
			match, ok := mm.matches[client.MatchID]
			mm.mu.Unlock()
			if ok {
				match.HandleBinaryMessage(client, msgType, payload)
			}
		}
	}
}

func (mm *Matchmaker) addToQueue(client *network.Client) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for _, c := range mm.queue {
		if c.ID == client.ID {
			return
		}
	}
	mm.queue = append(mm.queue, client)
	log.Printf("Player %d joined queue (queue size: %d)", client.ID, len(mm.queue))
}

func (mm *Matchmaker) removeFromQueue(client *network.Client) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for i, c := range mm.queue {
		if c.ID == client.ID {
			mm.queue = append(mm.queue[:i], mm.queue[i+1:]...)
			log.Printf("Player %d left queue (queue size: %d)", client.ID, len(mm.queue))
			return
		}
	}
}

func (mm *Matchmaker) tryMakeMatch() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if len(mm.queue) < minPlayersForMatch {
		return
	}

	count := playersPerMatch
	if count > len(mm.queue) {
		count = len(mm.queue)
	}

	players := make([]*network.Client, count)
	copy(players, mm.queue[:count])
	mm.queue = mm.queue[count:]

	match := NewMatch(players, mm.tickRate)
	mm.matches[match.ID] = match

	log.Printf("Match %s created with %d players", match.ID, count)
	go match.Run()
}
