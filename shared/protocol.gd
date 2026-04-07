class_name Protocol

## Shared network protocol constants and message types.
## This file defines the contract between client and server.
## Keep in sync with server/internal/protocol/messages.go

enum MessageType {
	# Connection
	HANDSHAKE = 1,
	HANDSHAKE_OK = 2,
	DISCONNECT = 3,
	PING = 4,
	PONG = 5,

	# Lobby
	FIND_MATCH = 10,
	MATCH_FOUND = 11,
	CANCEL_SEARCH = 12,
	SELECT_CLASS = 13,
	LOBBY_STATE = 14,
	MATCH_START = 15,

	# Game Input (client -> server)
	INPUT_MOVE = 20,
	INPUT_SHOOT = 21,
	INPUT_ABILITY = 22,
	INPUT_BUILD = 23,
	INPUT_COLLECT = 24,
	INPUT_DROP = 25,

	# Game State (server -> client)
	GAME_SNAPSHOT = 30,
	PLAYER_SPAWNED = 31,
	PLAYER_DIED = 32,
	PLAYER_RESPAWNED = 33,
	DAMAGE_DEALT = 34,
	RESOURCE_COLLECTED = 35,
	RESOURCE_DELIVERED = 36,
	BUILDING_PLACED = 37,
	BUILDING_DESTROYED = 38,
	PROJECTILE_SPAWNED = 39,

	# Map
	MAP_DATA = 40,
	FOG_UPDATE = 41,

	# Round
	ROUND_START = 50,
	ROUND_END = 51,
	MATCH_END = 52,
	PRE_GAME_START = 53,
	PRE_GAME_END = 54,

	# Coach
	COACH_DRAW = 60,
	COACH_PING = 61,
	COACH_CLEAR = 62,

	# Chat
	CHAT_MESSAGE = 70,
	QUICK_CHAT = 71,

	# Auth
	AUTH_LOGIN = 80,
	AUTH_REGISTER = 81,
	AUTH_RESPONSE = 82,
	AUTH_TOKEN = 83,
}

enum ClassType {
	COLLECTOR = 0,
	DEFENDER = 1,
	ATTACKER = 2,
}

enum Team {
	NONE = 0,
	ALPHA = 1,
	BRAVO = 2,
}

enum BuildingType {
	WALL = 0,
	TURRET = 1,
	SPEED_PATH = 2,
	SPEED_FIELD = 3,
	POWER_FIELD = 4,
	STEALTH_FIELD = 5,
	TELEPORT = 6,
}

enum GamePhase {
	WAITING = 0,
	PRE_GAME = 1,
	PLAYING = 2,
	ROUND_END = 3,
	MATCH_END = 4,
}

## Pack a message into a dictionary ready for JSON serialization.
static func pack_message(type: MessageType, data: Dictionary = {}) -> Dictionary:
	return {"t": type, "d": data, "ts": Time.get_ticks_msec()}

## Unpack a received JSON dictionary into type + data.
static func unpack_message(msg: Dictionary) -> Dictionary:
	return {
		"type": msg.get("t", -1),
		"data": msg.get("d", {}),
		"timestamp": msg.get("ts", 0),
	}
