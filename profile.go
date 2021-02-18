package main

import (
	"context"
	"fmt"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
	"github.com/meooow25/cfspy/fetch"
)

// Installs the profile watcher feature. The bot watches for Codeforces profile links and responds
// with an embed containing some profile info.
func installProfileFeature(bot *bot.Bot) {
	bot.Client.Logger().Info("Setting up CF profile feature")
	bot.OnMessageCreate(maybeHandleProfileURL)
}

func maybeHandleProfileURL(ctx bot.Context, evt *disgord.MessageCreate) {
	go func() {
		profileURLMatches := fetch.ParseProfileURLs(evt.Message.Content)
		if len(profileURLMatches) == 0 {
			return
		}
		first := profileURLMatches[0]
		handleProfileUrl(ctx, first.URL)
	}()
}

// Responds on the Discord channel with some user profile information.
func handleProfileUrl(ctx bot.Context, url string) {
	ctx.Logger.Info("Processing profile URL: ", url)

	profileInfo, err := fetch.Profile(context.Background(), url)
	if err != nil {
		err = fmt.Errorf("Error fetching profile from %v: %w", url, err)
		ctx.Logger.Error(err)
		ctx.SendTimed(timedErrorMsgTTL, ctx.MakeErrorEmbed(err.Error()))
		return
	}

	err = ctx.SendWithDelBtn(bot.OnePageWithDelParams{
		Embed: makeProfileEmbed(profileInfo),
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
	if err != nil {
		ctx.Logger.Error(fmt.Errorf("Error sending profile info: %w", err))
	}
}

func makeProfileEmbed(p *fetch.ProfileInfo) *disgord.Embed {
	desc := p.Rank
	if p.Rank != "Unrated" && p.Rank != "Headquarters" {
		desc += fmt.Sprintf("\nRating: %v (max. %v)", p.Rating, p.MaxRating)
	}
	return &disgord.Embed{
		Title:       p.Handle,
		URL:         p.URL,
		Thumbnail:   &disgord.EmbedThumbnail{URL: p.Avatar},
		Color:       p.Color,
		Description: desc,
	}
}
