package game

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"oversight-server/internal/network"
	"oversight-server/internal/protocol"
)

type PlayerState struct {
	ID       uint64
	Team     int
	Class    int
	X, Y     float64
	VX, VY   float64
	HP       float64
	MaxHP    float64
	Speed    float64
	Dead     bool
	Carrying bool
	LastSeq  int
}

type Match struct {
	ID       string
	Phase    int
	Round    int
	Score    [3]int // index by team
	TickRate int

	mu       sync.Mutex
	players  []*network.Client
	states   map[uint64]*PlayerState

	roundTimer float64
	preGameTimer float64

	stopCh chan struct{}
}

func NewMatch(players []*network.Client, tickRate int) *Match {
	matchID := fmt.Sprintf("match_%d", time.Now().UnixNano())

	m := &Match{
		ID:       matchID,
		Phase:    protocol.MsgPreGameStart,
		Round:    1,
		TickRate: tickRate,
		players:  players,
		states:   make(map[uint64]*PlayerState),
		stopCh:   make(chan struct{}),
	}

	// Assign teams (first half Alpha, second half Bravo)
	half := len(players) / 2
	for i, p := range players {
		p.MatchID = matchID
		team := protocol.TeamAlpha
		if i >= half {
			team = protocol.TeamBravo
		}
		p.Team = team

		state := &PlayerState{
			ID:    p.ID,
			Team:  team,
			Class: p.Class,
			MaxHP: classMaxHP(p.Class),
			Speed: classSpeed(p.Class),
		}
		state.HP = state.MaxHP
		m.states[p.ID] = state
	}

	m.spawnPlayers()
	return m
}

func (m *Match) Run() {
	log.Printf("Match %s started (round %d)", m.ID, m.Round)

	// Notify all players
	m.broadcastToAll(protocol.Message{
		Type: protocol.MsgMatchFound,
		Data: map[string]interface{}{
			"match_id": m.ID,
		},
		Timestamp: time.Now().UnixMilli(),
	})

	// Send spawn data to all players
	m.mu.Lock()
	for _, p := range m.players {
		st := m.states[p.ID]
		m.broadcastToAllUnlocked(protocol.Message{
			Type: protocol.MsgPlayerSpawned,
			Data: map[string]interface{}{
				"id":    st.ID,
				"team":  st.Team,
				"class": st.Class,
				"x":     st.X,
				"y":     st.Y,
			},
			Timestamp: time.Now().UnixMilli(),
		})
	}
	m.mu.Unlock()

	// Pre-game phase (60 seconds for normal, shortened for MVP testing)
	m.preGameTimer = 10.0 // 10 seconds for testing
	m.broadcastToAll(protocol.Message{
		Type:      protocol.MsgPreGameStart,
		Data:      map[string]interface{}{"duration": m.preGameTimer},
		Timestamp: time.Now().UnixMilli(),
	})

	// Game loop
	tickDuration := time.Duration(1000/m.TickRate) * time.Millisecond
	ticker := time.NewTicker(tickDuration)
	defer ticker.Stop()

	lastTime := time.Now()

	for {
		select {
		case <-m.stopCh:
			return
		case now := <-ticker.C:
			dt := now.Sub(lastTime).Seconds()
			lastTime = now
			m.update(dt)
		}
	}
}

func (m *Match) update(dt float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch {
	case m.preGameTimer > 0:
		m.preGameTimer -= dt
		if m.preGameTimer <= 0 {
			m.preGameTimer = 0
			m.roundTimer = 720.0 // 12 minutes
			m.broadcastToAllUnlocked(protocol.Message{
				Type:      protocol.MsgRoundStart,
				Data:      map[string]interface{}{"round": m.Round, "duration": m.roundTimer},
				Timestamp: time.Now().UnixMilli(),
			})
		}
		return

	case m.roundTimer > 0:
		m.roundTimer -= dt
		m.sendSnapshot()

		if m.roundTimer <= 0 {
			m.endRound()
		}
	}
}

func (m *Match) sendSnapshot() {
	playerData := make([]map[string]interface{}, 0, len(m.states))
	for _, st := range m.states {
		playerData = append(playerData, map[string]interface{}{
			"id":       st.ID,
			"x":        st.X,
			"y":        st.Y,
			"vx":       st.VX,
			"vy":       st.VY,
			"hp":       st.HP,
			"carrying": st.Carrying,
			"seq":      st.LastSeq,
		})
	}

	m.broadcastToAllUnlocked(protocol.Message{
		Type: protocol.MsgGameSnapshot,
		Data: map[string]interface{}{
			"players": playerData,
			"timer":   m.roundTimer,
		},
		Timestamp: time.Now().UnixMilli(),
	})
}

func (m *Match) HandleMessage(client *network.Client, msg protocol.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	st, ok := m.states[client.ID]
	if !ok || st.Dead {
		return
	}

	switch msg.Type {
	case protocol.MsgInputMove:
		dx, _ := msg.Data["dx"].(float64)
		dy, _ := msg.Data["dy"].(float64)
		dtVal, _ := msg.Data["dt"].(float64)
		seq, _ := msg.Data["seq"].(float64)

		speed := st.Speed
		if st.Carrying {
			speed *= 0.7
		}

		length := dx*dx + dy*dy
		if length > 0 {
			invLen := 1.0 / sqrt(length)
			dx *= invLen
			dy *= invLen
		}

		st.VX = dx * speed
		st.VY = dy * speed
		st.X += st.VX * dtVal
		st.Y += st.VY * dtVal
		st.LastSeq = int(seq)

	case protocol.MsgInputShoot:
		// TODO: projectile spawning
	case protocol.MsgInputBuild:
		// TODO: building placement
	case protocol.MsgInputCollect:
		// TODO: resource collection
	}
}

func (m *Match) endRound() {
	// Determine winner by remaining HP of nexus or total team HP
	winner := protocol.TeamAlpha // placeholder logic
	m.Score[winner]++

	m.broadcastToAllUnlocked(protocol.Message{
		Type: protocol.MsgRoundEnd,
		Data: map[string]interface{}{
			"winner": winner,
			"round":  m.Round,
			"score_alpha": m.Score[protocol.TeamAlpha],
			"score_bravo": m.Score[protocol.TeamBravo],
		},
		Timestamp: time.Now().UnixMilli(),
	})

	if m.Score[winner] >= 2 {
		m.endMatch(winner)
		return
	}

	// Next round
	m.Round++
	m.swapTeams()
	m.spawnPlayers()
	m.preGameTimer = 10.0
}

func (m *Match) endMatch(winner int) {
	m.broadcastToAllUnlocked(protocol.Message{
		Type: protocol.MsgMatchEnd,
		Data: map[string]interface{}{
			"winner":      winner,
			"score_alpha": m.Score[protocol.TeamAlpha],
			"score_bravo": m.Score[protocol.TeamBravo],
		},
		Timestamp: time.Now().UnixMilli(),
	})

	close(m.stopCh)
}

func (m *Match) swapTeams() {
	for _, p := range m.players {
		if p.Team == protocol.TeamAlpha {
			p.Team = protocol.TeamBravo
		} else {
			p.Team = protocol.TeamAlpha
		}
		m.states[p.ID].Team = p.Team
	}
}

func (m *Match) spawnPlayers() {
	mapWidth := 2000.0
	for _, st := range m.states {
		if st.Team == protocol.TeamAlpha {
			st.X = 100 + rand.Float64()*100
			st.Y = 400 + rand.Float64()*200
		} else {
			st.X = mapWidth - 200 + rand.Float64()*100
			st.Y = 400 + rand.Float64()*200
		}
		st.HP = st.MaxHP
		st.Dead = false
		st.Carrying = false
		st.VX = 0
		st.VY = 0
	}
}

func (m *Match) broadcastToAll(msg protocol.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcastToAllUnlocked(msg)
}

func (m *Match) broadcastToAllUnlocked(msg protocol.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	ids := make([]uint64, 0, len(m.players))
	for _, p := range m.players {
		ids = append(ids, p.ID)
	}
	m.players[0].Hub.SendToClients(ids, data)
}

func classMaxHP(class int) float64 {
	switch class {
	case protocol.ClassCollector:
		return 80
	case protocol.ClassDefender:
		return 120
	case protocol.ClassAttacker:
		return 100
	default:
		return 100
	}
}

func classSpeed(class int) float64 {
	switch class {
	case protocol.ClassCollector:
		return 250
	case protocol.ClassDefender:
		return 180
	case protocol.ClassAttacker:
		return 220
	default:
		return 200
	}
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Newton's method
	z := x / 2
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
