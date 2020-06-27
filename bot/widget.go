package bot

import (
	"time"

	"github.com/andersfylling/disgord"
)

// ReactHandler is an alias for a map of MessageReactionAdd handlers.
type ReactHandler = func(disgord.Session, *disgord.MessageReactionAdd)

// ReactHandlerMap is an alias for a map of emoji strings to handlers. Does not support custom
// emojis.
type ReactHandlerMap = map[string]ReactHandler

// AddButtons adds reacts to the given message and binds the given handlers so that when a user
// reacts the appropriate handler is called.
func AddButtons(
	ctx Context,
	widget *disgord.Message,
	buttons *ReactHandlerMap,
	deactivateAfter time.Duration,
) {
	for emoji, handler := range *buttons {
		widget.React(ctx.Ctx, ctx.Session, emoji)
		ctx.Session.On(
			disgord.EvtMessageReactionAdd,
			reactFilter(emoji, widget.ID),
			handler,
			&disgord.Ctrl{Duration: deactivateAfter})
	}
	time.AfterFunc(deactivateAfter, func() {
		// Fails without manage messages, ignore.
		ctx.Session.DeleteAllReactions(ctx.Ctx, widget.ChannelID, widget.ID)
	})
}

func reactFilter(emoji string, msgID disgord.Snowflake) func(interface{}) interface{} {
	return func(evt interface{}) interface{} {
		if evt, ok := evt.(*disgord.MessageReactionAdd); ok {
			if evt.MessageID == msgID && evt.PartialEmoji.Name == emoji {
				return evt
			}
		}
		return nil
	}
}
