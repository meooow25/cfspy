package fetch

import (
	"context"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
)

var (
	problemNameSelec         = cascadia.MustCompile(".problem-statement .header .title")
	problemNameAcmsguruSelec = cascadia.MustCompile(`[align]`)    // First align center
	contestNameSelec         = cascadia.MustCompile("#sidebar a") // Pick first. Couldn't find anything better :<
	contestStatusSelec       = cascadia.MustCompile(".contest-state-phase")
)

// Problem fetches problem information using the DefaultFetcher.
func Problem(ctx context.Context, url string) (*ProblemInfo, error) {
	return DefaultFetcher.Problem(ctx, url)
}

// Problem fetches problem information. The given URL must be a valid problem URL.
// TODO: Handle URLs like https://codeforces.com/gym/101002/K, which redirect to a page with the pdf
// of statements.
func (f *Fetcher) Problem(ctx context.Context, url string) (*ProblemInfo, error) {
	doc, err := f.FetchPage(ctx, url)
	if err != nil {
		return nil, err
	}

	var p ProblemInfo
	if p.Name = doc.FindMatcher(problemNameSelec).Text(); p.Name == "" {
		// Fallback for acmsguru. The name can be in a <div> or a <p>. "center" can be in lower or
		// caps.
		p.Name = doc.FindMatcher(problemNameAcmsguruSelec).
			FilterFunction(func(_ int, s *goquery.Selection) bool {
				return strings.EqualFold(s.AttrOr("align", "?!"), "center")
			}).
			First().
			Text()
		p.Name = strings.TrimSpace(p.Name)
	}
	p.ContestName = doc.FindMatcher(contestNameSelec).First().Text()
	p.ContestStatus = doc.FindMatcher(contestStatusSelec).Text()
	p.URL = url
	return &p, nil
}
