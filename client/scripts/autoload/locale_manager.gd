extends Node

## LocaleManager — handles language switching and locale persistence.
## Autoloaded as "LocaleManager".
## All UI strings must use tr("KEY") — never hardcoded text.

signal locale_changed(locale: String)

const SUPPORTED_LOCALES := ["en", "ru"]
const DEFAULT_LOCALE := "ru"
const SAVE_KEY := "user://settings.cfg"


func _ready() -> void:
	_load_locale()


func set_locale(locale: String) -> void:
	if locale not in SUPPORTED_LOCALES:
		locale = DEFAULT_LOCALE

	TranslationServer.set_locale(locale)
	_save_locale(locale)
	locale_changed.emit(locale)


func get_current_locale() -> String:
	return TranslationServer.get_locale().substr(0, 2)


func get_supported_locales() -> Array[String]:
	var arr: Array[String] = []
	arr.assign(SUPPORTED_LOCALES)
	return arr


func _load_locale() -> void:
	var config := ConfigFile.new()
	if config.load(SAVE_KEY) == OK:
		var saved_locale: String = config.get_value("settings", "locale", DEFAULT_LOCALE)
		TranslationServer.set_locale(saved_locale)
	else:
		var system_locale := OS.get_locale_language()
		if system_locale in SUPPORTED_LOCALES:
			TranslationServer.set_locale(system_locale)
		else:
			TranslationServer.set_locale(DEFAULT_LOCALE)


func _save_locale(locale: String) -> void:
	var config := ConfigFile.new()
	config.load(SAVE_KEY)
	config.set_value("settings", "locale", locale)
	config.save(SAVE_KEY)
