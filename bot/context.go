package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/andersfylling/disgord"
)

const ALERT_AMBER = 0xFFBF00

// Context is passed to all command handlers and contains fields relevant to the current command
// invocation.
type Context struct {
	Bot     *Bot
	Session disgord.Session
	Message *disgord.Message
	Command *Command
	Args    []string
	Ctx     context.Context
	Logger  disgord.Logger
}

// Send sends a message in the current channel.
func (ctx *Context) Send(data ...interface{}) (*disgord.Message, error) {
	return ctx.Session.SendMsg(ctx.Ctx, ctx.Message.ChannelID, data...)
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
	return ctx.Session.SetMsgContent(ctx.Ctx, msg.ChannelID, msg.ID, content)
}

// DeleteMsg deletes the given message.
func (ctx *Context) DeleteMsg(msg *disgord.Message) error {
	return ctx.Session.DeleteFromDiscord(ctx.Ctx, msg)
}

// React reacts on the given message with the given emoji.
func (ctx *Context) React(msg *disgord.Message, emoji interface{}) error {
	return msg.React(ctx.Ctx, ctx.Session, emoji)
}

// SendIncorrectUsageMsg sends the incorrect usage message for the current command.
func (ctx *Context) SendIncorrectUsageMsg() (*disgord.Message, error) {
	return ctx.Send(ctx.Command.IncorrectUsageMsg())
}

// SendInternalErrorMsg sends an error message with the bot's support URL if it exists.
func (ctx *Context) SendInternalErrorMsg(deleteAfter time.Duration) (*disgord.Message, error) {
	embed := disgord.Embed{
		Author: &disgord.EmbedAuthor{Name: "Internal error :("},
		Color:  ALERT_AMBER,
	}
	if ctx.Bot.Info.SupportURL != "" {
		desc := "If this issue is reproducible, please report it [here](%s)"
		embed.Description = fmt.Sprintf(desc, ctx.Bot.Info.SupportURL)
	}
	return ctx.SendTimed(deleteAfter, embed)
}

// SendPaginated sends a paginated message in the current channel.
func (ctx *Context) SendPaginated(params PaginateParams) (*disgord.Message, error) {
	return SendPaginated(ctx.Ctx, params, ctx.Session, ctx.Message.ChannelID)
}

// SendWithDelBtn sends a message and adds a delete button to it.
func (ctx *Context) SendWithDelBtn(params OnePageWithDelParams) (*disgord.Message, error) {
	return SendWithDelBtn(ctx.Ctx, params, ctx.Session, ctx.Message.ChannelID)
}
