extends Area2D
class_name Projectile

## Visual projectile on the client side.
## Server is authoritative — this is just a visual representation.

var projectile_id: int = -1
var owner_id: int = -1
var team: Protocol.Team = Protocol.Team.NONE
var direction: Vector2 = Vector2.RIGHT
var speed: float = 600.0
var damage: float = 10.0
var lifetime: float = 2.0

var _server_pos: Vector2 = Vector2.ZERO
var _age: float = 0.0


func _ready() -> void:
	rotation = direction.angle()


func _process(delta: float) -> void:
	position += direction * speed * delta
	_age += delta
	if _age >= lifetime:
		queue_free()


func update_from_server(pos: Vector2) -> void:
	_server_pos = pos
	position = position.lerp(_server_pos, 0.5)
