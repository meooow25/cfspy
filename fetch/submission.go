package fetch

import (
	"context"
	"strings"
	"time"

	"github.com/andybalholm/cascadia"
)

var (
	infoRowSelec  = cascadia.MustCompile(".datatable tr") // Pick second
	infoCellSelec = cascadia.MustCompile("td")
	problemSelec  = cascadia.MustCompile("a")
	sourceSelec   = cascadia.MustCompile("#program-source-text")
)

// Submission fetches the submission source code and accompanying information. The given URL must be
// a valid submission URL.
func Submission(ctx context.Context, url string) (*SubmissionInfo, error) {
	doc, err := scraperGetDoc(ctx, url)
	if err != nil {
		return nil, err
	}

	// Rows are
	// # | Author | Problem | Lang | Verdict | Time | Memory | Sent | Judged | <Compare>

	var s SubmissionInfo
	infoRow := doc.FindMatcher(infoRowSelec).Eq(1).FindMatcher(infoCellSelec)

	s.Type = strings.TrimSuffix(strings.TrimSpace(infoRow.Eq(1).Contents().First().Text()), ":")
	s.AuthorHandle, s.AuthorColor = parseHandleAndColor(infoRow.Eq(1))
	s.Problem = infoRow.Eq(2).FindMatcher(problemSelec).Text()
	s.Language = strings.TrimSpace(infoRow.Eq(3).Text())
	s.Verdict = strings.TrimSpace(infoRow.Eq(4).Text())
	s.SentTime, err = parseSubmissionTime(infoRow.Eq(7).Text())
	if err != nil {
		return nil, err
	}
	s.Content = doc.FindMatcher(sourceSelec).Text()
	s.URL = url
	return &s, nil
}

func parseSubmissionTime(text string) (t time.Time, err error) {
	if t, err = time.ParseInLocation("2006-01-02 15:04:05", text, moscowTZ); err != nil {
		return
	}
	t = t.UTC()
	return
}
