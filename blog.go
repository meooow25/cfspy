package main

import (
	"context"
	"fmt"

	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
	"github.com/meooow25/cfspy/fetch"
)

// Installs the blog watcher feature. The bot watches for Codeforces blog and comment links and
// responds with an embed containing info about the blog or comment.
func installBlogAndCommentFeature(bot *bot.Bot) {
	bot.Client.Logger().Info("Setting up CF blog and comment feature")
	bot.OnMessageCreate(maybeHandleBlogURL)
}

func maybeHandleBlogURL(ctx *bot.Context, evt *disgord.MessageCreate) {
	go func() {
		blogURLMatches := fetch.ParseBlogURLs(evt.Message.Content)
		if len(blogURLMatches) == 0 {
			return
		}
		first := blogURLMatches[0]
		if first.CommentID != "" {
			handleCommentURL(ctx, first.URL, first.CommentID)
		} else {
			handleBlogURL(ctx, first.URL)
		}
	}()
}

// Fetches the blog page and responds on the Discord channel with some basic info on the blog.
func handleBlogURL(ctx *bot.Context, blogURL string) {
	ctx.Logger.Info("Processing blog URL: ", blogURL)

	blogInfo, err := fetch.Blog(context.Background(), blogURL)
	if err != nil {
		err = fmt.Errorf("Error fetching blog from %v: %w", blogURL, err)
		ctx.Logger.Error(err)
		respondWithError(ctx, err)
		return
	}

	page := bot.NewPage("", makeBlogEmbed(blogInfo))
	if err = respondWithOnePagePreview(ctx, page); err != nil {
		ctx.Logger.Error(fmt.Errorf("Error sending blog info: %w", err))
	}
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
// channel.
func handleCommentURL(ctx *bot.Context, commentURL, commentID string) {
	ctx.Logger.Info("Processing comment URL: ", commentURL)

	revisionCount, infoGetter, err := fetch.Comment(context.Background(), commentURL, commentID)
	if err != nil {
		err = fmt.Errorf("Error fetching comment from %v: %w", commentURL, err)
		ctx.Logger.Error(err)
		respondWithError(ctx, err)
		return
	}

	getPage := func(revision int) *bot.Page {
		commentInfo, err := infoGetter(revision)
		if err != nil {
			err := fmt.Errorf("Error fetching revision %v of comment %v: %w", revision, commentURL, err)
			ctx.Logger.Error(err)
			return bot.NewPage("", ctx.MakeErrorEmbed(err.Error()))
		}
		short, full := makeCommentEmbeds(commentInfo)
		if full != nil {
			return bot.NewPageWithExpansion("", short, "", full)
		}
		return bot.NewPage("", short)
	}

	if err = respondWithMultiPagePreview(ctx, getPage, revisionCount); err != nil {
		ctx.Logger.Error(fmt.Errorf("Error sending comment preview: %w", err))
	}
}

func makeCommentEmbeds(c *fetch.CommentInfo) (short *disgord.Embed, full *disgord.Embed) {
	revisionStr := ""
	if c.RevisionCount > 1 {
		revisionStr = fmt.Sprintf(
			"  •  Revision %v/%v", c.Revision, c.RevisionCount)
	}
	full = &disgord.Embed{
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
	if len(c.Images) > 0 {
		full.Image = &disgord.EmbedImage{URL: c.Images[0]}
	}
	short = disgord.DeepCopy(full).(*disgord.Embed)
	short.Description = truncateMessage(full.Description)

	// If the comment is short enough, no need for full.
	// If the comment is too long, full cannot be shown.
	if full.Description == short.Description || bot.EmbedDescriptionTooLong(full) {
		full = nil
	}
	return
}
