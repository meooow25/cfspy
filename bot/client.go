package bot

import (
	"github.com/andersfylling/disgord"
)

// These functions add some extra functionality to disgord.Session.

// QueryBuilderFor returns a message query builder for the given message.
func QueryBuilderFor(s disgord.Session, msg *disgord.Message) disgord.MessageQueryBuilder {
	return s.Channel(msg.ChannelID).Message(msg.ID)
}

// SuppressEmbeds suppresses all embeds on the given message.
func SuppressEmbeds(session disgord.Session, msg *disgord.Message) (*disgord.Message, error) {
	return QueryBuilderFor(session, msg).Update().
		Set("flags", msg.Flags|disgord.MessageFlagSupressEmbeds).Execute()
}

// UnsuppressEmbeds unsuppresses all embeds on the given message.
func UnsuppressEmbeds(session disgord.Session, msg *disgord.Message) (*disgord.Message, error) {
	return QueryBuilderFor(session, msg).Update().
		Set("flags", msg.Flags&^disgord.MessageFlagSupressEmbeds).Execute()
}
