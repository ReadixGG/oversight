package protocol

// MessageType mirrors shared/protocol.gd MessageType enum.
// Keep these in sync!
const (
	// Connection
	MsgHandshake   = 1
	MsgHandshakeOK = 2
	MsgDisconnect  = 3
	MsgPing        = 4
	MsgPong        = 5

	// Lobby
	MsgFindMatch    = 10
	MsgMatchFound   = 11
	MsgCancelSearch = 12
	MsgSelectClass  = 13
	MsgLobbyState   = 14
	MsgMatchStart   = 15

	// Game Input (client -> server)
	MsgInputMove    = 20
	MsgInputShoot   = 21
	MsgInputAbility = 22
	MsgInputBuild   = 23
	MsgInputCollect = 24
	MsgInputDrop    = 25

	// Game State (server -> client)
	MsgGameSnapshot     = 30
	MsgPlayerSpawned    = 31
	MsgPlayerDied       = 32
	MsgPlayerRespawned  = 33
	MsgDamageDealt      = 34
	MsgResourceCollected = 35
	MsgResourceDelivered = 36
	MsgBuildingPlaced    = 37
	MsgBuildingDestroyed = 38
	MsgProjectileSpawned = 39

	// Map
	MsgMapData   = 40
	MsgFogUpdate = 41

	// Round
	MsgRoundStart  = 50
	MsgRoundEnd    = 51
	MsgMatchEnd    = 52
	MsgPreGameStart = 53
	MsgPreGameEnd   = 54

	// Coach
	MsgCoachDraw  = 60
	MsgCoachPing  = 61
	MsgCoachClear = 62

	// Chat
	MsgChatMessage = 70
	MsgQuickChat   = 71

	// Auth
	MsgAuthLogin    = 80
	MsgAuthRegister = 81
	MsgAuthResponse = 82
	MsgAuthToken    = 83
)

// ClassType
const (
	ClassCollector = 0
	ClassDefender  = 1
	ClassAttacker  = 2
)

// Team
const (
	TeamNone  = 0
	TeamAlpha = 1
	TeamBravo = 2
)

// Message is the wire format for all communication.
type Message struct {
	Type      int                    `json:"t"`
	Data      map[string]interface{} `json:"d"`
	Timestamp int64                  `json:"ts"`
}
