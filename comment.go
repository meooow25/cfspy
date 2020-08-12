package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
	"github.com/PuerkitoBio/goquery"
	"github.com/andersfylling/disgord"
	"github.com/andybalholm/cascadia"
	"github.com/meooow25/cfspy/bot"
)

var (
	commentRatingSelec = cascadia.MustCompile(".commentRating")
	contentSelec       = cascadia.MustCompile(".ttypography")
	spoilerContentCls  = "spoiler-content"
	spoilerSelec       = cascadia.MustCompile("." + spoilerContentCls)
	revisionCountAttr  = "revisioncount"
	revisionSpanSelec  = cascadia.MustCompile("span[" + revisionCountAttr + "]")
	csrfTokenSelec     = cascadia.MustCompile(`meta[name="X-Csrf-Token"]`)
)

func init() {
	rand.Seed(time.Now().Unix())
}

const timedMsgTTL = 20 * time.Second

type commentNotFoundErr struct {
	URL       string
	CommentID string
}

func (err *commentNotFoundErr) Error() string {
	return fmt.Sprintf("No comment with ID %v found on <%v>", err.CommentID, err.URL)
}

// Installs the comment watcher feature. The bot watches for Codeforces comment links and responds
// with an embed containing the comment.
func installCfCommentFeature(bot *bot.Bot) {
	bot.Client.Logger().Info("Setting up CF comment feature")
	bot.OnMessageCreate(maybeHandleCommentURL)
}

func maybeHandleCommentURL(ctx bot.Context, evt *disgord.MessageCreate) {
	if evt.Message.Author.Bot {
		return
	}
	commentURL, commentID := tryParseCFBlogURL(evt.Message.Content)
	if commentURL != "" && commentID != "" {
		go handleCommentURL(ctx, commentURL, commentID)
	}
}

// Fetches the comment from the blog page, converts it to markdown and responds on the Discord
// channel. Scrapes instead of using the API because
// - It's just easier, the comment and author details are together.
// - Some comments in Russian locale seems to be missing from the API.
// - Scraping allows access to different revisions.
func handleCommentURL(ctx bot.Context, commentURL, commentID string) {
	ctx.Logger.Info("Processing comment URL: ", commentURL)

	embedGetter, revisionCnt, err := getCommentEmbedGetter(commentURL, commentID)
	if err != nil {
		switch err.(type) {
		case *scrapeFetchErr, *commentNotFoundErr:
			ctx.SendErrorMsg(err.Error(), timedMsgTTL)
		default:
			ctx.SendInternalErrorMsg(timedMsgTTL)
		}
		ctx.Logger.Error(fmt.Errorf("Comment error: %w", err))
		return
	}

	// Allow the author to delete the preview or choose the revision.
	_, err = ctx.SendPaginated(bot.PaginateParams{
		GetPage:         embedGetter,
		NumPages:        revisionCnt,
		PageToShowFirst: revisionCnt,
		DeactivateAfter: time.Minute,
		DelBtn:          true,
		DelCallback: func(evt *disgord.MessageReactionAdd) {
			// This will fail without manage messages permission, that's fine.
			bot.UnsuppressEmbeds(evt.Ctx, ctx.Session, ctx.Message)
		},
		AllowOp: func(evt *disgord.MessageReactionAdd) bool {
			return evt.UserID == ctx.Message.Author.ID
		},
	})
	if err != nil {
		ctx.Logger.Error(fmt.Errorf("Error sending comment preview: %w", err))
		return
	}

	// This will fail without manage messages permission, that's fine.
	bot.SuppressEmbeds(ctx.Ctx, ctx.Session, ctx.Message)
}

func getCommentEmbedGetter(
	commentURL,
	commentID string,
) (pageGetter bot.PageGetter, revisionCnt int, err error) {
	doc, err := scraperGetDocBrowser(commentURL)
	if err != nil {
		return
	}
	comment := doc.Find(fmt.Sprintf(`[commentId="%v"]`, commentID))
	if comment.Length() == 0 {
		err = &commentNotFoundErr{URL: commentURL, CommentID: commentID}
		return
	}

	embed, err := makeCommentEmbed(commentURL, doc, comment)
	if err != nil {
		err = fmt.Errorf("Error building embed for %q: %w", commentURL, err)
		return
	}

	csrf := doc.FindMatcher(csrfTokenSelec).AttrOr("content", "?!")
	revisionCnt = parseCommentRevision(comment)
	commentCache := map[int]*goquery.Selection{revisionCnt: comment}

	getContent := func(revision int) (string, []string) {
		if comment, ok := commentCache[revision]; ok {
			return getCommentContent(comment)
		}
		resp, err := fetchCommentBrowser(commentID, revision, csrf)
		if err != nil {
			return "An error occured :(", nil
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp))
		if err != nil {
			return "An error occured :(", nil
		}
		commentCache[revision] = doc.Selection
		return getCommentContent(doc.Selection)
	}

	pageGetter = func(revision int) (string, *disgord.Embed) {
		commentContent, imgs := getContent(revision)
		embedCopy := embed.DeepCopy().(*disgord.Embed)
		embedCopy.Description = commentContent
		updateEmbedIfCommentTooLong(embedCopy)
		if len(imgs) > 0 {
			embedCopy.Image = &disgord.EmbedImage{URL: imgs[0]}
		}
		revisionStr := ""
		if revisionCnt > 1 {
			revisionStr = fmt.Sprintf("  •  Revision %v/%v", revision, revisionCnt)
		}
		embedCopy.Footer.Text += revisionStr
		return "", embedCopy
	}
	return
}

func updateEmbedIfCommentTooLong(embed *disgord.Embed) {
	if bot.EmbedDescriptionTooLong(embed) {
		if rand.Intn(20) == 0 {
			embed.Description = "I have discovered this truly marvelous comment, which this " +
				"embed is too narrow to contain."
		} else {
			embed.Description = "The comment is too large to display."
		}
	}
}

// Makes the content embed but leaves out the content and revision number because these vary with
// revisions.
func makeCommentEmbed(
	commentURL string,
	doc *goquery.Document,
	comment *goquery.Selection,
) (*disgord.Embed, error) {
	title := parseTitle(doc)
	avatarDiv := comment.FindMatcher(commentAvatarSelec)
	authorHandle := parseHandle(avatarDiv)
	authorPic := parseImg(avatarDiv)
	commentCreationTime, err := parseTime(comment)
	if err != nil {
		return nil, err
	}
	commentRating := comment.FindMatcher(commentRatingSelec).Text()
	color := parseHandleColor(avatarDiv)

	embed := &disgord.Embed{
		Title: title,
		URL:   commentURL,
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
	return embed, nil
}

func parseCommentRevision(comment *goquery.Selection) int {
	revisionSpan := comment.FindMatcher(revisionSpanSelec)
	if revisionSpan.Length() > 0 {
		if cnt, err := strconv.Atoi(revisionSpan.AttrOr(revisionCountAttr, "?!")); err == nil {
			return cnt
		}
	}
	return 1
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
	converter.Use(plugin.Strikethrough("~~"))

	// TODO: html-to-markdown removes pure whitespace text, which causes adjacent inline elements
	// (such as links) to stick together. Maybe file an issue.
	// https://github.com/JohannesKaufmann/html-to-markdown/blob/master/commonmark.go
	markdown := converter.Convert(comment.FindMatcher(contentSelec))

	// Replace LaTeX delimiter $$$ with $ because it looks ugly.
	// TODO: Maybe Unicode symbols can be used in simple cases.
	markdown = strings.ReplaceAll(markdown, "$$$", "$")
	return markdown, imgURLs
}
