extends Node

## NetworkManager — handles WebSocket connection to game server.
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
var rtt_ms: int = 0  # round-trip time


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
		send_message(Protocol.MessageType.DISCONNECT)
	_socket.close()
	_connected = false
	set_process(false)
	disconnected.emit()


func send_message(type: Protocol.MessageType, data: Dictionary = {}) -> void:
	if not _connected:
		return

	var msg := Protocol.pack_message(type, data)
	var json_str := JSON.stringify(msg)
	_socket.send_text(json_str)


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
	send_message(Protocol.MessageType.HANDSHAKE, {
		"version": "0.1.0",
	})


func _send_ping() -> void:
	_last_ping_time = Time.get_ticks_msec()
	send_message(Protocol.MessageType.PING)


func _handle_packet(pkt: PackedByteArray) -> void:
	var text := pkt.get_string_from_utf8()
	var json := JSON.new()
	if json.parse(text) != OK:
		return

	var msg: Dictionary = json.data
	var unpacked := Protocol.unpack_message(msg)
	var msg_type: int = unpacked["type"]
	var msg_data: Dictionary = unpacked["data"]

	if msg_type == Protocol.MessageType.PONG:
		rtt_ms = Time.get_ticks_msec() - _last_ping_time
		return

	message_received.emit(msg_type, msg_data)


func _try_reconnect() -> void:
	_reconnect_count += 1
	if _reconnect_count > RECONNECT_ATTEMPTS:
		connection_error.emit(tr("NETWORK_RECONNECT_FAILED"))
		return

	await get_tree().create_timer(RECONNECT_DELAY).timeout
	connect_to_server()
