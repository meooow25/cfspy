package main

import (
	"time"

	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
)

func respondWithError(ctx bot.Context, err error) {
	ctx.SendTimed(30*time.Second, ctx.MakeErrorEmbed(err.Error()))
}

func respondWithOnePagePreview(
	ctx bot.Context,
	content string,
	embed *disgord.Embed,
) error {
	return ctx.SendWithDelBtn(bot.OnePageWithDelParams{
		Content: content,
		Embed:   embed,
		MsgCallback: func(*disgord.Message) {
			// This will fail without manage messages permission, that's fine.
			go bot.SuppressEmbeds(ctx.Session, ctx.Message)
		},
		DeactivateAfter: time.Minute,
		DelCallback: func(evt *disgord.MessageReactionAdd) {
			// This will fail without manage messages permission, that's fine.
			go bot.UnsuppressEmbeds(ctx.Session, ctx.Message)
		},
		AllowOp: func(evt *disgord.MessageReactionAdd) bool {
			// Allow only the author to control the widget.
			return evt.UserID == ctx.Message.Author.ID
		},
	})
}

func respondWithMultiPagePreview(
	ctx bot.Context,
	getPage bot.PageGetter,
	numPages int,
) error {
	return ctx.SendPaginated(bot.PaginateParams{
		GetPage:         getPage,
		NumPages:        numPages,
		PageToShowFirst: numPages,
		MsgCallback: func(*disgord.Message) {
			// This will fail without manage messages permission, that's fine.
			go bot.SuppressEmbeds(ctx.Session, ctx.Message)
		},
		DeactivateAfter: time.Minute,
		DelBtn:          true,
		DelCallback: func(evt *disgord.MessageReactionAdd) {
			// This will fail without manage messages permission, that's fine.
			go bot.UnsuppressEmbeds(ctx.Session, ctx.Message)
		},
		AllowOp: func(evt *disgord.MessageReactionAdd) bool {
			// Allow only the author to control the widget.
			return evt.UserID == ctx.Message.Author.ID
		},
	})
}
