extends Node

## GameManager — global game state singleton.
## Autoloaded as "GameManager".

signal phase_changed(new_phase: Protocol.GamePhase)
signal round_started(round_number: int)
signal round_ended(winner_team: Protocol.Team)
signal match_ended(winner_team: Protocol.Team)

var current_phase: Protocol.GamePhase = Protocol.GamePhase.WAITING
var current_round: int = 0
var score: Dictionary = {
	Protocol.Team.ALPHA: 0,
	Protocol.Team.BRAVO: 0,
}

var local_player_id: int = -1
var local_team: Protocol.Team = Protocol.Team.NONE
var local_class: Protocol.ClassType = Protocol.ClassType.COLLECTOR

var match_id: String = ""
var round_timer: float = 0.0
var round_duration: float = 720.0  # 12 minutes default


func set_phase(phase: Protocol.GamePhase) -> void:
	current_phase = phase
	phase_changed.emit(phase)


func start_round(round_num: int) -> void:
	current_round = round_num
	round_timer = round_duration
	set_phase(Protocol.GamePhase.PLAYING)
	round_started.emit(round_num)


func end_round(winner: Protocol.Team) -> void:
	score[winner] += 1
	set_phase(Protocol.GamePhase.ROUND_END)
	round_ended.emit(winner)

	if score[winner] >= 2:
		end_match(winner)


func end_match(winner: Protocol.Team) -> void:
	set_phase(Protocol.GamePhase.MATCH_END)
	match_ended.emit(winner)


func reset() -> void:
	current_phase = Protocol.GamePhase.WAITING
	current_round = 0
	score = {
		Protocol.Team.ALPHA: 0,
		Protocol.Team.BRAVO: 0,
	}
	match_id = ""
	round_timer = 0.0


func _process(delta: float) -> void:
	if current_phase == Protocol.GamePhase.PLAYING:
		round_timer -= delta
		if round_timer <= 0.0:
			round_timer = 0.0
