package main

import (
	"fmt"
	"net/http"
	"net/url"
	urlpkg "net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andersfylling/disgord"
	"github.com/andybalholm/cascadia"
	"github.com/meooow25/cfspy/bot"
	"github.com/togatoga/goforces"
)

var (
	// Global clients.
	cfScraper = http.Client{Timeout: 10 * time.Second}
	cfAPI, _  = goforces.NewClient(nil)

	// Useful for scraping
	titleSelec         = cascadia.MustCompile(".title")
	handleSelec        = cascadia.MustCompile("a.rated-user")
	timeSelec          = cascadia.MustCompile(".info .format-humantime")
	moscowTZ           = time.FixedZone("Europe/Moscow", int(3*time.Hour/time.Second))
	commentAvatarSelec = cascadia.MustCompile(".avatar")
	imgSelec           = cascadia.MustCompile("img")

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

type scrapeFetchErr struct {
	URL *url.URL
	Err error
}

func (err *scrapeFetchErr) Error() string {
	return fmt.Errorf("Error fetching from %v: %w", err.URL, err.Err).Error()
}

// scraperGetDoc fetches the page from the given URL and returns a parsed goquery document.
func scraperGetDoc(url string) (*goquery.Document, error) {
	parsedURL, err := urlpkg.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("Error parsing URL %q: %w", url, err)
	}
	parsedURL.Fragment = ""
	parsedURL.ForceQuery = false
	resp, err := cfScraper.Get(parsedURL.String())
	if err != nil {
		return nil, &scrapeFetchErr{URL: parsedURL, Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, &scrapeFetchErr{URL: parsedURL, Err: fmt.Errorf("%v", resp.Status)}
	}
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("Error parsing HTML from %q: %w", parsedURL, err)
	}
	return doc, nil
}

func parseTitle(doc *goquery.Document) string {
	return doc.FindMatcher(titleSelec).First().Text()
}

func parseHandle(selec *goquery.Selection) string {
	return selec.FindMatcher(handleSelec).First().Text()
}

func parseHandleColor(selec *goquery.Selection) int {
	clss := strings.Fields(
		selec.FindMatcher(handleSelec).AttrOr("class", "missing-user-color-unexpected"))
	for _, cls := range clss {
		if col, ok := colorClsMap[cls]; ok {
			return col
		}
	}
	return 0x000000
}

func parseTime(selec *goquery.Selection) (t time.Time, err error) {
	comTime := selec.FindMatcher(timeSelec).AttrOr("title", "missing-time-unexpected")
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
	return withCodeforcesHost(
		selec.FindMatcher(imgSelec).AttrOr("src", "missing-src-unexpected"))
}

func withCodeforcesHost(url string) string {
	parsedURL, err := urlpkg.Parse(url)
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

// This is rather specific.
// Returns a handler that deletes widget and unsuppresses embeds on original, only if the
// deleter is the author of original.
func getWidgetDeleteHandler(widget *disgord.Message, original *disgord.Message) bot.ReactHandler {
	return func(s disgord.Session, evt *disgord.MessageReactionAdd) {
		if evt.UserID != original.Author.ID {
			return
		}
		// TODO: This is hacky, improve. Shouldn't use old ctx and shouldn't repeat logic.
		ctx2 := bot.Context{
			Ctx:     evt.Ctx,
			Session: s,
		}
		ctx2.DeleteMsg(widget)
		// This too will fail without manage messages permission, ignore.
		ctx2.UnsuppressEmbeds(original)
	}
}
