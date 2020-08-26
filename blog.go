package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
	"github.com/meooow25/cfspy/fetch"
)

var random = rand.New(rand.NewSource(time.Now().Unix()))

// Installs the blog watcher feature. The bot watches for Codeforces blog and comment links and
// responds with an embed containing info about the blog or comment.
func installBlogAndCommentFeature(bot *bot.Bot) {
	bot.Client.Logger().Info("Setting up CF blog and comment feature")
	bot.OnMessageCreate(maybeHandleBlogURL)
}

func maybeHandleBlogURL(ctx bot.Context, evt *disgord.MessageCreate) {
	if evt.Message.Author.Bot {
		return
	}
	go func() {
		blogURLMatches := fetch.ParseBlogURLs(evt.Message.Content)
		if len(blogURLMatches) == 0 {
			return
		}
		first := blogURLMatches[0]
		if first.Suppressed {
			return
		}
		if first.CommentID != "" {
			handleCommentURL(ctx, first.URL, first.CommentID)
		} else {
			handleBlogURL(ctx, first.URL)
		}
	}()
}

// Fetches the blog page and responds on the Discord channel with some basic info on the blog.
// Scrapes instead of using the API because a preview will be added but the blog content is not
// available through the API.
// TODO: Send blog content preview.
func handleBlogURL(ctx bot.Context, blogURL string) {
	ctx.Logger.Info("Processing blog URL: ", blogURL)

	blogInfo, err := fetch.Blog(ctx.Ctx, blogURL)
	if err != nil {
		err = fmt.Errorf("Error fetching blog from %v: %w", blogURL, err)
		ctx.Logger.Error(err)
		ctx.SendErrorMsg(err.Error(), timedErrorMsgTTL)
		return
	}

	// Allow the author to delete the preview.
	_, err = ctx.SendWithDelBtn(bot.OnePageWithDelParams{
		Embed:           makeBlogEmbed(blogInfo),
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

func makeBlogEmbed(b *fetch.BlogInfo) *disgord.Embed {
	return &disgord.Embed{
		Title: b.Title,
		URL:   b.URL,
		Author: &disgord.EmbedAuthor{
			Name: b.AuthorHandle + "'s blog",
		},
		Thumbnail: &disgord.EmbedThumbnail{
			URL: b.AuthorAvatar,
		},
		Timestamp: disgord.Time{
			Time: b.CreationTime,
		},
		Footer: &disgord.EmbedFooter{
			Text: fmt.Sprintf("Score %+d", b.Rating),
		},
		Color: b.AuthorColor,
	}
}

// Fetches the comment from the blog page, converts it to markdown and responds on the Discord
// channel. Scrapes instead of using the API because
// - It's just easier, the comment and author details are together.
// - Some comments in Russian locale seems to be missing from the API.
// - Scraping allows access to different revisions.
func handleCommentURL(ctx bot.Context, commentURL, commentID string) {
	ctx.Logger.Info("Processing comment URL: ", commentURL)

	revisionCount, infoGetter, err := fetch.Comment(ctx.Ctx, commentURL, commentID)
	if err != nil {
		err = fmt.Errorf("Error fetching comment from %v: %w", commentURL, err)
		ctx.Logger.Error(err)
		ctx.SendErrorMsg(err.Error(), timedErrorMsgTTL)
		return
	}

	// Allow the author to delete the preview or choose the revision.
	_, err = ctx.SendPaginated(bot.PaginateParams{
		GetPage: func(revision int) (string, *disgord.Embed) {
			commentInfo, err := infoGetter(revision)
			if err != nil {
				return fmt.Sprintf("Failed to fetch revision %v", revision), nil
			}
			return "", makeCommentEmbed(commentInfo)
		},
		NumPages:        revisionCount,
		PageToShowFirst: revisionCount,
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

func makeCommentEmbed(c *fetch.CommentInfo) *disgord.Embed {
	revisionStr := ""
	if c.RevisionCount > 1 {
		revisionStr = fmt.Sprintf(
			"  •  Revision %v/%v", c.Revision, c.RevisionCount)
	}
	embed := &disgord.Embed{
		Title: c.BlogTitle,
		URL:   c.URL,
		Author: &disgord.EmbedAuthor{
			Name: "Comment by " + c.AuthorHandle,
		},
		Description: c.Content,
		Thumbnail: &disgord.EmbedThumbnail{
			URL: c.AuthorAvatar,
		},
		Timestamp: disgord.Time{
			Time: c.CreationTime,
		},
		Footer: &disgord.EmbedFooter{
			Text: fmt.Sprintf("Score %+d%s", c.Rating, revisionStr),
		},
		Color: c.AuthorColor,
	}
	updateEmbedIfCommentTooLong(embed)
	return embed
}

func updateEmbedIfCommentTooLong(embed *disgord.Embed) {
	if bot.EmbedDescriptionTooLong(embed) {
		if random.Intn(20) == 0 {
			embed.Description = "I have discovered this truly marvelous comment, which this " +
				"embed is too narrow to contain."
		} else {
			embed.Description = "The comment is too large to display."
		}
	}
}
