package fetch

import (
	"context"
	"fmt"
	"net/http"
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
			Title: "Testing Beta Round (Unrated)",
			Content: `Hello.

Last days I did many improvements in [EDU](https://codeforces.com/edu/courses). I afraid it can affect the contest interface. So I invite you to take part in [contest:1390], just to test the system. Please, try to hacks: there were some changes in them. Please, don't expect new or interesting problems. It is just a test. Unrated.

The problems contain extremely weak pretests. Time limits are too tight. I made my best to increase number of hacks :-)

Thanks, Mike.

**UPD:** Thanks! It seems no bugs have been found.`,
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
			Title: "EDU: Segment Tree, part 1",
			Content: `Hello everyone!

I just published a [new lesson](https://codeforces.com/edu/course/2/lesson/4) in the EDU section. This is the first part of the lesson about the segment tree.

[img](https://codeforces.com/predownloaded/ad/f8/adf89a39c4a3c646e9b047bc1e24e894a01034d4.png)

In this lesson, we will learn how to build a simple segment tree (without mass modifications), and how to perform basic operations on it. We will also discuss some tasks that can be solved using the segment tree.

[Go to EDU →](https://codeforces.com/edu/courses)

More about EDU section you can read in [this](https://codeforces.com/blog/entry/79530) post.

Hope it will be helpful, enjoy!`,
			Images:       []string{"https://codeforces.com/predownloaded/ad/f8/adf89a39c4a3c646e9b047bc1e24e894a01034d4.png"},
			CreationTime: time.Date(2020, 7, 12, 10, 32, 0, 0, time.UTC),
			AuthorHandle: "pashka",
			AuthorAvatar: "fetchedavatarurl",
			AuthorColor:  colorClsMap["user-red"],
			Rating:       1309,
			URL:          "testurl",
		}
		testParseBlog(t, "blog_entry_80031.html", want)
	})

	t.Run("blogRussianLocale", func(t *testing.T) {
		want := &BlogInfo{
			Title: "Вузовско-академическая олимпиада по информатике 2021",
			Content: `[img-Спортивное программирование в УрФУ](https://codeforces.com/predownloaded/5b/78/5b78a8862b972fa67b6fd9436b32a9a01f523128.png)

Всем привет!

В этом году Уральский федеральный университет в 16-й раз проведет [Вузовско-академическую олимпиаду по информатике](https://sp.urfu.ru/vuzakadem/inform/2021/), вошедшую в 2020/21 учебном году в перечень РСОШ с III уровнем. Приглашаем школьников всех возрастов принять в ней участие!

Соревнование пройдет по правилам IOI и будет состоять из отборочного и заключительного этапов. Мы хотим ежегодно проводить качественное соревнование, в котором будет интересно участвовать крутым олимпиадникам. Посмотрите наши [задачи прошлого года](https://drive.google.com/file/d/1rvCiFgfx8eEWvHcCML0zj2sOK9eve4a-/view?usp=sharing)!

Отборочный тур пройдет онлайн с 27 февраля по 3 марта, старт виртуальный. Начать решать задачи в эти даты можно в любой момент, на решение дается 3 часа. По итогам отбора лучших участников мы пригласим в финал соревнования, который состоится во второй половине марта или начале апреля. Заключительный этап олимпиады также пройдет онлайн с применением прокторинга.

[Подайте заявку на участие!](https://acm.kontur.ru/registration/getregistrationpage?competitionid=1197be17-7db3-4e6c-818e-636499cad171)`,
			Images:       []string{"https://codeforces.com/predownloaded/5b/78/5b78a8862b972fa67b6fd9436b32a9a01f523128.png"},
			CreationTime: time.Date(2021, 2, 2, 8, 55, 0, 0, time.UTC),
			AuthorHandle: "xoposhiy",
			AuthorAvatar: "fetchedavatarurl",
			AuthorColor:  colorClsMap["user-black"],
			Rating:       72,
			URL:          "testurl",
		}
		testParseBlog(t, "blog_entry_87432.html", want)
	})
}

const commentRevFmt = `<div class="ttypography"><p>comment revision %v</p></div>`

func testParseComment(t *testing.T, filename string, commentID string, want *CommentInfo) {
	f := Fetcher{
		FetchPageWithClient: pageFetcherWithClientFor(filename, "testurl"),
		FetchCommentRevision: func(
			_ context.Context,
			gotCommentID string,
			revision int,
			_ string,
			_ *http.Client,
		) (*goquery.Document, error) {
			if gotCommentID != commentID {
				t.Fatalf("got %v, want %v", gotCommentID, commentID)
			}
			return goquery.NewDocumentFromReader(
				strings.NewReader(fmt.Sprintf(commentRevFmt, revision)))
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
