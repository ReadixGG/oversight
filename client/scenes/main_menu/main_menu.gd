extends Control

## Main menu screen.

@onready var play_btn: Button = $VBoxContainer/PlayButton
@onready var profile_btn: Button = $VBoxContainer/ProfileButton
@onready var friends_btn: Button = $VBoxContainer/FriendsButton
@onready var settings_btn: Button = $VBoxContainer/SettingsButton
@onready var title_label: Label = $TitleLabel
@onready var connection_label: Label = $ConnectionStatus


func _ready() -> void:
	play_btn.text = tr("BTN_PLAY")
	profile_btn.text = tr("BTN_PROFILE")
	friends_btn.text = tr("BTN_FRIENDS")
	settings_btn.text = tr("BTN_SETTINGS")
	title_label.text = tr("GAME_TITLE")

	play_btn.pressed.connect(_on_play)
	settings_btn.pressed.connect(_on_settings)

	NetworkManager.connected.connect(func() -> void:
		connection_label.text = tr("NETWORK_CONNECTED")
		connection_label.modulate = Color.GREEN
	)
	NetworkManager.disconnected.connect(func() -> void:
		connection_label.text = tr("NETWORK_DISCONNECTED")
		connection_label.modulate = Color.RED
	)
	NetworkManager.connection_error.connect(func(reason: String) -> void:
		connection_label.text = reason
		connection_label.modulate = Color.RED
	)

	NetworkManager.connect_to_server()


func _on_play() -> void:
	NetworkManager.send_message(Protocol.MessageType.FIND_MATCH)
	# TODO: switch to matchmaking screen


func _on_settings() -> void:
	# TODO: open settings screen
	pass
