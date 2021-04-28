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
	msgCallback, delCallback, allowOp := prepareCallbacks(ctx)
	return ctx.SendWithDelBtn(&bot.OnePageWithDelParams{
		Page:            page,
		MsgCallback:     msgCallback,
		DeactivateAfter: time.Minute,
		DelCallback:     delCallback,
		AllowOp:         allowOp,
	})
}

func respondWithMultiPagePreview(
	ctx *bot.Context,
	getPage bot.PageGetter,
	numPages int,
) error {
	msgCallback, delCallback, allowOp := prepareCallbacks(ctx)
	return ctx.SendPaginated(&bot.PaginateParams{
		GetPage:         getPage,
		NumPages:        numPages,
		PageToShowFirst: numPages,
		MsgCallback:     msgCallback,
		DeactivateAfter: time.Minute,
		DelCallback:     delCallback,
		AllowOp:         allowOp,
	})
}
