package bot

import (
	"context"

	"github.com/andersfylling/disgord"
)

// These functions add some extra functionality to disgord.Session.

// SuppressEmbeds suppresses all embeds on the given message.
func SuppressEmbeds(ctx context.Context, session disgord.Session, msg *disgord.Message) (*disgord.Message, error) {
	return session.UpdateMessage(
		ctx,
		msg.ChannelID,
		msg.ID,
	).Set(
		"flags",
		msg.Flags|disgord.MessageFlagSupressEmbeds,
	).Execute()
}

// UnsuppressEmbeds unsuppresses all embeds on the given message.
func UnsuppressEmbeds(ctx context.Context, session disgord.Session, msg *disgord.Message) (*disgord.Message, error) {
	return session.UpdateMessage(
		ctx,
		msg.ChannelID,
		msg.ID,
	).Set(
		"flags",
		msg.Flags&^disgord.MessageFlagSupressEmbeds,
	).Execute()
}
