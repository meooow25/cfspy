package fetch

import (
	"net/url"
	"regexp"
)

var (
	cfBlogURLRe         = regexp.MustCompile(`https?://codeforces.com/blog/entry/(\d+)\??\S*`)
	cfCommentFragmentRe = regexp.MustCompile(`comment-(\d+)`)
	cfProblemURLRe      = regexp.MustCompile(`https?://codeforces.com/(?:(?:contest|gym)/(\d+)/problem|problemset/problem/(\d+)|problemsets/acmsguru/problem/(\d+))/(\S+)\??\S*`)
)

// BlogURLMatch contains matched information for a blog URL.
type BlogURLMatch struct {
	URL        string
	BlogID     string
	CommentID  string
	Start, End int
}

// ProblemURLMatch contains matched information for a problem URL.
type ProblemURLMatch struct {
	URL        string
	ContestID  string
	ProblemID  string
	Start, End int
}

// ParseBlogURLs parses Codeforces blog URLS from the given string.
func ParseBlogURLs(s string) []BlogURLMatch {
	var matches []BlogURLMatch
	for _, idx := range cfBlogURLRe.FindAllStringSubmatchIndex(s, -1) {
		urlMatch := s[idx[0]:idx[1]]
		parsedURL, err := url.Parse(urlMatch)
		if err != nil {
			continue
		}
		match := BlogURLMatch{
			URL:    urlMatch,
			BlogID: s[idx[2]:idx[3]],
			Start:  idx[0],
			End:    idx[1],
		}
		commentMatch := cfCommentFragmentRe.FindStringSubmatch(parsedURL.Fragment)
		if len(commentMatch) > 0 {
			match.CommentID = commentMatch[1]
		}
		matches = append(matches, match)
	}
	return matches
}

// ParseProblemURLs parses Codeforces problem URLS from the given string.
func ParseProblemURLs(s string) []ProblemURLMatch {
	var matches []ProblemURLMatch
	for _, idx := range cfProblemURLRe.FindAllStringSubmatchIndex(s, -1) {
		urlMatch := s[idx[0]:idx[1]]
		if _, err := url.Parse(urlMatch); err != nil {
			continue
		}
		match := ProblemURLMatch{
			URL:       urlMatch,
			ProblemID: s[idx[8]:idx[9]],
			Start:     idx[0],
			End:       idx[1],
		}
		for i := 2; i < 8; i += 2 {
			if idx[i] != -1 {
				match.ContestID = s[idx[i]:idx[i+1]]
				break
			}
		}
		matches = append(matches, match)
	}
	return matches
}
