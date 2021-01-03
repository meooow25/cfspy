package fetch

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-test/deep"
)

func testParseBlog(t *testing.T, filename string, want *BlogInfo) {
	f := Fetcher{
		FetchPage: pageFetcherFor(filename, "testurl"),
		FetchAvatar: func(_ context.Context, handle string) (string, error) {
			if handle != want.AuthorHandle {
				t.Fatalf("got %v, want %v", handle, want.AuthorHandle)
			}
			return "fetchedavatarurl", nil
		},
	}
	got, err := f.Blog(context.Background(), "testurl")
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(got, want); diff != nil {
		t.Fatal(diff)
	}
}

func TestParseBlog(t *testing.T) {
	t.Run("blogAuthorHasComment", func(t *testing.T) {
		want := &BlogInfo{
			Title:        "Testing Beta Round (Unrated)",
			CreationTime: time.Date(2020, 7, 24, 5, 37, 0, 0, time.UTC),
			AuthorHandle: "MikeMirzayanov",
			AuthorAvatar: "https://userpic.codeforces.com/11/avatar/7d0648e330a57263.jpg",
			AuthorColor:  colorClsMap["user-admin"],
			Rating:       130,
			URL:          "testurl",
		}
		testParseBlog(t, "blog_entry_80540.html", want)
	})

	t.Run("blogAuthorHasNoComment", func(t *testing.T) {
		want := &BlogInfo{
			Title:        "EDU: Segment Tree, part 1",
			CreationTime: time.Date(2020, 7, 12, 10, 32, 0, 0, time.UTC),
			AuthorHandle: "pashka",
			AuthorAvatar: "fetchedavatarurl",
			AuthorColor:  colorClsMap["user-red"],
			Rating:       1309,
			URL:          "testurl",
		}
		testParseBlog(t, "blog_entry_80031.html", want)
	})
}

const commentRevFmt = `<div class="ttypography"><p>comment revision %v</p></div>`

func testParseComment(t *testing.T, filename string, commentID string, want *CommentInfo) {
	f := Fetcher{
		FetchPageBrowser: pageFetcherFor(filename, "testurl"),
		FetchCommentRevision: func(_ context.Context, gotCommentID string, revision int, _ string) (*goquery.Document, error) {
			if gotCommentID != commentID {
				t.Fatalf("got %v, want %v", gotCommentID, commentID)
			}
			return goquery.NewDocumentFromReader(strings.NewReader(fmt.Sprintf(commentRevFmt, revision)))
		},
	}
	gotRevCnt, getter, err := f.Comment(context.Background(), "testurl", commentID)
	if err != nil {
		t.Fatal(err)
	}
	if gotRevCnt != want.RevisionCount {
		t.Fatalf("got %v, want %v", gotRevCnt, want.RevisionCount)
	}
	got, err := getter(want.RevisionCount)
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(got, want); diff != nil {
		t.Fatal(diff)
	}
	for i := 1; i < want.RevisionCount; i++ {
		got, err = getter(i)
		if err != nil {
			t.Fatal(err)
		}
		wantCopy := *want
		wantCopy.Revision = i
		wantCopy.Content = fmt.Sprintf("comment revision %v", i)
		if diff := deep.Equal(got, &wantCopy); diff != nil {
			t.Fatal(diff)
		}
	}
}

func TestParseComment(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		want := &CommentInfo{
			Content:       "So is this just a test on the interface and not the judging servers? Should we try to submit repeated TLE solutions?",
			BlogTitle:     "Testing Beta Round (Unrated)",
			CreationTime:  time.Date(2020, 7, 24, 5, 41, 0, 0, time.UTC),
			AuthorHandle:  "qpwoeirut",
			AuthorColor:   colorClsMap["user-orange"],
			AuthorAvatar:  "https://userpic.codeforces.com/no-avatar.jpg",
			RevisionCount: 1,
			Revision:      1,
			Rating:        4,
			URL:           "testurl",
		}
		testParseComment(t, "blog_entry_80540.html", "667681", want)
	})

	t.Run("withRevisions", func(t *testing.T) {
		want := &CommentInfo{
			Content:       "I accidentally submitted the code for merging the intervals using `+` operator instead of `max()` and I am getting WA in C But the same gives AC on D in step-2 should this be happening?",
			BlogTitle:     "EDU: Segment Tree, part 1",
			CreationTime:  time.Date(2020, 7, 14, 0, 17, 0, 0, time.UTC),
			AuthorHandle:  "rds__98",
			AuthorColor:   colorClsMap["user-green"],
			AuthorAvatar:  "https://userpic.codeforces.com/593078/avatar/19f208be79c61b19.jpg",
			RevisionCount: 3,
			Revision:      3,
			Rating:        0,
			URL:           "testurl",
		}
		testParseComment(t, "blog_entry_80031.html", "661717", want)
	})
}

// TODO: Add tests for various comment contents (formattings, spoilers, images, etc)
