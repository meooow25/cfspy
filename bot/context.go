package bot

import (
	"context"
	"time"

	"github.com/andersfylling/disgord"
)

const alertAmber = 0xFFBF00

// Context is passed to all command handlers and contains fields relevant to the current command
// invocation.
type Context struct {
	Bot     *Bot
	Session disgord.Session
	Message *disgord.Message
	Command *Command
	Args    []string
	Logger  disgord.Logger
}

// Send sends a message in the current channel.
func (ctx *Context) Send(data ...interface{}) (*disgord.Message, error) {
	return ctx.Message.Reply(context.Background(), ctx.Session, data...)
}

// SendTimed sends a message and deletes it after a delay. Ignores any error if the delete fails.
func (ctx *Context) SendTimed(
	deleteAfter time.Duration,
	data ...interface{},
) (*disgord.Message, error) {
	msg, err := ctx.Send(data...)
	if err == nil {
		time.AfterFunc(deleteAfter, func() { ctx.DeleteMsg(msg) })
	}
	return msg, err
}

// EditMsg edits a message to set the given string as content.
func (ctx *Context) EditMsg(msg *disgord.Message, content string) (*disgord.Message, error) {
	return MsgQueryBuilder(ctx.Session, msg).SetContent(content)
}

// DeleteMsg deletes the given message.
func (ctx *Context) DeleteMsg(msg *disgord.Message) error {
	return MsgQueryBuilder(ctx.Session, msg).Delete()
}

// React reacts on the given message with the given emoji.
func (ctx *Context) React(msg *disgord.Message, emoji interface{}) error {
	return msg.React(context.Background(), ctx.Session, emoji)
}

// SendIncorrectUsageMsg sends the incorrect usage message for the current command.
func (ctx *Context) SendIncorrectUsageMsg() (*disgord.Message, error) {
	return ctx.Send(ctx.Command.IncorrectUsageMsg())
}

// MakeErrorEmbed prepares an error embed with the bot's support URL if it exists.
func (ctx *Context) MakeErrorEmbed(msg string) *disgord.Embed {
	embed := disgord.Embed{
		Color:       alertAmber,
		Description: msg,
	}
	if ctx.Bot.Info.SupportURL != "" {
		embed.Description += "\n_If this is a reproducible bug, please [report it](" +
			ctx.Bot.Info.SupportURL + ")._"
	}
	return &embed
}

// SendPaginated sends a paginated message in the current channel.
func (ctx *Context) SendPaginated(params *PaginateParams) error {
	return SendPaginated(context.Background(), params, ctx.Session, ctx.Message.ChannelID)
}

// SendWithDelBtn sends a message and adds a delete button to it.
func (ctx *Context) SendWithDelBtn(params *OnePageWithDelParams) error {
	return SendWithDelBtn(context.Background(), params, ctx.Session, ctx.Message.ChannelID)
}
