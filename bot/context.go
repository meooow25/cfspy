package bot

import (
	"context"
	"time"

	"github.com/andersfylling/disgord"
)

// Context is passed to all command handlers and contains fields relevant to the current command
// invocation.
type Context struct {
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

// SuppressEmbeds suppresses all embeds on the given message.
func (ctx *Context) SuppressEmbeds(msg *disgord.Message) (*disgord.Message, error) {
	return ctx.Session.UpdateMessage(
		ctx.Ctx,
		msg.ChannelID,
		msg.ID,
	).Set(
		"flags",
		msg.Flags|disgord.MessageFlagSupressEmbeds,
	).Execute()
}

// UnsuppressEmbeds unsuppresses all embeds on the given message.
func (ctx *Context) UnsuppressEmbeds(msg *disgord.Message) (*disgord.Message, error) {
	return ctx.Session.UpdateMessage(
		ctx.Ctx,
		msg.ChannelID,
		msg.ID,
	).Set(
		"flags",
		msg.Flags&^disgord.MessageFlagSupressEmbeds,
	).Execute()
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
