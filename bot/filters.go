package bot

import (
	"strings"

	"github.com/andersfylling/disgord"
)

// Filters MessageCreate events, allowing only non-bot authors.
func filterMsgCreateNotBot(evt interface{}) interface{} {
	evtMsgCreate := evt.(*disgord.MessageCreate)
	if evtMsgCreate.Message.Author == nil || evtMsgCreate.Message.Author.Bot {
		return nil
	}
	return evt
}

// Returns a filter for MessageCreate events, which allows messages with the given prefix only.
func filterMsgCreatePrefix(prefix string) func(evt interface{}) interface{} {
	return func(evt interface{}) interface{} {
		evtMsgCreate := evt.(*disgord.MessageCreate)
		if !strings.HasPrefix(evtMsgCreate.Message.Content, prefix) {
			return nil
		}
		return evt
	}
}

// Returns a filter for MessageReactionAdd events, which allows reactions on the given message ID
// only.
func filterReactionAddForMsg(msgID disgord.Snowflake) func(interface{}) interface{} {
	return func(evt interface{}) interface{} {
		evtReactionAdd := evt.(*disgord.MessageReactionAdd)
		if evtReactionAdd.MessageID != msgID {
			return nil
		}
		return evt
	}
}
