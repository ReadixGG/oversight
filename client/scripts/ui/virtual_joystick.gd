extends Control
class_name VirtualJoystick

## Mobile virtual joystick for touch input.
## Emits normalized direction vector.

signal joystick_input(direction: Vector2)
signal joystick_released()

@export var dead_zone: float = 10.0
@export var clamp_radius: float = 64.0
@export_enum("Left", "Right") var side: String = "Left"

@onready var base: TextureRect = $Base
@onready var knob: TextureRect = $Base/Knob

var _touch_index: int = -1
var _center: Vector2 = Vector2.ZERO
var _output: Vector2 = Vector2.ZERO


func _ready() -> void:
	_center = base.size / 2.0
	knob.position = _center - knob.size / 2.0


func get_output() -> Vector2:
	return _output


func _input(event: InputEvent) -> void:
	if event is InputEventScreenTouch:
		_handle_touch(event as InputEventScreenTouch)
	elif event is InputEventScreenDrag:
		_handle_drag(event as InputEventScreenDrag)


func _handle_touch(event: InputEventScreenTouch) -> void:
	if event.pressed:
		if _is_in_zone(event.position) and _touch_index == -1:
			_touch_index = event.index
	else:
		if event.index == _touch_index:
			_reset()


func _handle_drag(event: InputEventScreenDrag) -> void:
	if event.index != _touch_index:
		return

	var local_pos := base.get_global_transform().affine_inverse() * event.position
	var diff := local_pos - _center
	var dist := diff.length()

	if dist < dead_zone:
		_output = Vector2.ZERO
		knob.position = _center - knob.size / 2.0
		return

	if dist > clamp_radius:
		diff = diff.normalized() * clamp_radius

	_output = diff / clamp_radius
	knob.position = _center + diff - knob.size / 2.0

	joystick_input.emit(_output)


func _is_in_zone(screen_pos: Vector2) -> bool:
	var viewport_size := get_viewport_rect().size
	if side == "Left":
		return screen_pos.x < viewport_size.x * 0.5
	else:
		return screen_pos.x >= viewport_size.x * 0.5


func _reset() -> void:
	_touch_index = -1
	_output = Vector2.ZERO
	knob.position = _center - knob.size / 2.0
	joystick_released.emit()
