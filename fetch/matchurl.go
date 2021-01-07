package fetch

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	queryAndFragment  = `/?\??[\w\-~!\*'\(\);:@&=\+\$,/\?%#\[\]]*`
	blogURLRe         = regexp.MustCompile(`https?://codeforces.com/blog/entry/\d+` + queryAndFragment)
	commentFragmentRe = regexp.MustCompile(`comment-(\d+)`)
	problemURLRe      = regexp.MustCompile(`https?://codeforces.com/(?:(?:contest|gym)/\d+/problem|problemset/problem/\d+|problemsets/acmsguru/problem/\d+)/\w+` + queryAndFragment)
	submissionURLRe   = regexp.MustCompile(`https?://codeforces.com/(?:(?:contest|gym)/\d+/submission|problemset/submission/\d+)/\d+` + queryAndFragment)
	lineNumFragmentRe = regexp.MustCompile(`L(\d+)(?:-L(\d+))?`)
)

// ParseBlogURLs parses Codeforces blog URLS from the given string.
func ParseBlogURLs(s string) []*BlogURLMatch {
	s = removeSpoilers(s)
	var matches []*BlogURLMatch
	for _, idx := range blogURLRe.FindAllStringSubmatchIndex(s, -1) {
		if checkEmbedsSuppressed(s, idx[0], idx[1]) {
			continue
		}
		urlMatch := s[idx[0]:idx[1]]
		parsedURL, err := url.Parse(urlMatch)
		if err != nil {
			continue
		}
		match := BlogURLMatch{
			URL: urlMatch,
		}
		commentMatch := commentFragmentRe.FindStringSubmatch(parsedURL.Fragment)
		if len(commentMatch) > 0 {
			match.CommentID = commentMatch[1]
		}
		matches = append(matches, &match)
	}
	return matches
}

// ParseProblemURLs parses Codeforces problem URLS from the given string.
func ParseProblemURLs(s string) []*ProblemURLMatch {
	s = removeSpoilers(s)
	var matches []*ProblemURLMatch
	for _, idx := range problemURLRe.FindAllStringSubmatchIndex(s, -1) {
		if checkEmbedsSuppressed(s, idx[0], idx[1]) {
			continue
		}
		urlMatch := s[idx[0]:idx[1]]
		if _, err := url.Parse(urlMatch); err != nil {
			continue
		}
		match := ProblemURLMatch{
			URL: urlMatch,
		}
		matches = append(matches, &match)
	}
	return matches
}

// ParseSubmissionURLs parses Codeforces submission URLs from the given string.
func ParseSubmissionURLs(s string) []*SubmissionURLMatch {
	s = removeSpoilers(s)
	var matches []*SubmissionURLMatch
	for _, idx := range submissionURLRe.FindAllStringSubmatchIndex(s, -1) {
		if checkEmbedsSuppressed(s, idx[0], idx[1]) {
			continue
		}
		urlMatch := s[idx[0]:idx[1]]
		parsedURL, err := url.Parse(urlMatch)
		if err != nil {
			continue
		}
		match := SubmissionURLMatch{
			URL: urlMatch,
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
		matches = append(matches, &match)
	}
	return matches
}

// Removes spoilered parts of the input string (parts surrounded by ||).
func removeSpoilers(s string) string {
	marker := "||"
	var result strings.Builder
	for {
		var i int
		if i = strings.Index(s, marker); i == -1 {
			result.WriteString(s)
			break
		}
		result.WriteString(s[:i])
		s = s[i+len(marker):]
		if i = strings.Index(s, marker); i == -1 {
			result.WriteString(marker)
			result.WriteString(s)
			break
		}
		s = s[i+len(marker):]
	}
	return result.String()
}

// Checks whether the given substring is surrounded by <>. Used to check if a link embed is
// suppressed.
func checkEmbedsSuppressed(s string, start, end int) bool {
	return start > 0 && s[start-1] == '<' && end < len(s) && s[end] == '>'
}
