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
	handleSelec        = cascadia.MustCompile("a.rated-user")
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

	// From https://sta.codeforces.com/s/50332/css/community.css
	colorClsMap = map[string]int{
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
)

// Blog fetches blog information. The given URL must be a valid blog URL.
func Blog(ctx context.Context, url string) (*BlogInfo, error) {
	doc, err := scraperGetDoc(ctx, url)
	if err != nil {
		return nil, err
	}

	var b BlogInfo
	b.URL = url
	b.Title = doc.FindMatcher(titleSelec).First().Text()
	blogDiv := doc.FindMatcher(blogSelec)
	b.AuthorHandle = blogDiv.FindMatcher(handleSelec).First().Text()
	if b.CreationTime, err = parseTime(blogDiv); err != nil {
		return nil, err
	}
	if b.Rating, err = strconv.Atoi(blogDiv.FindMatcher(blogRatingSelec).Text()); err != nil {
		return nil, err
	}
	b.AuthorColor = parseHandleColor(blogDiv)

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
		infos, err := cfAPI.GetUserInfo(ctx, []string{b.AuthorHandle})
		if err != nil {
			return nil, err
		}
		b.AuthorAvatar = withCodeforcesHost(infos[0].Avatar)
	}

	return &b, nil
}

// CommentInfoGetter is a function that returns the comment info for a given revision.
type CommentInfoGetter func(revision int) (*CommentInfo, error)

// Comment fetches comment information. The given URL must be a valid comment URL. A
// CommentInfoGetter is returned. The last revision is immediately available, other revisions are
// fetched lazily when the CommentInfoGetter is called.
func Comment(
	ctx context.Context,
	url string,
	commentID string,
) (revisionCount int, getter CommentInfoGetter, err error) {
	doc, err := scraperGetDocBrowser(ctx, url)
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
	base.BlogTitle = doc.FindMatcher(titleSelec).First().Text()
	avatarDiv := comment.FindMatcher(commentAvatarSelec)
	if base.CreationTime, err = parseTime(comment); err != nil {
		return
	}
	base.AuthorHandle = avatarDiv.FindMatcher(handleSelec).First().Text()
	base.AuthorAvatar = parseImg(avatarDiv)
	if base.Rating, err = strconv.Atoi(comment.FindMatcher(commentRatingSelec).Text()); err != nil {
		return
	}
	base.AuthorColor = parseHandleColor(avatarDiv)

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
			html, err := fetchCommentBrowser(ctx, commentID, revision, csrf)
			if err != nil {
				return nil, err
			}
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
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

func parseHandleColor(selec *goquery.Selection) int {
	clss := strings.Fields(
		selec.FindMatcher(handleSelec).AttrOr("class", "?!"))
	for _, cls := range clss {
		if col, ok := colorClsMap[cls]; ok {
			return col
		}
	}
	return 0x000000
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
