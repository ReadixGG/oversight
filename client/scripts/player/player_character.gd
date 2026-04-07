extends CharacterBody2D
class_name PlayerCharacter

## Base player character for top-down movement.
## Server-authoritative: this handles local prediction + visual interpolation.

@export var player_id: int = -1
@export var team: Protocol.Team = Protocol.Team.NONE
@export var player_class: Protocol.ClassType = Protocol.ClassType.COLLECTOR

@onready var sprite: Sprite2D = $Sprite2D
@onready var collision: CollisionShape2D = $CollisionShape2D
@onready var health_bar: ProgressBar = $HealthBar

var max_hp: float = 100.0
var current_hp: float = 100.0
var move_speed: float = 200.0
var is_local: bool = false
var is_dead: bool = false
var carrying_resource: bool = false

## Server reconciliation
var _input_sequence: int = 0
var _pending_inputs: Array[Dictionary] = []
var _server_position: Vector2 = Vector2.ZERO
var _server_velocity: Vector2 = Vector2.ZERO


func _ready() -> void:
	_apply_class_stats()
	current_hp = max_hp
	_update_health_bar()


func _physics_process(delta: float) -> void:
	if is_dead:
		return

	if is_local:
		_process_local_input(delta)
	else:
		_interpolate_to_server_pos(delta)


func _apply_class_stats() -> void:
	match player_class:
		Protocol.ClassType.COLLECTOR:
			max_hp = 80.0
			move_speed = 250.0
		Protocol.ClassType.DEFENDER:
			max_hp = 120.0
			move_speed = 180.0
		Protocol.ClassType.ATTACKER:
			max_hp = 100.0
			move_speed = 220.0


func _process_local_input(delta: float) -> void:
	var input_dir := _get_movement_input()
	if input_dir == Vector2.ZERO:
		velocity = Vector2.ZERO
	else:
		var speed := move_speed
		if carrying_resource:
			speed *= 0.7
		velocity = input_dir.normalized() * speed

	move_and_slide()

	if input_dir != Vector2.ZERO:
		_input_sequence += 1
		var input_data := {
			"seq": _input_sequence,
			"dx": input_dir.x,
			"dy": input_dir.y,
			"dt": delta,
		}
		_pending_inputs.append(input_data)
		NetworkManager.send_message(Protocol.MessageType.INPUT_MOVE, input_data)


func _get_movement_input() -> Vector2:
	return Input.get_vector("move_left", "move_right", "move_up", "move_down")


func apply_server_state(pos: Vector2, vel: Vector2, last_seq: int) -> void:
	_server_position = pos
	_server_velocity = vel

	if is_local:
		# Reconciliation: discard acknowledged inputs, replay pending ones
		_pending_inputs = _pending_inputs.filter(
			func(inp: Dictionary) -> bool: return inp["seq"] > last_seq
		)
		position = pos
		for inp in _pending_inputs:
			var dir := Vector2(inp["dx"], inp["dy"]).normalized()
			var speed := move_speed
			if carrying_resource:
				speed *= 0.7
			velocity = dir * speed
			move_and_slide()


func _interpolate_to_server_pos(delta: float) -> void:
	position = position.lerp(_server_position, 10.0 * delta)


func take_damage(amount: float) -> void:
	current_hp = clampf(current_hp - amount, 0.0, max_hp)
	_update_health_bar()
	_flash_damage()

	if current_hp <= 0.0:
		die()


func heal(amount: float) -> void:
	current_hp = clampf(current_hp + amount, 0.0, max_hp)
	_update_health_bar()


func die() -> void:
	is_dead = true
	visible = false
	collision.set_deferred("disabled", true)


func respawn(spawn_pos: Vector2) -> void:
	is_dead = false
	visible = true
	collision.set_deferred("disabled", false)
	position = spawn_pos
	current_hp = max_hp
	carrying_resource = false
	_update_health_bar()


func _update_health_bar() -> void:
	if health_bar:
		health_bar.value = (current_hp / max_hp) * 100.0


func _flash_damage() -> void:
	if sprite:
		var tween := create_tween()
		tween.tween_property(sprite, "modulate", Color.RED, 0.05)
		tween.tween_property(sprite, "modulate", Color.WHITE, 0.15)
