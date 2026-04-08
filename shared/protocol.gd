class_name Protocol

## Binary wire protocol for OverSight.
## Format: [2 bytes LE: msg_type][payload bytes...]
## All numbers are little-endian. Floats are 32-bit. IDs are 64-bit.
## Keep in sync with server/internal/protocol/binary.go

enum MessageType {
	HANDSHAKE = 1,
	HANDSHAKE_OK = 2,
	DISCONNECT = 3,
	PING = 4,
	PONG = 5,

	FIND_MATCH = 10,
	MATCH_FOUND = 11,
	CANCEL_SEARCH = 12,
	SELECT_CLASS = 13,
	LOBBY_STATE = 14,
	MATCH_START = 15,

	INPUT_MOVE = 20,
	INPUT_SHOOT = 21,
	INPUT_ABILITY = 22,
	INPUT_BUILD = 23,
	INPUT_COLLECT = 24,
	INPUT_DROP = 25,

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

	MAP_DATA = 40,
	FOG_UPDATE = 41,

	ROUND_START = 50,
	ROUND_END = 51,
	MATCH_END = 52,
	PRE_GAME_START = 53,
	PRE_GAME_END = 54,

	COACH_DRAW = 60,
	COACH_PING = 61,
	COACH_CLEAR = 62,

	CHAT_MESSAGE = 70,
	QUICK_CHAT = 71,

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


# ============================================================
#  BufWriter — binary encoder (little-endian)
# ============================================================

class BufWriter:
	var buf: PackedByteArray = PackedByteArray()

	func write_u16(v: int) -> void:
		buf.append(v & 0xFF)
		buf.append((v >> 8) & 0xFF)

	func write_u32(v: int) -> void:
		buf.append(v & 0xFF)
		buf.append((v >> 8) & 0xFF)
		buf.append((v >> 16) & 0xFF)
		buf.append((v >> 24) & 0xFF)

	func write_u64(v: int) -> void:
		for i in range(8):
			buf.append((v >> (i * 8)) & 0xFF)

	func write_f32(v: float) -> void:
		var tmp := PackedFloat32Array([v])
		buf.append_array(tmp.to_byte_array())

	func write_bool(v: bool) -> void:
		buf.append(1 if v else 0)

	func write_bytes(data: PackedByteArray) -> void:
		write_u32(data.size())
		buf.append_array(data)

	func write_string(s: String) -> void:
		write_bytes(s.to_utf8_buffer())


# ============================================================
#  BufReader — binary decoder (little-endian)
# ============================================================

class BufReader:
	var data: PackedByteArray
	var pos: int = 0

	func _init(d: PackedByteArray) -> void:
		data = d

	func remaining() -> int:
		return data.size() - pos

	func read_u16() -> int:
		if remaining() < 2:
			return 0
		var v := data[pos] | (data[pos + 1] << 8)
		pos += 2
		return v

	func read_u32() -> int:
		if remaining() < 4:
			return 0
		var v := data[pos] | (data[pos + 1] << 8) | (data[pos + 2] << 16) | (data[pos + 3] << 24)
		pos += 4
		return v

	func read_u64() -> int:
		if remaining() < 8:
			return 0
		var v: int = 0
		for i in range(8):
			v |= data[pos + i] << (i * 8)
		pos += 8
		return v

	func read_f32() -> float:
		if remaining() < 4:
			return 0.0
		var slice := data.slice(pos, pos + 4)
		pos += 4
		var arr := slice.to_float32_array()
		if arr.size() > 0:
			return arr[0]
		return 0.0

	func read_bool() -> bool:
		if remaining() < 1:
			return false
		var v := data[pos]
		pos += 1
		return v != 0

	func read_bytes() -> PackedByteArray:
		var length := read_u32()
		if remaining() < length:
			return PackedByteArray()
		var v := data.slice(pos, pos + length)
		pos += length
		return v

	func read_string() -> String:
		return read_bytes().get_string_from_utf8()


# ============================================================
#  Message encoding (client -> server)
# ============================================================

static func encode_message(msg_type: int, payload: PackedByteArray = PackedByteArray()) -> PackedByteArray:
	var w := BufWriter.new()
	w.write_u16(msg_type)
	w.buf.append_array(payload)
	return w.buf


static func encode_handshake(version: String) -> PackedByteArray:
	var w := BufWriter.new()
	w.write_u16(MessageType.HANDSHAKE)
	w.write_string(version)
	return w.buf


static func encode_ping() -> PackedByteArray:
	var w := BufWriter.new()
	w.write_u16(MessageType.PING)
	return w.buf


static func encode_disconnect() -> PackedByteArray:
	var w := BufWriter.new()
	w.write_u16(MessageType.DISCONNECT)
	return w.buf


static func encode_find_match() -> PackedByteArray:
	var w := BufWriter.new()
	w.write_u16(MessageType.FIND_MATCH)
	return w.buf


static func encode_cancel_search() -> PackedByteArray:
	var w := BufWriter.new()
	w.write_u16(MessageType.CANCEL_SEARCH)
	return w.buf


static func encode_select_class(class_type: int) -> PackedByteArray:
	var w := BufWriter.new()
	w.write_u16(MessageType.SELECT_CLASS)
	w.write_u32(class_type)
	return w.buf


static func encode_input_move(dx: float, dy: float, dt: float, seq: int) -> PackedByteArray:
	var w := BufWriter.new()
	w.write_u16(MessageType.INPUT_MOVE)
	w.write_f32(dx)
	w.write_f32(dy)
	w.write_f32(dt)
	w.write_u32(seq)
	return w.buf


static func encode_input_shoot(dx: float, dy: float, x: float, y: float) -> PackedByteArray:
	var w := BufWriter.new()
	w.write_u16(MessageType.INPUT_SHOOT)
	w.write_f32(dx)
	w.write_f32(dy)
	w.write_f32(x)
	w.write_f32(y)
	return w.buf


# ============================================================
#  Message decoding (server -> client)
# ============================================================

static func decode_msg_type(data: PackedByteArray) -> int:
	if data.size() < 2:
		return -1
	return data[0] | (data[1] << 8)


static func decode_payload(data: PackedByteArray) -> BufReader:
	return BufReader.new(data.slice(2) if data.size() > 2 else PackedByteArray())


static func decode_handshake_ok(r: BufReader) -> Dictionary:
	return {"player_id": r.read_u64()}


static func decode_match_found(r: BufReader) -> Dictionary:
	return {"match_id": r.read_string()}


static func decode_player_spawned(r: BufReader) -> Dictionary:
	return {
		"id": r.read_u64(),
		"team": r.read_u32(),
		"class": r.read_u32(),
		"x": r.read_f32(),
		"y": r.read_f32(),
	}


static func decode_player_died(r: BufReader) -> Dictionary:
	return {"id": r.read_u64(), "killer": r.read_u64()}


static func decode_player_respawned(r: BufReader) -> Dictionary:
	return {"id": r.read_u64(), "x": r.read_f32(), "y": r.read_f32()}


static func decode_damage_dealt(r: BufReader) -> Dictionary:
	return {"target": r.read_u64(), "attacker": r.read_u64(), "amount": r.read_f32()}


static func decode_projectile_spawned(r: BufReader) -> Dictionary:
	return {
		"id": r.read_u64(),
		"owner": r.read_u64(),
		"team": r.read_u32(),
		"x": r.read_f32(),
		"y": r.read_f32(),
		"dx": r.read_f32(),
		"dy": r.read_f32(),
		"speed": r.read_f32(),
		"damage": r.read_f32(),
	}


static func decode_game_snapshot(r: BufReader) -> Dictionary:
	var timer := r.read_f32()
	var count := r.read_u32()
	var players: Array[Dictionary] = []
	for i in range(count):
		players.append({
			"id": r.read_u64(),
			"x": r.read_f32(),
			"y": r.read_f32(),
			"vx": r.read_f32(),
			"vy": r.read_f32(),
			"hp": r.read_f32(),
			"carrying": r.read_bool(),
			"seq": r.read_u32(),
		})
	return {"timer": timer, "players": players}


static func decode_map_data(r: BufReader) -> Dictionary:
	var width := r.read_u32()
	var height := r.read_u32()
	var tile_size := r.read_u32()
	var tiles_bytes := r.read_bytes()
	var seed := r.read_u64()

	var tiles: Array[int] = []
	tiles.resize(tiles_bytes.size())
	for i in range(tiles_bytes.size()):
		tiles[i] = tiles_bytes[i]

	return {
		"width": width,
		"height": height,
		"tile_size": tile_size,
		"tiles": tiles,
		"seed": seed,
	}


static func decode_round_start(r: BufReader) -> Dictionary:
	return {"round": r.read_u32(), "duration": r.read_f32()}


static func decode_round_end(r: BufReader) -> Dictionary:
	return {
		"winner": r.read_u32(),
		"round": r.read_u32(),
		"score_alpha": r.read_u32(),
		"score_bravo": r.read_u32(),
	}


static func decode_match_end(r: BufReader) -> Dictionary:
	return {
		"winner": r.read_u32(),
		"score_alpha": r.read_u32(),
		"score_bravo": r.read_u32(),
	}


static func decode_pre_game_start(r: BufReader) -> Dictionary:
	return {"duration": r.read_f32()}
