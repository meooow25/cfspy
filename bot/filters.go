package bot

import (
	"fmt"
	"strings"

	"github.com/andersfylling/disgord"
)

// Filters MessageCreate events, allowing only non-bot authors.
func filterMsgCreateNotBot(evt interface{}) interface{} {
	evtMsgCreate, ok := evt.(*disgord.MessageCreate)
	maybeTypePanic(ok, "*disgord.MessageCreate", evt)
	if evtMsgCreate.Message.Author == nil || evtMsgCreate.Message.Author.Bot {
		return nil
	}
	return evt
}

// Returns a filter for MessageCreate events, which allows messages with the given prefix only.
func filterMsgCreatePrefix(prefix string) func(evt interface{}) interface{} {
	return func(evt interface{}) interface{} {
		evtMsgCreate, ok := evt.(*disgord.MessageCreate)
		maybeTypePanic(ok, "*disgord.MessageCreate", evt)
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
		evtReactionAdd, ok := evt.(*disgord.MessageReactionAdd)
		maybeTypePanic(ok, "*disgord.MessageReactionAdd", evt)
		if evtReactionAdd.MessageID != msgID {
			return nil
		}
		return evt
	}
}

func maybeTypePanic(ok bool, expected string, got interface{}) {
	if !ok {
		panic(fmt.Errorf("Expected %v, got %T", expected, got))
	}
}
