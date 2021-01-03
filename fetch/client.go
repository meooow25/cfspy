package fetch

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/togatoga/goforces"
)

var (
	// Ordinary client
	cfScraper = &http.Client{
		Jar: newRestrictedJar("JSESSIONID", "RCPC"),
	}

	// Client that uses a browser user agent
	cfScraperBrowser = &http.Client{
		Transport: &browserUATransport{},
		Jar:       newRestrictedJar("JSESSIONID", "RCPC"),
	}

	// API client
	cfAPI, _ = goforces.NewClient(nil)
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

// Cookie jar that accepts only a fixed set of cookie names.
type restrictedJar struct {
	*cookiejar.Jar
	allowed map[string]bool
}

func newRestrictedJar(allowedCookies ...string) *restrictedJar {
	jar, _ := cookiejar.New(nil)
	allowed := make(map[string]bool)
	for _, name := range allowedCookies {
		allowed[name] = true
	}
	return &restrictedJar{Jar: jar, allowed: allowed}
}

func (j *restrictedJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	var allowed []*http.Cookie
	for _, cookie := range cookies {
		if j.allowed[cookie.Name] {
			allowed = append(allowed, cookie)
		}
	}
	j.Jar.SetCookies(u, allowed)
}

// These are for bypassing a cookie check introduced by Codeforces.
// See https://github.com/meooow25/cfspy/issues/4

var (
	aRe = regexp.MustCompile(`a=toNumbers\("([0-9a-f]+)"\)`)
	bRe = regexp.MustCompile(`b=toNumbers\("([0-9a-f]+)"\)`)
	cRe = regexp.MustCompile(`c=toNumbers\("([0-9a-f]+)"\)`)
)

func setRCPCCookieOnClient(script string, client *http.Client) error {
	var a, b, c string
	if match := aRe.FindStringSubmatch(script); match != nil {
		a = match[1]
	} else {
		return errors.New("a not found")
	}
	if match := bRe.FindStringSubmatch(script); match != nil {
		b = match[1]
	} else {
		return errors.New("b not found")
	}
	if match := cRe.FindStringSubmatch(script); match != nil {
		c = match[1]
	} else {
		return errors.New("c not found")
	}

	// Adapted from example at https://golang.org/pkg/crypto/cipher/#NewCBCDecrypter
	key, err := hex.DecodeString(a)
	if err != nil {
		return err
	}
	ciphertext, err := hex.DecodeString(c)
	if err != nil {
		return err
	}
	iv, err := hex.DecodeString(b)
	if err != nil {
		return err
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return errors.New("ciphertext is not a multiple of the block size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	rcpc := &http.Cookie{
		Name:  "RCPC",
		Value: hex.EncodeToString(ciphertext),
		Path:  "/",
	}
	cfURL := &url.URL{Scheme: "https", Host: "codeforces.com"}
	client.Jar.SetCookies(cfURL, []*http.Cookie{rcpc})
	return nil
}

// scraperGetDoc fetches the page from the given URL and returns a parsed goquery document. Uses the
// cfScraper client.
func scraperGetDoc(ctx context.Context, url string) (*goquery.Document, error) {
	return scraperGetDocInternal(ctx, url, cfScraper)
}

// Same as scraperGetDoc but uses the cfScraperBrowser client, which uses a browser user agent.
func scraperGetDocBrowser(ctx context.Context, url string) (*goquery.Document, error) {
	return scraperGetDocInternal(ctx, url, cfScraperBrowser)
}

var errorMsgRe = regexp.MustCompile(`Codeforces.showMessage\("(.*)"\);\s*Codeforces\.reformatTimes`)

func scraperGetDocInternal(ctx context.Context, url string, client *http.Client) (*goquery.Document, error) {
	doc, err := fetch(ctx, url, client)
	if err != nil {
		return nil, err
	}
	scripts := doc.FindMatcher(scriptSelec)
	if scripts.Length() <= 2 {
		// Got RCPC page, set cookie and refetch
		if err = setRCPCCookieOnClient(scripts.Text(), client); err != nil {
			return nil, fmt.Errorf("Set RCPC cookie failed: %w", err)
		}
		if doc, err = fetch(ctx, url, client); err != nil {
			return nil, err
		}
		scripts = doc.FindMatcher(scriptSelec)
	}
	// Instead of serving a 404 page if the resourse is missing, Codeforces redirects to the last
	// visited page and shows an error message. Don't ask me why.
	if match := errorMsgRe.FindStringSubmatch(scripts.Text()); match != nil {
		return nil, errors.New(match[1])
	}
	// If the page is not public, Codeforces redirects to the login page.
	if doc.Url.Path == "/enter" {
		return nil, errors.New("Login is required to access this page")
	}
	return doc, nil
}

func fetch(ctx context.Context, url string, client *http.Client) (*goquery.Document, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%v", resp.Status)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "text/html") {
		return nil, fmt.Errorf("Expected text/html, got %v", contentType)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error parsing HTML from %v: %w", url, err)
	}
	doc.Url = resp.Request.URL
	return doc, nil
}

// Fetches a comment revision. This endpoint rejects non-browser user agents, so cfScraperBrowser is
// used. This endpoint only works if there is more than one revision, otherwise it would be usable
// for fetching comments more easily. Requires a CSRF token and the JSESSIONID cookie. The session
// cookie should already be present if cfScraperBrowser was used to fetch a page before this.
func fetchCommentBrowser(
	ctx context.Context,
	commentID string,
	revision int,
	csrfToken string,
) (*goquery.Document, error) {
	formData := url.Values{
		"action":     {"revision"},
		"commentId":  {commentID},
		"revision":   {strconv.Itoa(revision)},
		"csrf_token": {csrfToken},
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://codeforces.com/data/comment-data",
		strings.NewReader(formData.Encode()),
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := cfScraperBrowser.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%v", resp.Status)
	}
	var jsonResp struct {
		Content string `json:"content"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return nil, fmt.Errorf("JSON decode error: %w", err)
	}
	if jsonResp.Content == "" {
		return nil, errors.New("'content' not present in JSON response")
	}
	return goquery.NewDocumentFromReader(strings.NewReader(jsonResp.Content))
}

// Fetches the avatar URL for a handle using the Codeforces API.
func fetchAvatar(ctx context.Context, handle string) (string, error) {
	infos, err := cfAPI.GetUserInfo(ctx, []string{handle})
	if err != nil {
		return "", err
	}
	return withCodeforcesHost(infos[0].Avatar), nil
}

// Fetcher is a Codeforces info fetcher.
type Fetcher struct {
	FetchPage            func(ctx context.Context, url string) (*goquery.Document, error)
	FetchPageBrowser     func(ctx context.Context, url string) (*goquery.Document, error)
	FetchCommentRevision func(ctx context.Context, commentID string, revision int, csrfToken string) (*goquery.Document, error)
	FetchAvatar          func(ctx context.Context, handle string) (string, error)
}

// DefaultFetcher is the default fetcher that fetches data from Codeforces web and API.
var DefaultFetcher = Fetcher{
	FetchPage:            scraperGetDoc,
	FetchPageBrowser:     scraperGetDocBrowser,
	FetchCommentRevision: fetchCommentBrowser,
	FetchAvatar:          fetchAvatar,
}
