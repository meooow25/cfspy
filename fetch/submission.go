package fetch

import (
	"context"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
)

var (
	infoRowSelec  = cascadia.MustCompile(".datatable tr") // Pick second
	infoCellSelec = cascadia.MustCompile("td")
	ghostSelec    = cascadia.MustCompile(`span[title="Ghost participant"]`)
	teamNameSelec = cascadia.MustCompile(`a[href^="/team"]`)
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
	return parseSubmission(url, doc)
}

func parseSubmission(url string, doc *goquery.Document) (*SubmissionInfo, error) {

	// Rows are
	// # | Author | Problem | Lang | Verdict | Time | Memory | Sent | Judged | <Compare>

	var s SubmissionInfo
	infoRow := doc.FindMatcher(infoRowSelec).Eq(1).FindMatcher(infoCellSelec)

	s.Type = strings.TrimSuffix(strings.TrimSpace(infoRow.Eq(1).Contents().First().Text()), ":")
	authorCell := infoRow.Eq(1)
	if s.AuthorGhost = parseGhost(authorCell); s.AuthorGhost == "" {
		authors := parseAuthors(authorCell)
		if teamName := parseTeamName(authorCell); teamName != "" {
			s.AuthorTeam = &SubmissionInfoTeam{
				Name:    teamName,
				Authors: authors,
			}
		} else {
			s.Author = authors[0]
		}
	}
	s.Problem = infoRow.Eq(2).FindMatcher(problemSelec).Text()
	s.Language = strings.TrimSpace(infoRow.Eq(3).Text())
	s.Verdict = strings.TrimSpace(infoRow.Eq(4).Text())
	var err error
	if s.SentTime, err = parseSubmissionTime(infoRow.Eq(7).Text()); err != nil {
		return nil, err
	}
	s.Content = doc.FindMatcher(sourceSelec).Text()
	s.URL = url
	return &s, nil
}

func parseGhost(authorCell *goquery.Selection) string {
	if s := authorCell.FindMatcher(ghostSelec); s.Length() != 0 {
		return s.Text()
	}
	return ""
}

func parseTeamName(authorCell *goquery.Selection) string {
	if s := authorCell.FindMatcher(teamNameSelec); s.Length() != 0 {
		return s.Text()
	}
	return ""
}

func parseAuthors(authorCell *goquery.Selection) []*SubmissionInfoAuthor {
	var authors []*SubmissionInfoAuthor
	authorCell.FindMatcher(handleSelec).Each(func(_ int, s *goquery.Selection) {
		authors = append(authors, &SubmissionInfoAuthor{
			Handle: s.Text(),
			Color:  userColor(s),
		})
	})
	return authors
}

func parseSubmissionTime(text string) (t time.Time, err error) {
	if t, err = time.ParseInLocation("2006-01-02 15:04:05", text, moscowTZ); err != nil {
		return
	}
	t = t.UTC()
	return
}
