package main

import (
	"context"
	"fmt"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andersfylling/disgord"
	"github.com/andybalholm/cascadia"
	"github.com/meooow25/cfspy/bot"
)

var (
	blogSelec       = cascadia.MustCompile(".topic")
	blogRatingSelec = cascadia.MustCompile(`[title="Topic rating"]`)
)

// Installs the blog watcher feature. The bot watches for Codeforces blog links and responds with an
// embed containing info about the blog.
func installCfBlogFeature(bot *bot.Bot) {
	bot.Client.Logger().Info("Setting up CF blog feature")
	bot.OnMessageCreate(maybeHandleBlogURL)
}

func maybeHandleBlogURL(ctx bot.Context, evt *disgord.MessageCreate) {
	if evt.Message.Author.Bot {
		return
	}
	blogURL, commentId := tryParseCFURL(evt.Message.Content)
	if blogURL != "" && commentId == "" {
		go handleBlogURL(ctx, blogURL)
	}
}

// Fetches the blog page and responds on the Discord channel with some basic info on the blog.
// Scrapes instead of using the API because a preview will be added but the blog content is not
// available through the API.
// TODO: Send blog content preview.
func handleBlogURL(ctx bot.Context, blogURL string) {
	ctx.Logger.Info("Processing blog URL: ", blogURL)

	embed, err := getBlogEmbed(blogURL)
	if err != nil {
		switch err.(type) {
		case *scrapeFetchErr:
			ctx.SendTimed(timedMsgTTL, err.Error())
		default:
			ctx.SendInternalErrorMsg(timedMsgTTL)
		}
		ctx.Logger.Info(fmt.Errorf("Blog error: %w", err))
		return
	}

	// Allow the author to delete the preview.
	_, err = ctx.SendWithDelBtn(bot.OnePageWithDelParams{
		Embed:           embed,
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
		ctx.Logger.Error(fmt.Errorf("Error sending blog info: %w", err))
		return
	}

	// This will fail without manage messages permission, that's fine.
	bot.SuppressEmbeds(ctx.Ctx, ctx.Session, ctx.Message)
}

func getBlogEmbed(blogURL string) (*disgord.Embed, error) {
	doc, err := scraperGetDoc(blogURL)
	if err != nil {
		return nil, err
	}
	embed, err := makeBlogEmbed(blogURL, doc)
	if err != nil {
		return nil, fmt.Errorf("Error building embed for %q: %w", blogURL, err)
	}
	return embed, nil
}

func makeBlogEmbed(blogURL string, doc *goquery.Document) (*disgord.Embed, error) {
	title := parseTitle(doc)
	blogDiv := doc.FindMatcher(blogSelec)
	authorHandle := parseHandle(blogDiv)
	blogCreationTime, err := parseTime(blogDiv)
	if err != nil {
		return nil, err
	}
	blogRating := blogDiv.FindMatcher(blogRatingSelec).Text()
	color := parseHandleColor(blogDiv)

	var authorPic string
	// If the author commented under the blog we get the pic, otherwise fetch from the API.
	if authorCommentAvatars := blogDiv.FindMatcher(commentAvatarSelec).FilterFunction(
		func(_ int, s *goquery.Selection) bool {
			return s.FindMatcher(handleSelec).Text() == authorHandle
		},
	); authorCommentAvatars.Length() > 0 {
		authorPic = parseImg(authorCommentAvatars)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		infos, err := cfAPI.GetUserInfo(ctx, []string{authorHandle})
		if err != nil {
			return nil, err
		}
		authorPic = withCodeforcesHost(infos[0].Avatar)
	}

	embed := &disgord.Embed{
		Title: title,
		URL:   blogURL,
		Author: &disgord.EmbedAuthor{
			Name: authorHandle + "'s blog",
		},
		Thumbnail: &disgord.EmbedThumbnail{
			URL: authorPic,
		},
		Timestamp: disgord.Time{
			Time: blogCreationTime,
		},
		Footer: &disgord.EmbedFooter{
			Text: "Score " + blogRating,
		},
		Color: color,
	}

	return embed, nil
}
