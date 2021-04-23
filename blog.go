package main

import (
	"context"
	"fmt"

	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
	"github.com/meooow25/cfspy/fetch"
)

// Length limits for short preview of blog/comments. Much lower than Discord message limits.
const (
	msgLimit = 200
	msgSlack = 100
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

	short, full := makeBlogEmbeds(blogInfo)
	var page *bot.Page
	if full != nil {
		page = bot.NewPageWithExpansion("", short, "", full)
	} else {
		page = bot.NewPage("", short)
	}
	if err = respondWithOnePagePreview(ctx, page); err != nil {
		ctx.Logger.Error(fmt.Errorf("Error sending blog info: %w", err))
	}
}

func makeBlogEmbeds(b *fetch.BlogInfo) (short *disgord.Embed, full *disgord.Embed) {
	embed := &disgord.Embed{
		Title: b.Title,
		URL:   b.URL,
		Author: &disgord.EmbedAuthor{
			Name: b.AuthorHandle + "'s blog",
		},
		Description: b.Content,
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
	if len(b.Images) > 0 {
		embed.Image = &disgord.EmbedImage{URL: b.Images[0]}
	}
	return makeShortAndFullEmbeds(embed)
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
	if len(c.Images) > 0 {
		embed.Image = &disgord.EmbedImage{URL: c.Images[0]}
	}
	return makeShortAndFullEmbeds(embed)
}

func makeShortAndFullEmbeds(embed *disgord.Embed) (short *disgord.Embed, full *disgord.Embed) {
	full = embed
	short = disgord.DeepCopy(full).(*disgord.Embed)
	short.Description = truncate(full.Description)

	// If the content is short enough, no need for full.
	// If the content is too long, full cannot be shown.
	if full.Description == short.Description || bot.EmbedDescriptionTooLong(full) {
		full = nil
	}
	return
}

// Returns the string unchanged if the length is within msgLimit+msgSlack, otherwise returns it
// truncated to msgLimit chars. The motivation for the slack is that the poster would probably want
// to display the full comment anyway if it is a bit over the limit.
func truncate(s string) string {
	if len(s) <= msgLimit+msgSlack {
		return s
	}
	// Cutting off everything beyond limit doesn't care about markdown formatting and can leave
	// unclosed markup.
	// TODO: Maybe use a markdown parser to properly handle these.
	return s[:msgLimit] + "…"
}
