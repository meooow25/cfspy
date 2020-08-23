package fetch

import "time"

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
