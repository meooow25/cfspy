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
		CheckRedirect: redirectPolicyFunc,
		Jar:           newRestrictedJar("RCPC"),
	}

	// Client that uses a browser user agent
	cfScraperBrowser = &http.Client{
		Transport:     &browserUATransport{},
		CheckRedirect: redirectPolicyFunc,
		Jar:           newRestrictedJar("JSESSIONID", "RCPC"),
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
	if len(allowed) > 0 {
		j.Jar.SetCookies(u, allowed)
	}
}

type redirectErr struct {
	From, To *url.URL
}

func (err *redirectErr) Error() string {
	return fmt.Sprintf("Redirect from %v to %v", err.From, err.To)
}

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	return &redirectErr{From: via[len(via)-1].URL, To: req.URL}
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

func scraperGetDocInternal(ctx context.Context, url string, client *http.Client) (*goquery.Document, error) {
	doc, err := fetch(ctx, url, client)
	if err != nil {
		return nil, err
	}
	scripts := doc.FindMatcher(scriptSelec)
	if scripts.Length() > 2 { // Got the right page, setting RCPC not needed.
		return doc, nil
	}
	if err = setRCPCCookieOnClient(scripts.Text(), client); err != nil {
		return nil, fmt.Errorf("Set RCPC cookie failed: %w", err)
	}
	return fetch(ctx, url, client)
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
		var r *redirectErr
		if errors.As(err, &r) {
			// Instead of serving a 404 page if the resourse is missing, Codeforces redirects to the
			// last visited page. Don't ask me why.
			err = fmt.Errorf("Page not found")
		}
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
) (string, error) {
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
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("%v", resp.Status)
	}
	var jsonResp struct {
		Content string `json:"content"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return "", fmt.Errorf("JSON decode error: %w", err)
	}
	if jsonResp.Content == "" {
		return "", errors.New("'content' not present in JSON response")
	}
	return jsonResp.Content, nil
}
