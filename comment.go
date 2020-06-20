package main

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/andersfylling/disgord"
	"github.com/andybalholm/cascadia"
)

var (
	commentClient     = http.Client{Timeout: 10 * time.Second}
	cfBlogURLRe       = regexp.MustCompile(`https?://codeforces.com/blog/entry/\d+\??(?:locale=ru)?#comment-(\d+)`)
	titleSelec        = cascadia.MustCompile("title")
	moscowTZ          = time.FixedZone("Europe/Moscow", int(3*time.Hour/time.Second))
	timeSelec         = cascadia.MustCompile(".info .format-humantime")
	ratingSelec       = cascadia.MustCompile(".info .commentRating")
	authorHandleSelec = cascadia.MustCompile(".avatar a.rated-user")
	authorPicSelec    = cascadia.MustCompile(".avatar img")
	contentSelec      = cascadia.MustCompile(".ttypography")
	spoilerContentCls = "spoiler-content"
	spoilerSelec      = cascadia.MustCompile("." + spoilerContentCls)
)

const timedMsgTTL = 20 * time.Second

// From https://sta.codeforces.com/s/50332/css/community.css
var colorClsMap = map[string]int{
	"user-black":     0x000000,
	"user-legendary": 0x000000,
	"user-red":       0xFF0000,
	"user-fire":      0xFF0000,
	"user-yellow":    0xBBBB00,
	"user-violet":    0xAA00AA,
	"user-orange":    0xFF8C00,
	"user-blue":      0x0000FF,
	"user-cyan":      0x03A89E,
	"user-green":     0x008000,
	"user-gray":      0x808080,
	"user-admin":     0x000000,
}

type commentFetchErr struct {
	URL *url.URL
	Err error
}

func (err *commentFetchErr) Error() string {
	return fmt.Errorf("Error fetching from %v: %w", err.URL, err.Err).Error()
}

type commentNotFoundErr struct {
	URL       *url.URL
	CommentID string
}

func (err *commentNotFoundErr) Error() string {
	return fmt.Sprintf("No comment with ID %v found on %v", err.CommentID, err.URL)
}

// InstallCfCommentFeature installs the comment watcher feature. The bot watches for Codeforces
// comment links and responds with an embed containing the comment.
func InstallCfCommentFeature(bot *Bot) {
	bot.Client.Logger().Info("Setting up CF comment feature")
	bot.OnMessageCreate(maybeHandleCommentURL)
}

func maybeHandleCommentURL(ctx BotContext, evt *disgord.MessageCreate) {
	match := cfBlogURLRe.FindStringSubmatch(evt.Message.Content)
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
func handleCommentURL(ctx BotContext, commentURL, commentID string) {
	ctx.Logger.Info("Processing comment URL: ", commentURL)

	embed, err := getEmbed(commentURL, commentID)
	if err != nil {
		switch err.(type) {
		case *commentFetchErr, *commentNotFoundErr:
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
	delMsg := func(s disgord.Session, evt *disgord.MessageReactionAdd) {
		if evt.UserID != ctx.Message.Author.ID {
			return
		}
		// TODO: This is hacky, improve. Shouldn't use old ctx and shouldn't repeat logic.
		ctx2 := BotContext{
			Ctx:     evt.Ctx,
			Session: s,
		}
		ctx2.DeleteMsg(msg)
		// This too will fail without manage messages permission, ignore.
		ctx2.UnsuppressEmbeds(ctx.Message)
	}
	AddButtons(ctx, msg, &ReactHandlerMap{"🗑": delMsg}, time.Minute)
}

func getEmbed(commentURL, commentID string) (*disgord.Embed, error) {
	parsedURL, err := url.Parse(commentURL)
	if err != nil {
		return nil, fmt.Errorf("Error parsing URL %q: %w", commentURL, err)
	}
	parsedURL.Fragment = ""
	parsedURL.ForceQuery = false
	resp, err := commentClient.Get(parsedURL.String())
	if err != nil {
		return nil, &commentFetchErr{URL: parsedURL, Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, &commentFetchErr{URL: parsedURL, Err: fmt.Errorf("%v", resp.Status)}
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("Error parsing HTML from %q: %w", parsedURL, err)
	}

	com := doc.Find(fmt.Sprintf(`[commentId="%v"]`, commentID))
	if com.Length() == 0 {
		return nil, &commentNotFoundErr{URL: parsedURL, CommentID: commentID}
	}

	embed, err := makeEmbed(commentURL, doc, com)
	if err != nil {
		return nil, fmt.Errorf("Error building embed for %q: %w", parsedURL, err)
	}

	return embed, nil
}

func makeEmbed(
	commentURL string,
	doc *goquery.Document,
	comment *goquery.Selection,
) (*disgord.Embed, error) {
	title := getTitle(doc)
	authorHandle := getAuthorHandle(comment)
	authorPic := getAuthorPic(comment)
	commentContent, imgs := getCommentContent(comment)
	commentCreationTime, err := getCreationTime(comment)
	if err != nil {
		return nil, err
	}
	commentRating := getRating(comment)
	color := getAuthorColor(comment)

	embed := &disgord.Embed{
		Title:       title,
		URL:         commentURL,
		Description: commentContent,
		Thumbnail: &disgord.EmbedThumbnail{
			URL: authorPic,
		},
		Author: &disgord.EmbedAuthor{
			Name: "Comment by " + authorHandle,
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

func getTitle(doc *goquery.Document) string {
	return doc.FindMatcher(titleSelec).Text()
}

func getCreationTime(comment *goquery.Selection) (t time.Time, err error) {
	comTime := comment.FindMatcher(timeSelec).AttrOr("title", "missing-time-unexpected")
	if t, err = time.ParseInLocation("Jan/2/2006 15:04", comTime, moscowTZ); err != nil {
		// Russian locale has different format, don't ask me why.
		if t, err = time.ParseInLocation("2.1.2006 15:04", comTime, moscowTZ); err != nil {
			return
		}
	}
	t = t.UTC()
	return
}

func getRating(comment *goquery.Selection) string {
	return comment.FindMatcher(ratingSelec).Text()
}

func getAuthorHandle(comment *goquery.Selection) string {
	return comment.FindMatcher(authorHandleSelec).Text()
}

func getAuthorColor(comment *goquery.Selection) int {
	clss := strings.Fields(
		comment.FindMatcher(authorHandleSelec).AttrOr("class", "missing-user-color-unexpected"))
	for _, cls := range clss {
		if col, ok := colorClsMap[cls]; ok {
			return col
		}
	}
	return 0x000000
}

func getAuthorPic(comment *goquery.Selection) string {
	return "https:" + comment.FindMatcher(authorPicSelec).AttrOr("src", "missing-src-unexpected")
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
				src = setCodeforcesHost(src)
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
				text := fmt.Sprintf("[%v](%v)", content, setCodeforcesHost(href))
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

func setCodeforcesHost(URL string) string {
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return ""
	}
	if parsedURL.Host == "" {
		parsedURL.Scheme = "https"
		parsedURL.Host = "codeforces.com"
	}
	return parsedURL.String()
}
