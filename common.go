package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	urlpkg "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/togatoga/goforces"
)

var (
	// Ordinary client
	cfScraper = &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: redirectPolicyFunc,
		Jar:           newCFJar("RCPC"),
	}

	// Client that uses a browser user agent
	cfScraperBrowser = &http.Client{
		Transport:     &browserUATransport{},
		Timeout:       10 * time.Second,
		CheckRedirect: redirectPolicyFunc,
		Jar:           newCFJar("JSESSIONID", "RCPC"),
	}

	// API client
	cfAPI, _ = goforces.NewClient(nil)

	// Useful for scraping
	titleSelec         = cascadia.MustCompile(".title")
	handleSelec        = cascadia.MustCompile("a.rated-user")
	timeSelec          = cascadia.MustCompile(".info .format-humantime")
	moscowTZ           = time.FixedZone("Europe/Moscow", int(3*time.Hour/time.Second))
	commentAvatarSelec = cascadia.MustCompile(".avatar")
	imgSelec           = cascadia.MustCompile("img")
	scriptSelec        = cascadia.MustCompile("script")

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

// Transport that sets a browser user agent.
type browserUATransport struct{}

func (t *browserUATransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set(
		"User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:78.0) Gecko/20100101 Firefox/78.0",
	)
	return http.DefaultTransport.RoundTrip(req)
}

// Cookie jar that does not persist more state than necessary.
type cfJar struct {
	*cookiejar.Jar
	allowed map[string]bool
}

func newCFJar(allowedCookies ...string) *cfJar {
	jar, _ := cookiejar.New(nil)
	allowed := make(map[string]bool)
	for _, name := range allowedCookies {
		allowed[name] = true
	}
	return &cfJar{Jar: jar, allowed: allowed}
}

func (j *cfJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	var allowed []*http.Cookie
	for _, cookie := range cookies {
		if j.allowed[cookie.Name] {
			allowed = append(allowed, cookie)
		}
	}
	if len(allowed) > 0 {
		j.Jar.SetCookies(u, allowed)
	}
}

type redirectErr struct {
	From *url.URL
	To   *url.URL
}

func (err *redirectErr) Error() string {
	return fmt.Sprintf("Redirect from %v to %v", err.From, err.To)
}

type scrapeFetchErr struct {
	URL *url.URL
	Err error
}

func (err *scrapeFetchErr) Error() string {
	return fmt.Errorf("Error fetching from <%v>: %w", err.URL, err.Err).Error()
}

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	return &redirectErr{From: via[len(via)-1].URL, To: req.URL}
}

// scraperGetDoc fetches the page from the given URL and returns a parsed goquery document. Uses the
// cfScraper client.
func scraperGetDoc(url string) (*goquery.Document, error) {
	return scraperGetDocInternal(url, cfScraper)
}

// Same as scraperGetDoc but uses the cfScraperBrowser client, which uses a browser user agent.
func scraperGetDocBrowser(url string) (*goquery.Document, error) {
	return scraperGetDocInternal(url, cfScraperBrowser)
}

func scraperGetDocInternal(url string, client *http.Client) (doc *goquery.Document, err error) {
	parsedURL, err := urlpkg.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("Error parsing URL %q: %w", url, err)
	}
	parsedURL.Fragment = ""
	parsedURL.ForceQuery = false

	doc, err = fetch(parsedURL, client)
	if err != nil {
		return
	}
	scripts := doc.FindMatcher(scriptSelec)
	if scripts.Length() > 2 { // Got the right page, setting RCPC not needed.
		return
	}
	if err = setRCPCCookieOnClient(scripts.Text(), client); err != nil {
		err = fmt.Errorf("Set RCPC cookie failed: %w", err)
		return
	}
	return fetch(parsedURL, client)
}

func fetch(url *urlpkg.URL, client *http.Client) (*goquery.Document, error) {
	resp, err := client.Get(url.String())
	if err != nil {
		inner := errors.Unwrap(err)
		if _, ok := inner.(*redirectErr); ok {
			// Instead of serving a 404 page if the resourse is missing, Codeforces redirects to the
			// last visited page. Don't ask me why.
			err = fmt.Errorf("Page not found")
		}
		return nil, &scrapeFetchErr{URL: url, Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, &scrapeFetchErr{URL: url, Err: fmt.Errorf("%v", resp.Status)}
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error parsing HTML from %q: %w", url, err)
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

// Fetches a comment revision. This endpoint rejects non-browser user agents, so cfScraperBrowser is
// used. This endpoint only works if there is more than one revision, otherwise it would be usable
// for fetching comments more easily. Requires a CSRF token and the JSESSIONID cookie. The session
// cookie should already be present if cfScraperBrowser was used to fetch a page before this.
func fetchCommentBrowser(
	commentID string,
	revision int,
	csrfToken string,
) (content string, err error) {
	formData := urlpkg.Values{
		"action":     {"revision"},
		"commentId":  {commentID},
		"revision":   {strconv.Itoa(revision)},
		"csrf_token": {csrfToken},
	}
	req, _ := http.NewRequest(
		"POST",
		"http://codeforces.com/data/comment-data",
		strings.NewReader(formData.Encode()),
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := cfScraperBrowser.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		err = fmt.Errorf("%v", resp.Status)
		return
	}
	var jsonResp struct {
		Content string `json:"content"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		err = fmt.Errorf("JSON decode error: %w", err)
		return
	}
	if jsonResp.Content == "" {
		err = errors.New("'content' not present in JSON response")
		return
	}
	content = jsonResp.Content
	return
}
