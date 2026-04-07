extends Node2D

## GameWorld — root of the game scene during a match.
## Manages players, map, buildings, resources.

const PlayerScene := preload("res://client/scenes/game/player.tscn")
const ProjectileScene := preload("res://client/scenes/game/projectile.tscn")

@onready var camera: Camera2D = $TopDownCamera
@onready var map_layer: TileMapLayer = $MapLayer
@onready var players_node: Node2D = $Players
@onready var buildings_node: Node2D = $Buildings
@onready var resources_node: Node2D = $Resources
@onready var projectiles_node: Node2D = $Projectiles
@onready var hud: CanvasLayer = $HUD

var players: Dictionary = {}  # player_id -> PlayerCharacter
var projectiles: Dictionary = {}  # projectile_id -> Projectile
var local_player: PlayerCharacter = null
var fog: FogOfWar = null
var map_tile_size: int = 32


func _ready() -> void:
	NetworkManager.message_received.connect(_on_network_message)

	fog = FogOfWar.new()
	add_child(fog)


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
		Protocol.MessageType.DAMAGE_DEALT:
			_on_damage_dealt(data)
		Protocol.MessageType.RESOURCE_COLLECTED:
			_on_resource_collected(data)


func _apply_snapshot(data: Dictionary) -> void:
	var player_states: Array = data.get("players", [])
	var ally_positions: Array[Vector2] = []

	for ps: Dictionary in player_states:
		var pid: int = ps.get("id", -1)
		if pid in players:
			var pos := Vector2(ps.get("x", 0.0), ps.get("y", 0.0))
			var vel := Vector2(ps.get("vx", 0.0), ps.get("vy", 0.0))
			var last_seq: int = ps.get("seq", 0)
			players[pid].apply_server_state(pos, vel, last_seq)
			players[pid].current_hp = ps.get("hp", players[pid].max_hp)
			players[pid].carrying_resource = ps.get("carrying", false)

			if players[pid].team == GameManager.local_team:
				ally_positions.append(pos)

	if fog:
		fog.update_fog(ally_positions)


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


func _load_map(data: Dictionary) -> void:
	MapLoader.load_map(map_layer, data)

	map_tile_size = data.get("tile_size", 32)
	var map_w: int = data.get("width", 64) * map_tile_size
	var map_h: int = data.get("height", 32) * map_tile_size

	if fog:
		fog.set_map_size(map_w, map_h)


func _spawn_projectile(data: Dictionary) -> void:
	var proj_id: int = data.get("id", -1)
	if proj_id in projectiles:
		return

	var proj: Projectile = ProjectileScene.instantiate()
	proj.projectile_id = proj_id
	proj.owner_id = data.get("owner", -1)
	proj.team = data.get("team", Protocol.Team.NONE) as Protocol.Team
	proj.position = Vector2(data.get("x", 0.0), data.get("y", 0.0))
	proj.direction = Vector2(data.get("dx", 1.0), data.get("dy", 0.0)).normalized()
	proj.speed = data.get("speed", 600.0)
	proj.damage = data.get("damage", 10.0)

	# Color projectile by team
	var proj_sprite: Sprite2D = proj.get_node("Sprite2D")
	if proj_sprite:
		if proj.team == GameManager.local_team:
			proj_sprite.modulate = Color(0.3, 0.7, 1.0)  # blue for allies
		else:
			proj_sprite.modulate = Color(1.0, 0.3, 0.3)  # red for enemies

	projectiles_node.add_child(proj)
	projectiles[proj_id] = proj

	# Auto-cleanup when projectile dies
	proj.tree_exited.connect(func() -> void:
		projectiles.erase(proj_id)
	)


func _on_damage_dealt(data: Dictionary) -> void:
	var target_id: int = data.get("target", -1)
	var amount: float = data.get("amount", 0.0)
	if target_id in players:
		players[target_id].take_damage(amount)


func _place_building(_data: Dictionary) -> void:
	# TODO: implement building placement visual
	pass


func _destroy_building(_data: Dictionary) -> void:
	# TODO: implement building destruction visual
	pass


func _on_resource_collected(_data: Dictionary) -> void:
	# TODO: implement resource collection visual
	pass
