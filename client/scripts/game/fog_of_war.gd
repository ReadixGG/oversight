extends CanvasLayer
class_name FogOfWar

## Client-side fog of war rendering.
## The server already filters enemy data — this is purely visual.
## Renders a dark overlay with holes cut for visible areas around allies.

@export var fog_color: Color = Color(0.0, 0.0, 0.0, 0.7)
@export var vision_radius: float = 400.0
@export var fog_resolution: int = 4

var _fog_image: Image
var _fog_texture: ImageTexture
var _fog_sprite: Sprite2D

var _map_width: int = 2048
var _map_height: int = 1024
var _ally_positions: Array[Vector2] = []


func _ready() -> void:
	var scaled_w := _map_width / fog_resolution
	var scaled_h := _map_height / fog_resolution

	_fog_image = Image.create(scaled_w, scaled_h, false, Image.FORMAT_RGBA8)
	_fog_texture = ImageTexture.create_from_image(_fog_image)

	_fog_sprite = Sprite2D.new()
	_fog_sprite.texture = _fog_texture
	_fog_sprite.scale = Vector2(fog_resolution, fog_resolution)
	_fog_sprite.centered = false
	add_child(_fog_sprite)


func set_map_size(width: int, height: int) -> void:
	_map_width = width
	_map_height = height

	var scaled_w := _map_width / fog_resolution
	var scaled_h := _map_height / fog_resolution
	_fog_image = Image.create(scaled_w, scaled_h, false, Image.FORMAT_RGBA8)
	_fog_texture.set_image(_fog_image)


func update_fog(ally_positions: Array[Vector2]) -> void:
	_ally_positions = ally_positions
	_render_fog()


func _render_fog() -> void:
	var w := _fog_image.get_width()
	var h := _fog_image.get_height()
	var scaled_radius := vision_radius / fog_resolution

	_fog_image.fill(fog_color)

	for ally_pos in _ally_positions:
		var cx := int(ally_pos.x / fog_resolution)
		var cy := int(ally_pos.y / fog_resolution)
		var r := int(scaled_radius)

		for dy in range(-r, r + 1):
			for dx in range(-r, r + 1):
				var px := cx + dx
				var py := cy + dy
				if px < 0 or px >= w or py < 0 or py >= h:
					continue

				var dist := sqrt(float(dx * dx + dy * dy))
				if dist <= scaled_radius:
					var alpha := 0.0
					if dist > scaled_radius * 0.7:
						alpha = remap(dist, scaled_radius * 0.7, scaled_radius, 0.0, fog_color.a)
					_fog_image.set_pixel(px, py, Color(0.0, 0.0, 0.0, alpha))

	_fog_texture.update(_fog_image)
