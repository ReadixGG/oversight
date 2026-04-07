extends Node
class_name MapLoader

## Loads map data received from the server into a TileMapLayer.

const TILE_GROUND := 0
const TILE_WALL := 1
const TILE_WATER := 2
const TILE_RESOURCE := 3
const TILE_BASE_ALPHA := 4
const TILE_BASE_BRAVO := 5


static func load_map(tile_map: TileMapLayer, data: Dictionary) -> void:
	var width: int = data.get("width", 0)
	var height: int = data.get("height", 0)
	var tiles: Array = data.get("tiles", [])

	if tiles.size() != width * height:
		return

	tile_map.clear()

	for y in range(height):
		for x in range(width):
			var tile_type: int = tiles[y * width + x]
			var atlas_coords := _tile_type_to_atlas(tile_type)
			tile_map.set_cell(Vector2i(x, y), 0, atlas_coords)


static func _tile_type_to_atlas(tile_type: int) -> Vector2i:
	match tile_type:
		TILE_GROUND:
			return Vector2i(0, 0)
		TILE_WALL:
			return Vector2i(1, 0)
		TILE_WATER:
			return Vector2i(2, 0)
		TILE_RESOURCE:
			return Vector2i(3, 0)
		TILE_BASE_ALPHA:
			return Vector2i(4, 0)
		TILE_BASE_BRAVO:
			return Vector2i(5, 0)
		_:
			return Vector2i(0, 0)
