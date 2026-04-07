extends Camera2D

## Top-down camera that follows a target node smoothly.

@export var target: Node2D
@export var follow_speed: float = 8.0
@export var default_zoom_level: float = 1.5
@export var min_zoom: float = 0.8
@export var max_zoom: float = 3.0

var _target_zoom: float


func _ready() -> void:
	_target_zoom = default_zoom_level
	zoom = Vector2(_target_zoom, _target_zoom)
	make_current()


func _process(delta: float) -> void:
	if target and is_instance_valid(target):
		global_position = global_position.lerp(target.global_position, follow_speed * delta)

	zoom = zoom.lerp(Vector2(_target_zoom, _target_zoom), 5.0 * delta)


func set_zoom_level(level: float) -> void:
	_target_zoom = clampf(level, min_zoom, max_zoom)


func zoom_in(amount: float = 0.1) -> void:
	set_zoom_level(_target_zoom + amount)


func zoom_out(amount: float = 0.1) -> void:
	set_zoom_level(_target_zoom - amount)
