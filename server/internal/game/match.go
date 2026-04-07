package game

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
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

	RespawnTimer float64
	ShootCD      float64
}

type ProjectileState struct {
	ID       uint64
	OwnerID  uint64
	Team     int
	X, Y     float64
	DX, DY   float64
	Speed    float64
	Damage   float64
	Lifetime float64
	Age      float64
}

// Tile types for the map grid
const (
	TileGround    = 0
	TileWall      = 1
	TileWater     = 2
	TileResource  = 3
	TileBaseAlpha = 4
	TileBaseBravo = 5
)

type GameMap struct {
	Width    int
	Height   int
	TileSize int
	Tiles    [][]int
	Seed     int64
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

	projectiles   map[uint64]*ProjectileState
	nextProjID    uint64

	gameMap *GameMap

	roundTimer   float64
	preGameTimer float64

	stopCh chan struct{}
}

func NewMatch(players []*network.Client, tickRate int) *Match {
	matchID := fmt.Sprintf("match_%d", time.Now().UnixNano())

	seed := time.Now().UnixNano()
	gMap := generateMap(seed)

	m := &Match{
		ID:          matchID,
		Phase:       protocol.MsgPreGameStart,
		Round:       1,
		TickRate:    tickRate,
		players:     players,
		states:      make(map[uint64]*PlayerState),
		projectiles: make(map[uint64]*ProjectileState),
		gameMap:     gMap,
		stopCh:      make(chan struct{}),
	}

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

	// Send map data to all players
	m.broadcastToAll(protocol.Message{
		Type: protocol.MsgMapData,
		Data: m.gameMap.toNetworkData(),
		Timestamp: time.Now().UnixMilli(),
	})

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
		m.updateProjectiles(dt)
		m.updateRespawns(dt)
		m.sendSnapshot()

		if m.roundTimer <= 0 {
			m.endRound()
		}
	}
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
			invLen := 1.0 / math.Sqrt(length)
			dx *= invLen
			dy *= invLen
		}

		st.VX = dx * speed
		st.VY = dy * speed
		st.X += st.VX * dtVal
		st.Y += st.VY * dtVal
		st.LastSeq = int(seq)

	case protocol.MsgInputShoot:
		if st.ShootCD > 0 {
			return
		}
		dx, _ := msg.Data["dx"].(float64)
		dy, _ := msg.Data["dy"].(float64)

		length := dx*dx + dy*dy
		if length > 0 {
			invLen := 1.0 / math.Sqrt(length)
			dx *= invLen
			dy *= invLen
		} else {
			dx = 1
			dy = 0
		}

		st.ShootCD = classShootRate(st.Class)
		m.spawnProjectile(st, dx, dy)

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
	mapPixelW := float64(m.gameMap.Width * m.gameMap.TileSize)
	mapPixelH := float64(m.gameMap.Height * m.gameMap.TileSize)
	centerY := mapPixelH / 2.0

	for _, st := range m.states {
		if st.Team == protocol.TeamAlpha {
			st.X = 80 + rand.Float64()*60
			st.Y = centerY - 60 + rand.Float64()*120
		} else {
			st.X = mapPixelW - 140 + rand.Float64()*60
			st.Y = centerY - 60 + rand.Float64()*120
		}
		st.HP = st.MaxHP
		st.Dead = false
		st.Carrying = false
		st.VX = 0
		st.VY = 0
		st.ShootCD = 0
		st.RespawnTimer = 0
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

// --- Projectile system ---

func (m *Match) spawnProjectile(shooter *PlayerState, dx, dy float64) {
	m.nextProjID++
	proj := &ProjectileState{
		ID:       m.nextProjID,
		OwnerID:  shooter.ID,
		Team:     shooter.Team,
		X:        shooter.X + dx*20,
		Y:        shooter.Y + dy*20,
		DX:       dx,
		DY:       dy,
		Speed:    classProjectileSpeed(shooter.Class),
		Damage:   classShootDamage(shooter.Class),
		Lifetime: classShootRange(shooter.Class) / classProjectileSpeed(shooter.Class),
	}
	m.projectiles[proj.ID] = proj

	m.broadcastToAllUnlocked(protocol.Message{
		Type: protocol.MsgProjectileSpawned,
		Data: map[string]interface{}{
			"id":     proj.ID,
			"owner":  proj.OwnerID,
			"team":   proj.Team,
			"x":      proj.X,
			"y":      proj.Y,
			"dx":     proj.DX,
			"dy":     proj.DY,
			"speed":  proj.Speed,
			"damage": proj.Damage,
		},
		Timestamp: time.Now().UnixMilli(),
	})
}

func (m *Match) updateProjectiles(dt float64) {
	toRemove := make([]uint64, 0)

	for id, proj := range m.projectiles {
		proj.X += proj.DX * proj.Speed * dt
		proj.Y += proj.DY * proj.Speed * dt
		proj.Age += dt

		if proj.Age >= proj.Lifetime {
			toRemove = append(toRemove, id)
			continue
		}

		// Check wall collision
		if m.gameMap != nil {
			tileX := int(proj.X) / m.gameMap.TileSize
			tileY := int(proj.Y) / m.gameMap.TileSize
			if tileX >= 0 && tileX < m.gameMap.Width && tileY >= 0 && tileY < m.gameMap.Height {
				if m.gameMap.Tiles[tileY][tileX] == TileWall {
					toRemove = append(toRemove, id)
					continue
				}
			}
		}

		// Check hit on players
		for _, st := range m.states {
			if st.Dead || st.Team == proj.Team {
				continue
			}
			distSq := (st.X-proj.X)*(st.X-proj.X) + (st.Y-proj.Y)*(st.Y-proj.Y)
			hitRadius := 16.0
			if distSq <= hitRadius*hitRadius {
				st.HP -= proj.Damage
				toRemove = append(toRemove, id)

				m.broadcastToAllUnlocked(protocol.Message{
					Type: protocol.MsgDamageDealt,
					Data: map[string]interface{}{
						"target":  st.ID,
						"attacker": proj.OwnerID,
						"amount":  proj.Damage,
					},
					Timestamp: time.Now().UnixMilli(),
				})

				if st.HP <= 0 {
					st.HP = 0
					st.Dead = true
					st.RespawnTimer = 5.0

					m.broadcastToAllUnlocked(protocol.Message{
						Type: protocol.MsgPlayerDied,
						Data: map[string]interface{}{
							"id":     st.ID,
							"killer": proj.OwnerID,
						},
						Timestamp: time.Now().UnixMilli(),
					})
				}
				break
			}
		}
	}

	for _, id := range toRemove {
		delete(m.projectiles, id)
	}
}

func (m *Match) updateRespawns(dt float64) {
	for _, st := range m.states {
		if !st.Dead {
			st.ShootCD -= dt
			if st.ShootCD < 0 {
				st.ShootCD = 0
			}
			continue
		}
		st.RespawnTimer -= dt
		if st.RespawnTimer <= 0 {
			st.Dead = false
			st.HP = st.MaxHP
			st.Carrying = false

			// Respawn at base
			if st.Team == protocol.TeamAlpha {
				st.X = 100 + rand.Float64()*100
				st.Y = 400 + rand.Float64()*200
			} else {
				st.X = float64(m.gameMap.Width*m.gameMap.TileSize) - 200 + rand.Float64()*100
				st.Y = 400 + rand.Float64()*200
			}

			m.broadcastToAllUnlocked(protocol.Message{
				Type: protocol.MsgPlayerRespawned,
				Data: map[string]interface{}{
					"id": st.ID,
					"x":  st.X,
					"y":  st.Y,
				},
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}
}

// --- Fog of War: filter snapshot per team ---

func (m *Match) sendSnapshot() {
	// Build per-team snapshots (fog of war filtering)
	for _, p := range m.players {
		visiblePlayers := make([]map[string]interface{}, 0, len(m.states))
		for _, st := range m.states {
			if st.Team == p.Team {
				// Always show teammates
				visiblePlayers = append(visiblePlayers, playerToMap(st))
			} else if !st.Dead {
				// Only show enemy if within vision range of any teammate
				if m.isVisibleByTeam(st.X, st.Y, p.Team) {
					visiblePlayers = append(visiblePlayers, playerToMap(st))
				}
			}
		}

		msg := protocol.Message{
			Type: protocol.MsgGameSnapshot,
			Data: map[string]interface{}{
				"players": visiblePlayers,
				"timer":   m.roundTimer,
			},
			Timestamp: time.Now().UnixMilli(),
		}
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		p.Hub.SendToClients([]uint64{p.ID}, data)
	}
}

const visionRange = 400.0

func (m *Match) isVisibleByTeam(x, y float64, team int) bool {
	for _, st := range m.states {
		if st.Team != team || st.Dead {
			continue
		}
		distSq := (st.X-x)*(st.X-x) + (st.Y-y)*(st.Y-y)
		if distSq <= visionRange*visionRange {
			return true
		}
	}
	return false
}

func playerToMap(st *PlayerState) map[string]interface{} {
	return map[string]interface{}{
		"id":       st.ID,
		"x":        st.X,
		"y":        st.Y,
		"vx":       st.VX,
		"vy":       st.VY,
		"hp":       st.HP,
		"carrying": st.Carrying,
		"seq":      st.LastSeq,
	}
}

// --- Map generation ---

func generateMap(seed int64) *GameMap {
	rng := rand.New(rand.NewSource(seed))

	width := 64  // tiles
	height := 32 // tiles
	tileSize := 32

	tiles := make([][]int, height)
	for y := 0; y < height; y++ {
		tiles[y] = make([]int, width)
	}

	// Fill borders with walls
	for x := 0; x < width; x++ {
		tiles[0][x] = TileWall
		tiles[height-1][x] = TileWall
	}
	for y := 0; y < height; y++ {
		tiles[y][0] = TileWall
		tiles[y][width-1] = TileWall
	}

	// Place bases
	for y := height/2 - 3; y <= height/2+3; y++ {
		for x := 1; x <= 4; x++ {
			tiles[y][x] = TileBaseAlpha
			tiles[y][width-1-x] = TileBaseBravo
		}
	}

	// Scatter walls symmetrically (mirror left-right for fairness)
	numWallClusters := 8 + rng.Intn(6)
	for i := 0; i < numWallClusters; i++ {
		cx := 6 + rng.Intn(width/2-8)
		cy := 2 + rng.Intn(height-4)
		clusterSize := 2 + rng.Intn(3)

		for dy := 0; dy < clusterSize; dy++ {
			for dx := 0; dx < clusterSize; dx++ {
				wx := cx + dx
				wy := cy + dy
				if wy > 0 && wy < height-1 && wx > 0 && wx < width-1 {
					if tiles[wy][wx] == TileGround {
						tiles[wy][wx] = TileWall
					}
					// Mirror
					mx := width - 1 - wx
					if tiles[wy][mx] == TileGround {
						tiles[wy][mx] = TileWall
					}
				}
			}
		}
	}

	// Scatter resources symmetrically
	numResources := 10 + rng.Intn(8)
	for i := 0; i < numResources; i++ {
		rx := 6 + rng.Intn(width/2-8)
		ry := 2 + rng.Intn(height-4)
		if tiles[ry][rx] == TileGround {
			tiles[ry][rx] = TileResource
			mx := width - 1 - rx
			tiles[ry][mx] = TileResource
		}
	}

	return &GameMap{
		Width:    width,
		Height:   height,
		TileSize: tileSize,
		Tiles:    tiles,
		Seed:     seed,
	}
}

func (gm *GameMap) toNetworkData() map[string]interface{} {
	flat := make([]int, gm.Width*gm.Height)
	for y := 0; y < gm.Height; y++ {
		for x := 0; x < gm.Width; x++ {
			flat[y*gm.Width+x] = gm.Tiles[y][x]
		}
	}
	return map[string]interface{}{
		"width":     gm.Width,
		"height":    gm.Height,
		"tile_size": gm.TileSize,
		"tiles":     flat,
		"seed":      gm.Seed,
	}
}

// --- Class stats ---

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

func classShootRate(class int) float64 {
	switch class {
	case protocol.ClassCollector:
		return 0.5
	case protocol.ClassDefender:
		return 0.4
	case protocol.ClassAttacker:
		return 0.25
	default:
		return 0.4
	}
}

func classShootDamage(class int) float64 {
	switch class {
	case protocol.ClassCollector:
		return 8
	case protocol.ClassDefender:
		return 12
	case protocol.ClassAttacker:
		return 15
	default:
		return 10
	}
}

func classShootRange(class int) float64 {
	switch class {
	case protocol.ClassCollector:
		return 300
	case protocol.ClassDefender:
		return 350
	case protocol.ClassAttacker:
		return 450
	default:
		return 350
	}
}

func classProjectileSpeed(class int) float64 {
	switch class {
	case protocol.ClassCollector:
		return 500
	case protocol.ClassDefender:
		return 550
	case protocol.ClassAttacker:
		return 650
	default:
		return 600
	}
}

// Ensure math.Sqrt is used (imported above)
var _ = math.Sqrt
