package fetch

import (
	"net/url"
	"regexp"
	"strconv"
)

var (
	queryAndFragment    = `/?\??[\w\-~!\*'\(\);:@&=\+\$,/\?%#\[\]]*`
	cfBlogURLRe         = regexp.MustCompile(`https?://codeforces.com/blog/entry/\d+` + queryAndFragment)
	cfCommentFragmentRe = regexp.MustCompile(`comment-(\d+)`)
	cfProblemURLRe      = regexp.MustCompile(`https?://codeforces.com/(?:(?:contest|gym)/\d+/problem|problemset/problem/\d+|problemsets/acmsguru/problem/\d+)/\w+` + queryAndFragment)
	submissionURLRe     = regexp.MustCompile(`https?://codeforces.com/(?:(?:contest|gym)/\d+/submission|problemset/submission/\d+)/\d+` + queryAndFragment)
	lineNumFragmentRe   = regexp.MustCompile(`L(\d+)(?:-L(\d+))?`)
)

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
			URL:        urlMatch,
			Suppressed: checkEmbedsSuppressed(s, idx[0], idx[1]),
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
			URL:        urlMatch,
			Suppressed: checkEmbedsSuppressed(s, idx[0], idx[1]),
		}
		matches = append(matches, match)
	}
	return matches
}

// ParseSubmissionURLs parses Codeforces submission URLs from the given string.
func ParseSubmissionURLs(s string) []SubmissionURLMatch {
	var matches []SubmissionURLMatch
	for _, idx := range submissionURLRe.FindAllStringSubmatchIndex(s, -1) {
		urlMatch := s[idx[0]:idx[1]]
		parsedURL, err := url.Parse(urlMatch)
		if err != nil {
			continue
		}
		match := SubmissionURLMatch{
			URL:        urlMatch,
			Suppressed: checkEmbedsSuppressed(s, idx[0], idx[1]),
		}
		lineNumsMatch := lineNumFragmentRe.FindStringSubmatch(parsedURL.Fragment)
		if len(lineNumsMatch) > 0 {
			if match.LineBegin, err = strconv.Atoi(lineNumsMatch[1]); err == nil {
				if match.LineEnd, err = strconv.Atoi(lineNumsMatch[2]); err != nil {
					match.LineEnd = match.LineBegin
				}
			}
			if match.LineBegin > match.LineEnd {
				match.LineBegin, match.LineEnd = match.LineEnd, match.LineBegin
			}
		}
		matches = append(matches, match)
	}
	return matches
}

// Checks whether the given substring is surrounded by <>. Used to check if a link embed is
// suppressed.
func checkEmbedsSuppressed(s string, start, end int) bool {
	return start > 0 && s[start-1] == '<' && end < len(s) && s[end] == '>'
}
