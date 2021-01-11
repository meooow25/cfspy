package fetch

import (
	"time"

	"github.com/togatoga/goforces"
)

// BlogURLMatch contains matched information for a blog URL.
type BlogURLMatch struct {
	URL       string
	CommentID string
}

// ProblemURLMatch contains matched information for a problem URL.
type ProblemURLMatch struct {
	URL string
}

// SubmissionURLMatch contains matched information for a submission URL.
type SubmissionURLMatch struct {
	URL       string
	LineBegin int
	LineEnd   int
}

// ProfileURLMatch contains matched information for a profile URL.
type ProfileURLMatch struct {
	URL    string
	Handle string
}

// BlogInfo contains blog information.
type BlogInfo struct {
	Title        string
	CreationTime time.Time
	AuthorHandle string
	AuthorAvatar string
	AuthorColor  int
	Rating       int
	URL          string
}

// CommentInfo contains comment information.
type CommentInfo struct {
	Content       string
	Images        []string
	BlogTitle     string
	CreationTime  time.Time
	AuthorHandle  string
	AuthorAvatar  string
	AuthorColor   int
	RevisionCount int
	Revision      int
	Rating        int
	URL           string
}

// ProblemInfo contains problem information.
type ProblemInfo struct {
	Name          string
	ContestName   string
	ContestStatus string
	URL           string
}

// SubmissionInfoAuthor contains submission author information.
type SubmissionInfoAuthor struct {
	Handle string
	Color  int
}

// SubmissionInfoTeam contains submission author team information.
type SubmissionInfoTeam struct {
	Name    string
	Authors []*SubmissionInfoAuthor
}

// SubmissionInfo contains submission information.
type SubmissionInfo struct {
	// These are mutually exclusive.
	Author      *SubmissionInfoAuthor
	AuthorTeam  *SubmissionInfoTeam
	AuthorGhost string

	Problem  string
	Language string
	Verdict  string
	Type     string
	SentTime time.Time
	Content  string
	URL      string
}

type ProfileInfo struct {
	*goforces.User
	Color int
	URL   string
}
