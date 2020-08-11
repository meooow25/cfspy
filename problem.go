package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andersfylling/disgord"
	"github.com/andybalholm/cascadia"
	"github.com/meooow25/cfspy/bot"
)

var (
	cfProblemURLRe        = regexp.MustCompile(`https?://codeforces.com/(?:contest|gym)/\d+/problem/\S+`)
	problemNameSelec      = cascadia.MustCompile(".problem-statement .header .title")
	contestNameSelec      = cascadia.MustCompile("#sidebar a") // Pick first. Couldn't find anything better
	contestStatusSelec    = cascadia.MustCompile(".contest-state-phase")
	contestMaterialsSelec = cascadia.MustCompile("#sidebar li a") // Couldn't find anything better
)

// Installs the problem watcher feature. The bot watches for Codeforces problem links and responds
// with an embed containing info about the problem.
func installCfProblemFeature(bot *bot.Bot) {
	bot.Client.Logger().Info("Setting up CF problem feature")
	bot.OnMessageCreate(maybeHandleProblemURL)
}

func maybeHandleProblemURL(ctx bot.Context, evt *disgord.MessageCreate) {
	if evt.Message.Author.Bot {
		return
	}
	problemURL := cfProblemURLRe.FindString(evt.Message.Content)
	if problemURL == "" {
		return
	}
	if _, err := url.Parse(problemURL); err == nil {
		go handleProblemURL(ctx, problemURL)
	}
}

// Fetches the problem page and responds on the Discord channel with some basic info on the problem.
// Scrapes instead of using the API because announcement and tutorial links are not available on
// the API.
func handleProblemURL(ctx bot.Context, problemURL string) {
	ctx.Logger.Info("Processing problem URL: ", problemURL)

	embed, err := getProblemEmbed(problemURL)
	if err != nil {
		switch err.(type) {
		case *scrapeFetchErr:
			ctx.SendErrorMsg(err.Error(), timedMsgTTL)
		default:
			ctx.SendInternalErrorMsg(timedMsgTTL)
		}
		ctx.Logger.Error(fmt.Errorf("Problem error: %w", err))
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
		ctx.Logger.Error(fmt.Errorf("Error sending problem info: %w", err))
		return
	}

	// This will fail without manage messages permission, that's fine.
	bot.SuppressEmbeds(ctx.Ctx, ctx.Session, ctx.Message)
}

func getProblemEmbed(problemURL string) (embed *disgord.Embed, err error) {
	doc, err := scraperGetDoc(problemURL)
	if err != nil {
		return
	}
	if embed, err = makeProblemEmbed(problemURL, doc); err != nil {
		err = fmt.Errorf("Error building embed for %q: %w", problemURL, err)
	}
	return
}

func makeProblemEmbed(problemURL string, doc *goquery.Document) (*disgord.Embed, error) {
	problemName := doc.FindMatcher(problemNameSelec).Text()
	contestName := doc.FindMatcher(contestNameSelec).First().Text()
	contestStatusSelec := doc.FindMatcher(contestStatusSelec).Text()
	var materials []string
	doc.FindMatcher(contestMaterialsSelec).Each(func(_ int, s *goquery.Selection) {
		url := withCodeforcesHost(s.AttrOr("href", "?!"))
		materials = append(materials, fmt.Sprintf("[%s](%s)", s.Text(), url))
	})

	contestStr := contestName
	if contestStatusSelec != "" {
		contestStr += " [" + contestStatusSelec + "]"
	}
	var fields []*disgord.EmbedField
	if len(materials) > 0 {
		fields = append(fields, &disgord.EmbedField{
			Name:  "Contest materials",
			Value: strings.Join(materials, "\n"),
		})
	}

	embed := &disgord.Embed{
		Title: problemName,
		URL:   problemURL,
		Author: &disgord.EmbedAuthor{
			Name: contestStr,
		},
		Fields: fields,
	}

	return embed, nil
}
