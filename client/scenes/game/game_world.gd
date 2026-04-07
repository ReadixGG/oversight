extends Node2D

## GameWorld — root of the game scene during a match.
## Manages players, map, buildings, resources.

const PlayerScene := preload("res://client/scenes/game/player.tscn")

@onready var camera: Camera2D = $TopDownCamera
@onready var map_layer: TileMapLayer = $MapLayer
@onready var players_node: Node2D = $Players
@onready var buildings_node: Node2D = $Buildings
@onready var resources_node: Node2D = $Resources
@onready var projectiles_node: Node2D = $Projectiles
@onready var hud: CanvasLayer = $HUD

var players: Dictionary = {}  # player_id -> PlayerCharacter
var local_player: PlayerCharacter = null


func _ready() -> void:
	NetworkManager.message_received.connect(_on_network_message)


func _on_network_message(type: int, data: Dictionary) -> void:
	match type:
		Protocol.MessageType.GAME_SNAPSHOT:
			_apply_snapshot(data)
		Protocol.MessageType.PLAYER_SPAWNED:
			_spawn_player(data)
		Protocol.MessageType.PLAYER_DIED:
			_on_player_died(data)
		Protocol.MessageType.PLAYER_RESPAWNED:
			_on_player_respawned(data)
		Protocol.MessageType.MAP_DATA:
			_load_map(data)
		Protocol.MessageType.PROJECTILE_SPAWNED:
			_spawn_projectile(data)
		Protocol.MessageType.BUILDING_PLACED:
			_place_building(data)
		Protocol.MessageType.BUILDING_DESTROYED:
			_destroy_building(data)
		Protocol.MessageType.RESOURCE_COLLECTED:
			_on_resource_collected(data)


func _apply_snapshot(data: Dictionary) -> void:
	var player_states: Array = data.get("players", [])
	for ps: Dictionary in player_states:
		var pid: int = ps.get("id", -1)
		if pid in players:
			var pos := Vector2(ps.get("x", 0.0), ps.get("y", 0.0))
			var vel := Vector2(ps.get("vx", 0.0), ps.get("vy", 0.0))
			var last_seq: int = ps.get("seq", 0)
			players[pid].apply_server_state(pos, vel, last_seq)
			players[pid].current_hp = ps.get("hp", players[pid].max_hp)
			players[pid].carrying_resource = ps.get("carrying", false)


func _spawn_player(data: Dictionary) -> void:
	var pid: int = data.get("id", -1)
	if pid in players:
		return

	var player_node: PlayerCharacter = PlayerScene.instantiate()
	player_node.player_id = pid
	player_node.team = data.get("team", Protocol.Team.NONE) as Protocol.Team
	player_node.player_class = data.get("class", Protocol.ClassType.COLLECTOR) as Protocol.ClassType
	player_node.position = Vector2(data.get("x", 0.0), data.get("y", 0.0))

	if pid == GameManager.local_player_id:
		player_node.is_local = true
		local_player = player_node
		camera.target = player_node

	players_node.add_child(player_node)
	players[pid] = player_node


func _on_player_died(data: Dictionary) -> void:
	var pid: int = data.get("id", -1)
	if pid in players:
		players[pid].die()


func _on_player_respawned(data: Dictionary) -> void:
	var pid: int = data.get("id", -1)
	if pid in players:
		var spawn_pos := Vector2(data.get("x", 0.0), data.get("y", 0.0))
		players[pid].respawn(spawn_pos)


func _load_map(_data: Dictionary) -> void:
	# TODO: parse tile data from server and populate TileMapLayer
	pass


func _spawn_projectile(_data: Dictionary) -> void:
	# TODO: implement projectile visual
	pass


func _place_building(_data: Dictionary) -> void:
	# TODO: implement building placement visual
	pass


func _destroy_building(_data: Dictionary) -> void:
	# TODO: implement building destruction visual
	pass


func _on_resource_collected(_data: Dictionary) -> void:
	# TODO: implement resource collection visual
	pass
