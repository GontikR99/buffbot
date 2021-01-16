package storage

type ControllerConfig interface {
	// Settings established before startup
	EverQuestDirectory() string
	SelectedCharacter() string

	SetEverQuestDirectory(string)
	SetSelectedCharacter(string)

	GetConfItem(string) string
	HasConfItem(string) bool
	SetConfItem(name, value string)
}

type VoiceChannel struct {
	GuildID   string
	ChannelID string
}
