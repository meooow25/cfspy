package main

import (
	"time"

	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
)

func respondWithError(ctx *bot.Context, err error) {
	ctx.SendTimed(30*time.Second, ctx.MakeErrorEmbed(err.Error()))
}

func prepareCallbacks(ctx *bot.Context) (
	msgCallback func(*disgord.Message),
	delCallback func(*disgord.MessageReactionAdd),
	allowOp func(*disgord.MessageReactionAdd) bool,
) {
	return func(*disgord.Message) {
			// This will fail without manage messages permission, that's fine.
			go bot.SuppressEmbeds(ctx.Session, ctx.Message)
		},
		func(*disgord.MessageReactionAdd) {
			// This will fail without manage messages permission, that's fine.
			go bot.UnsuppressEmbeds(ctx.Session, ctx.Message)
		},
		func(evt *disgord.MessageReactionAdd) bool {
			// Allow only the author to control the widget.
			return evt.UserID == ctx.Message.Author.ID
		}
}

func respondWithOnePagePreview(
	ctx *bot.Context,
	page *bot.Page,
) error {
	getPage := func(int) *bot.Page { return page }
	return respondWithMultiPagePreview(ctx, getPage, 1)
}

func respondWithMultiPagePreview(
	ctx *bot.Context,
	getPage func(int) *bot.Page,
	numPages int,
) error {
	msgCallback, delCallback, allowOp := prepareCallbacks(ctx)
	return ctx.SendWidget(&bot.WidgetParams{
		Pages: &bot.Pages{
			Get:   getPage,
			Total: numPages,
			First: numPages,
		},
		MsgCallback: msgCallback,
		Lifetime:    time.Minute,
		DelCallback: delCallback,
		AllowOp:     allowOp,
	})
}
