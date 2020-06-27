package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andersfylling/disgord"
	"github.com/andybalholm/cascadia"
	"github.com/meooow25/cfspy/bot"
)

var (
	blogClient      = http.Client{Timeout: 10 * time.Second}
	cfBlogURLRe     = regexp.MustCompile(`https?://codeforces.com/blog/entry/(\d+)\??(?:locale=ru)?(?:$|\s)`)
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
	match := cfBlogURLRe.FindStringSubmatch(evt.Message.Content)
	if len(match) == 0 {
		return
	}
	go handleBlogURL(ctx, match[0], match[1])
}

// TODO: Send blog content preview.
func handleBlogURL(ctx bot.Context, blogURL, blogID string) {
	ctx.Logger.Info("Processing blog URL: ", blogURL)

	embed, err := getBlogEmbed(blogURL, blogID)
	if err != nil {
		switch err.(type) {
		case *scrapeFetchErr:
			ctx.SendTimed(timedMsgTTL, err.Error())
		default:
			ctx.SendTimed(timedMsgTTL, "Internal error :(")
		}
		ctx.Logger.Info(fmt.Errorf("Blog error: %w", err))
		return
	}

	msg, err := ctx.Send(embed)
	if err != nil {
		ctx.Logger.Error(fmt.Errorf("Error sending blog info: %w", err))
		return
	}
	// This will fail without manage messages permission, that's fine.
	ctx.SuppressEmbeds(ctx.Message)

	// Allow the author to delete the preview.
	handlers := bot.ReactHandlerMap{"🗑": getWidgetDeleteHandler(msg, ctx.Message)}
	bot.AddButtons(ctx, msg, &handlers, time.Minute)
}

func getBlogEmbed(blogURL, blogID string) (*disgord.Embed, error) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	infos, err := cfAPI.GetUserInfo(ctx, []string{authorHandle})
	if err != nil {
		return nil, err
	}
	authorPic := withCodeforcesHost(infos[0].Avatar)

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
