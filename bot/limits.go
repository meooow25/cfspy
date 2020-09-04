package bot

import "github.com/andersfylling/disgord"

// See
// https://discord.com/developers/docs/resources/channel#create-message
// https://discord.com/developers/docs/resources/channel#embed-limits
const (
	contentCharLimit = 2000

	titleCharLimit       = 256
	descriptionCharLimit = 2048
	fieldCountLimit      = 25
	fieldNameCharLimit   = 256
	fieldValueCharLimit  = 1024
	footerTextCharLimit  = 2048
	authorNameCharLimit  = 256
)

// ContentTooLong checks whether the given message content is longer than the acceptable limit.
func ContentTooLong(content string) bool {
	return len(content) > contentCharLimit
}

// EmbedDescriptionTooLong checks whether the given embed has description longer than the acceptable
// limit.
func EmbedDescriptionTooLong(embed *disgord.Embed) bool {
	return len(embed.Description) > descriptionCharLimit
}
