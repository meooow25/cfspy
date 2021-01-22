package fetch

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
)

var (
	infoDivSelec = cascadia.MustCompile(".info")
	rankSelec    = cascadia.MustCompile(".user-rank")
	photoSelec   = cascadia.MustCompile(".title-photo img")
	infoLiSelec  = cascadia.MustCompile("li")
	numberRe     = regexp.MustCompile("[0-9]+")
)

// Profile fetches user profile information using the DefaultFetcher.
func Profile(ctx context.Context, url string) (*ProfileInfo, error) {
	return DefaultFetcher.Profile(ctx, url)
}

// Profile fetches user profile information. The given URL must be a valid profile URL.
//
// Scrapes instead of using API because
// 1. If this is an old handle, it will be redirected to the new handle but the API errors out.
// 2. Scraped data shows "Headquarters" rank and black color sometimes which you don't get from the
//    API. So a preview generated by scraping better represents what one will see on opening the link.
func (f *Fetcher) Profile(ctx context.Context, url string) (*ProfileInfo, error) {
	doc, err := f.FetchPage(ctx, url)
	if err != nil {
		return nil, err
	}

	var p ProfileInfo
	infoDiv := doc.FindMatcher(infoDivSelec)
	p.Handle, p.Color = parseHandleAndColor(infoDiv)
	p.Rank = strings.TrimSpace(doc.FindMatcher(rankSelec).Text())
	lis := infoDiv.FindMatcher(infoLiSelec)
	lis.EachWithBreak(func(_ int, s *goquery.Selection) bool {
		text := s.Text()
		if strings.Contains(text, "Contest rating") || strings.Contains(text, "Рейтинг") {
			// Format is "Contest rating: <value> (max. <rank>, <value>)"
			text = strings.Join(strings.Fields(text), " ")
			numbers := numberRe.FindAllString(text, -1)
			if len(numbers) != 2 {
				err = fmt.Errorf("Unexpected format of rating line: %v", text)
				return false
			}
			p.Rating, _ = strconv.Atoi(numbers[0])
			p.MaxRating, _ = strconv.Atoi(numbers[1])
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	p.Avatar = withCodeforcesHost(doc.FindMatcher(photoSelec).AttrOr("src", "?!"))
	p.URL = url
	return &p, nil
}
