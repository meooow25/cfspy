package fetch

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
)

var (
	titleSelec         = cascadia.MustCompile(".title")
	timeSelec          = cascadia.MustCompile(".info .format-humantime")
	moscowTZ           = time.FixedZone("Europe/Moscow", int(3*time.Hour/time.Second))
	commentAvatarSelec = cascadia.MustCompile(".avatar")
	imgSelec           = cascadia.MustCompile("img")
	scriptSelec        = cascadia.MustCompile("script")
	blogSelec          = cascadia.MustCompile(".topic")
	blogRatingSelec    = cascadia.MustCompile(`[title="Topic rating"]`)
	commentRatingSelec = cascadia.MustCompile(".commentRating")
	contentSelec       = cascadia.MustCompile(".ttypography")
	spoilerContentCls  = "spoiler-content"
	spoilerSelec       = cascadia.MustCompile("." + spoilerContentCls)
	revisionCountAttr  = "revisioncount"
	revisionSpanSelec  = cascadia.MustCompile("span[" + revisionCountAttr + "]")
	csrfTokenSelec     = cascadia.MustCompile(`meta[name="X-Csrf-Token"]`)
)

// Blog fetches blog information using the DefaultFetcher.
func Blog(ctx context.Context, url string) (*BlogInfo, error) {
	return DefaultFetcher.Blog(ctx, url)
}

// Blog fetches blog information. The given URL must be a valid blog URL.
//
// Scrapes instead of using the API because a preview will be added but the blog content is not
// available through the API.
// TODO: Use blog content.
func (f *Fetcher) Blog(ctx context.Context, url string) (*BlogInfo, error) {
	doc, err := f.FetchPage(ctx, url)
	if err != nil {
		return nil, err
	}

	var b BlogInfo
	b.URL = url
	b.Title = strings.TrimSpace(doc.FindMatcher(titleSelec).First().Text())
	blogDiv := doc.FindMatcher(blogSelec)
	b.AuthorHandle, b.AuthorColor = parseHandleAndColor(blogDiv)
	if b.CreationTime, err = parseTime(blogDiv); err != nil {
		return nil, err
	}
	if b.Rating, err = strconv.Atoi(blogDiv.FindMatcher(blogRatingSelec).Text()); err != nil {
		return nil, err
	}

	// If the author commented under the blog get the pic, otherwise fetch from the API.
	if authorCommentAvatars := blogDiv.FindMatcher(commentAvatarSelec).FilterFunction(
		func(_ int, s *goquery.Selection) bool {
			return s.FindMatcher(handleSelec).Text() == b.AuthorHandle
		},
	); authorCommentAvatars.Length() > 0 {
		b.AuthorAvatar = parseImg(authorCommentAvatars)
	} else {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if b.AuthorAvatar, err = f.FetchAvatar(ctx, b.AuthorHandle); err != nil {
			return nil, err
		}
	}

	return &b, nil
}

// Comment fetches comment information using the DefaultFetcher.
func Comment(
	ctx context.Context,
	url string,
	commentID string,
) (revisionCount int, getter CommentInfoGetter, err error) {
	return DefaultFetcher.Comment(ctx, url, commentID)
}

// CommentInfoGetter is a function that returns the comment info for a given revision.
type CommentInfoGetter func(revision int) (*CommentInfo, error)

// Comment fetches comment information. The given URL must be a valid comment URL. A
// CommentInfoGetter is returned. The last revision is immediately available, other revisions are
// fetched lazily when the CommentInfoGetter is called.
//
// Scrapes instead of using the API because
// - It's just easier, the comment and author details are together.
// - Some comments in Russian locale seems to be missing from the API.
// - Scraping allows access to different revisions.
func (f *Fetcher) Comment(
	ctx context.Context,
	url string,
	commentID string,
) (revisionCount int, getter CommentInfoGetter, err error) {
	doc, err := f.FetchPageBrowser(ctx, url)
	if err != nil {
		return
	}
	comment := doc.Find(fmt.Sprintf(`[commentId="%v"]`, commentID))
	if comment.Length() == 0 {
		err = fmt.Errorf("No comment with ID %v found", commentID)
		return
	}

	var base CommentInfo
	base.URL = url
	base.BlogTitle = strings.TrimSpace(doc.FindMatcher(titleSelec).First().Text())
	avatarDiv := comment.FindMatcher(commentAvatarSelec)
	if base.CreationTime, err = parseTime(comment); err != nil {
		return
	}
	base.AuthorHandle, base.AuthorColor = parseHandleAndColor(avatarDiv)
	base.AuthorAvatar = parseImg(avatarDiv)
	if base.Rating, err = strconv.Atoi(comment.FindMatcher(commentRatingSelec).Text()); err != nil {
		return
	}

	csrf := doc.FindMatcher(csrfTokenSelec).AttrOr("content", "?!")
	revisionCount = parseCommentRevision(comment)
	base.RevisionCount = revisionCount

	latest := base
	latest.Content, latest.Images = getCommentContent(comment)
	latest.Revision = revisionCount
	cache := map[int]*CommentInfo{revisionCount: &latest}

	getter = func(revision int) (*CommentInfo, error) {
		if revision <= 0 || revision > base.RevisionCount {
			return nil, fmt.Errorf(
				"Expected revision between 1 and %v, got %v", base.RevisionCount, revision)
		}
		if _, ok := cache[revision]; !ok {
			doc, err := f.FetchCommentRevision(ctx, commentID, revision, csrf)
			if err != nil {
				return nil, err
			}
			cur := base
			cur.Revision = revision
			cur.Content, cur.Images = getCommentContent(doc.Selection)
			cache[revision] = &cur
		}
		return cache[revision], nil
	}
	return
}

func parseCommentRevision(comment *goquery.Selection) int {
	cnt, err := strconv.Atoi(comment.FindMatcher(revisionSpanSelec).AttrOr(revisionCountAttr, "?!"))
	if err != nil {
		return 1
	}
	return cnt
}

func getCommentContent(comment *goquery.Selection) (markdown string, imgURLs []string) {
	converter := md.NewConverter("", true, nil)
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
				text = md.AddSpaceIfNessesary(selec, text)
				return &text
			},
		},
		md.Rule{
			Filter: []string{"div"},
			Replacement: func(content string, selec *goquery.Selection, opt *md.Options) *string {
				hide := selec.HasClass(spoilerContentCls)
				if parSpoilers := selec.ParentsMatcher(spoilerSelec); parSpoilers.Length() > 0 {
					// Nested spoiler, only hide the top level spoiler.
					hide = false
				}
				if hide {
					content = "\n\n||" + strings.TrimSpace(content) + "||\n\n"
				}
				return &content
			},
		},
	)
	converter.Use(plugin.Strikethrough("~~"))

	markdown = converter.Convert(comment.FindMatcher(contentSelec))

	// Replace LaTeX delimiter $$$ with $ because it looks ugly.
	// TODO: Maybe Unicode symbols can be used in simple cases.
	markdown = strings.ReplaceAll(markdown, "$$$", "$")
	return markdown, imgURLs
}

func parseTime(selec *goquery.Selection) (t time.Time, err error) {
	comTime := selec.FindMatcher(timeSelec).AttrOr("title", "?!")
	if t, err = time.ParseInLocation("Jan/2/2006 15:04", comTime, moscowTZ); err != nil {
		// Russian locale has different format, don't ask me why.
		if t, err = time.ParseInLocation("2.1.2006 15:04", comTime, moscowTZ); err != nil {
			return
		}
	}
	t = t.UTC()
	return
}

func parseImg(selec *goquery.Selection) string {
	return withCodeforcesHost(selec.FindMatcher(imgSelec).AttrOr("src", "?!"))
}

func withCodeforcesHost(u string) string {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return ""
	}
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
	}
	if parsedURL.Host == "" {
		parsedURL.Host = "codeforces.com"
	}
	return parsedURL.String()
}
