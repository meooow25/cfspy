package main

import (
	"fmt"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
	"github.com/meooow25/cfspy/fetch"
)

// Installs the problem watcher feature. The bot watches for Codeforces problem links and responds
// with an embed containing info about the problem.
func installProblemFeature(bot *bot.Bot) {
	bot.Client.Logger().Info("Setting up CF problem feature")
	bot.OnMessageCreate(maybeHandleProblemURL)
}

func maybeHandleProblemURL(ctx bot.Context, evt *disgord.MessageCreate) {
	if evt.Message.Author.Bot {
		return
	}
	go func() {
		problemURLMatches := fetch.ParseProblemURLs(evt.Message.Content)
		if len(problemURLMatches) == 0 {
			return
		}
		first := problemURLMatches[0]
		if first.Suppressed {
			return
		}
		handleProblemURL(ctx, first.URL)
	}()
}

// Fetches the problem page and responds on the Discord channel with some basic info on the problem.
// TODO: Maybe show a preview of the statement like DMOJ.
func handleProblemURL(ctx bot.Context, problemURL string) {
	ctx.Logger.Info("Processing problem URL: ", problemURL)

	problemInfo, err := fetch.Problem(ctx.Ctx, problemURL)
	if err != nil {
		err = fmt.Errorf("Error fetching problem from %v: %w", problemURL, err)
		ctx.Logger.Error(err)
		ctx.SendErrorMsg(err.Error(), timedErrorMsgTTL)
		return
	}

	// Allow the author to delete the preview.
	_, err = ctx.SendWithDelBtn(bot.OnePageWithDelParams{
		Embed:           makeProblemEmbed(problemInfo),
		DeactivateAfter: time.Minute,
		DelCallback: func(evt *disgord.MessageReactionAdd) {
			// This will fail without manage messages permission, that's fine.
			bot.UnsuppressEmbeds(evt.Ctx, ctx.Session, ctx.Message)
		},
		AllowOp: func(evt *disgord.MessageReactionAdd) bool {
			return evt.UserID == ctx.Message.Author.ID
		},
	})
	if err != nil {
		ctx.Logger.Error(fmt.Errorf("Error sending problem info: %w", err))
		return
	}

	// This will fail without manage messages permission, that's fine.
	bot.SuppressEmbeds(ctx.Ctx, ctx.Session, ctx.Message)
}

func makeProblemEmbed(p *fetch.ProblemInfo) *disgord.Embed {
	return &disgord.Embed{
		Title: p.Name,
		URL:   p.URL,
		Author: &disgord.EmbedAuthor{
			Name: fmt.Sprintf("%s [%s]", p.ContestName, p.ContestStatus),
		},
	}
}
