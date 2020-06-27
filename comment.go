package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/andersfylling/disgord"
	"github.com/andybalholm/cascadia"
	"github.com/meooow25/cfspy/bot"
)

var (
	cfCommentURLRe     = regexp.MustCompile(`https?://codeforces.com/blog/entry/\d+\??(?:locale=ru)?#comment-(\d+)`)
	commentRatingSelec = cascadia.MustCompile(".commentRating")
	commentAvatarSelec = cascadia.MustCompile(".avatar")
	imgSelec           = cascadia.MustCompile("img")
	contentSelec       = cascadia.MustCompile(".ttypography")
	spoilerContentCls  = "spoiler-content"
	spoilerSelec       = cascadia.MustCompile("." + spoilerContentCls)
)

const timedMsgTTL = 20 * time.Second

type commentNotFoundErr struct {
	URL       string
	CommentID string
}

func (err *commentNotFoundErr) Error() string {
	return fmt.Sprintf("No comment with ID %v found on %v", err.CommentID, err.URL)
}

// Installs the comment watcher feature. The bot watches for Codeforces comment links and responds
// with an embed containing the comment.
func installCfCommentFeature(bot *bot.Bot) {
	bot.Client.Logger().Info("Setting up CF comment feature")
	bot.OnMessageCreate(maybeHandleCommentURL)
}

func maybeHandleCommentURL(ctx bot.Context, evt *disgord.MessageCreate) {
	match := cfCommentURLRe.FindStringSubmatch(evt.Message.Content)
	if len(match) == 0 {
		return
	}
	go handleCommentURL(ctx, match[0], match[1])
}

// Fetches the comment from the blog page, converts it to markdown and responds on the Discord
// channel. Scrapes instead of using the API because
// - It's just easier, the comment and author details are together.
// - Some comments in Russian locale seems to be missing from the API.
// - Scraping can allow access to different revisions (not yet supported).
func handleCommentURL(ctx bot.Context, commentURL, commentID string) {
	ctx.Logger.Info("Processing comment URL: ", commentURL)

	embed, err := getCommentEmbed(commentURL, commentID)
	if err != nil {
		switch err.(type) {
		case *scrapeFetchErr, *commentNotFoundErr:
			ctx.SendTimed(timedMsgTTL, err.Error())
		default:
			ctx.SendTimed(timedMsgTTL, "Internal error :(")
		}
		ctx.Logger.Info(fmt.Errorf("Comment error: %w", err))
		return
	}

	msg, err := ctx.Send(embed)
	if restErr, ok := err.(*disgord.ErrRest); ok {
		if strings.Contains(restErr.Suggestion, "Embed size exceeds maximum size") {
			ctx.SendTimed(timedMsgTTL, "Comment too long :(")
		}
	}
	if err != nil {
		ctx.Logger.Error(fmt.Errorf("Error sending comment preview: %w", err))
		return
	}
	// This will fail without manage messages permission, that's fine.
	ctx.SuppressEmbeds(ctx.Message)

	// Allow the author to delete the preview.
	handlers := bot.ReactHandlerMap{"🗑": getWidgetDeleteHandler(msg, ctx.Message)}
	bot.AddButtons(ctx, msg, &handlers, time.Minute)
}

func getCommentEmbed(commentURL, commentID string) (*disgord.Embed, error) {
	doc, err := scraperGetDoc(commentURL)
	if err != nil {
		return nil, err
	}
	com := doc.Find(fmt.Sprintf(`[commentId="%v"]`, commentID))
	if com.Length() == 0 {
		return nil, &commentNotFoundErr{URL: commentURL, CommentID: commentID}
	}
	embed, err := makeCommentEmbed(commentURL, doc, com)
	if err != nil {
		return nil, fmt.Errorf("Error building embed for %q: %w", commentURL, err)
	}
	return embed, nil
}

func makeCommentEmbed(
	commentURL string,
	doc *goquery.Document,
	comment *goquery.Selection,
) (*disgord.Embed, error) {
	title := parseTitle(doc)
	avatarDiv := comment.FindMatcher(commentAvatarSelec)
	authorHandle := parseHandle(avatarDiv)
	authorPic := withCodeforcesHost(
		comment.FindMatcher(imgSelec).AttrOr("src", "missing-src-unexpected"))
	commentContent, imgs := getCommentContent(comment)
	commentCreationTime, err := parseTime(comment)
	if err != nil {
		return nil, err
	}
	commentRating := comment.FindMatcher(commentRatingSelec).Text()
	color := parseHandleColor(avatarDiv)

	embed := &disgord.Embed{
		Title:       title,
		URL:         commentURL,
		Description: commentContent,
		Author: &disgord.EmbedAuthor{
			Name: "Comment by " + authorHandle,
		},
		Thumbnail: &disgord.EmbedThumbnail{
			URL: authorPic,
		},
		Timestamp: disgord.Time{
			Time: commentCreationTime,
		},
		Footer: &disgord.EmbedFooter{
			Text: "Score " + commentRating,
		},
		Color: color,
	}
	if len(imgs) > 0 {
		embed.Image = &disgord.EmbedImage{
			URL: imgs[0],
		}
	}

	return embed, nil
}

func getCommentContent(comment *goquery.Selection) (string, []string) {
	converter := md.NewConverter("", true, nil)
	var imgURLs []string
	converter.AddRules(
		md.Rule{
			Filter: []string{"img"},
			Replacement: func(content string, selec *goquery.Selection, opt *md.Options) *string {
				alt := strings.TrimSpace(selec.AttrOr("alt", ""))
				if alt != "" {
					alt = "-" + alt
				}
				src, ok := selec.Attr("src")
				if !ok {
					return new(string)
				}
				src = withCodeforcesHost(src)
				imgURLs = append(imgURLs, src)
				text := fmt.Sprintf("[img%s](%s)", alt, src)
				return &text
			},
		},
		md.Rule{
			Filter: []string{"a"},
			Replacement: func(content string, selec *goquery.Selection, opt *md.Options) *string {
				href, ok := selec.Attr("href")
				if !ok {
					return &content
				}
				text := fmt.Sprintf("[%v](%v)", content, withCodeforcesHost(href))
				return &text
			},
		},
		md.Rule{
			Filter: []string{"div"},
			Replacement: func(content string, selec *goquery.Selection, opt *md.Options) *string {
				toHide := selec.HasClass(spoilerContentCls)
				if parSpoilers := selec.ParentsMatcher(spoilerSelec); parSpoilers.Length() > 0 {
					// Nested spoiler, only hide the top level spoiler.
					toHide = false
				}
				if toHide {
					content = "\n\n||" + strings.TrimSpace(content) + "||\n\n"
				}
				return &content
			},
		},
	)
	// TODO: html-to-markdown removes pure whitespace text, which causes adjacent inline elements
	// (such as links) to stick together. Maybe file an issue.
	// https://github.com/JohannesKaufmann/html-to-markdown/blob/master/commonmark.go
	markdown := converter.Convert(comment.FindMatcher(contentSelec))

	// Replace LaTeX delimiter $$$ with $ because it looks ugly.
	// TODO: Maybe Unicode symbols can be used in simple cases.
	markdown = strings.ReplaceAll(markdown, "$$$", "$")
	return markdown, imgURLs
}
