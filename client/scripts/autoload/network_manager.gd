extends Node

## NetworkManager — handles WebSocket connection to game server.
## Now uses binary wire format: [2 bytes LE: msg_type][payload]
## Autoloaded as "NetworkManager".

signal connected()
signal disconnected()
signal connection_error(reason: String)
signal message_received(type: int, data: Dictionary)

const RECONNECT_ATTEMPTS := 3
const RECONNECT_DELAY := 2.0
const PING_INTERVAL := 5.0

var server_url: String = "ws://127.0.0.1:8080/ws"

var _socket: WebSocketPeer = WebSocketPeer.new()
var _connected: bool = false
var _reconnect_count: int = 0
var _ping_timer: float = 0.0
var _last_ping_time: int = 0
var rtt_ms: int = 0


func _ready() -> void:
	set_process(false)


func connect_to_server(url: String = "") -> void:
	if url != "":
		server_url = url

	var err := _socket.connect_to_url(server_url)
	if err != OK:
		connection_error.emit(tr("NETWORK_CONNECT_FAILED"))
		return

	set_process(true)
	_reconnect_count = 0


func disconnect_from_server() -> void:
	if _connected:
		send_raw(Protocol.encode_disconnect())
	_socket.close()
	_connected = false
	set_process(false)
	disconnected.emit()


## Send raw binary packet
func send_raw(data: PackedByteArray) -> void:
	if not _connected:
		return
	_socket.send(data)


## Convenience: encode and send INPUT_MOVE
func send_input_move(dx: float, dy: float, dt: float, seq: int) -> void:
	send_raw(Protocol.encode_input_move(dx, dy, dt, seq))


## Convenience: encode and send INPUT_SHOOT
func send_input_shoot(dx: float, dy: float, x: float, y: float) -> void:
	send_raw(Protocol.encode_input_shoot(dx, dy, x, y))


## Convenience: encode and send FIND_MATCH
func send_find_match() -> void:
	send_raw(Protocol.encode_find_match())


## Convenience: encode and send CANCEL_SEARCH
func send_cancel_search() -> void:
	send_raw(Protocol.encode_cancel_search())


## Convenience: encode and send SELECT_CLASS
func send_select_class(class_type: int) -> void:
	send_raw(Protocol.encode_select_class(class_type))


func _process(delta: float) -> void:
	_socket.poll()

	var state := _socket.get_ready_state()

	match state:
		WebSocketPeer.STATE_OPEN:
			if not _connected:
				_connected = true
				_reconnect_count = 0
				connected.emit()
				_send_handshake()

			_ping_timer += delta
			if _ping_timer >= PING_INTERVAL:
				_ping_timer = 0.0
				_send_ping()

			while _socket.get_available_packet_count() > 0:
				var pkt := _socket.get_packet()
				_handle_packet(pkt)

		WebSocketPeer.STATE_CLOSING:
			pass

		WebSocketPeer.STATE_CLOSED:
			if _connected:
				_connected = false
				disconnected.emit()
				_try_reconnect()
			set_process(false)


func _send_handshake() -> void:
	send_raw(Protocol.encode_handshake("0.1.0"))


func _send_ping() -> void:
	_last_ping_time = Time.get_ticks_msec()
	send_raw(Protocol.encode_ping())


func _handle_packet(pkt: PackedByteArray) -> void:
	if pkt.size() < 2:
		return

	var msg_type := Protocol.decode_msg_type(pkt)
	var r := Protocol.decode_payload(pkt)

	match msg_type:
		Protocol.MessageType.PONG:
			rtt_ms = Time.get_ticks_msec() - _last_ping_time
			return

		Protocol.MessageType.HANDSHAKE_OK:
			var d := Protocol.decode_handshake_ok(r)
			GameManager.local_player_id = d["player_id"]
			return

		Protocol.MessageType.MATCH_FOUND:
			message_received.emit(msg_type, Protocol.decode_match_found(r))

		Protocol.MessageType.PLAYER_SPAWNED:
			message_received.emit(msg_type, Protocol.decode_player_spawned(r))

		Protocol.MessageType.PLAYER_DIED:
			message_received.emit(msg_type, Protocol.decode_player_died(r))

		Protocol.MessageType.PLAYER_RESPAWNED:
			message_received.emit(msg_type, Protocol.decode_player_respawned(r))

		Protocol.MessageType.DAMAGE_DEALT:
			message_received.emit(msg_type, Protocol.decode_damage_dealt(r))

		Protocol.MessageType.PROJECTILE_SPAWNED:
			message_received.emit(msg_type, Protocol.decode_projectile_spawned(r))

		Protocol.MessageType.GAME_SNAPSHOT:
			message_received.emit(msg_type, Protocol.decode_game_snapshot(r))

		Protocol.MessageType.MAP_DATA:
			message_received.emit(msg_type, Protocol.decode_map_data(r))

		Protocol.MessageType.ROUND_START:
			message_received.emit(msg_type, Protocol.decode_round_start(r))

		Protocol.MessageType.ROUND_END:
			message_received.emit(msg_type, Protocol.decode_round_end(r))

		Protocol.MessageType.MATCH_END:
			message_received.emit(msg_type, Protocol.decode_match_end(r))

		Protocol.MessageType.PRE_GAME_START:
			message_received.emit(msg_type, Protocol.decode_pre_game_start(r))

		_:
			# Unknown or unhandled message type
			pass


func _try_reconnect() -> void:
	_reconnect_count += 1
	if _reconnect_count > RECONNECT_ATTEMPTS:
		connection_error.emit(tr("NETWORK_RECONNECT_FAILED"))
		return

	await get_tree().create_timer(RECONNECT_DELAY).timeout
	connect_to_server()
