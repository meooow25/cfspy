package fetch

import (
	"context"
	"testing"

	"github.com/go-test/deep"
)

func testParseProblem(t *testing.T, filename string, want *ProblemInfo) {
	f := Fetcher{
		FetchPage: pageFetcherFor(filename, "testurl"),
	}
	got, err := f.Problem(context.Background(), "testurl")
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(got, want); diff != nil {
		t.Fatal(diff)
	}
}

func TestParseProblem(t *testing.T) {
	t.Run("finished", func(t *testing.T) {
		want := &ProblemInfo{
			Name:          "A. Avoid Trygub",
			ContestName:   "Codeforces Global Round 12",
			ContestStatus: "Finished",
			URL:           "testurl",
		}
		testParseProblem(t, "contest_1450_problem_A.html", want)
	})

	t.Run("running", func(t *testing.T) {
		want := &ProblemInfo{
			Name:          "A. String Generation",
			ContestName:   "Codeforces Round #689 (Div. 2, based on Zed Code Competition)",
			ContestStatus: "Contest is running",
			URL:           "testurl",
		}
		testParseProblem(t, "contest_1461_problem_A.html", want)
	})

	t.Run("acmsguruOld", func(t *testing.T) {
		want := &ProblemInfo{
			Name:          "100. A+B",
			ContestName:   "acmsguru",
			ContestStatus: "Finished",
			URL:           "testurl",
		}
		testParseProblem(t, "problemsets_acmsguru_problem_99999_100.html", want)
	})

	t.Run("acmsguruNew", func(t *testing.T) {
		want := &ProblemInfo{
			Name:          "553. Sultan's Pearls",
			ContestName:   "acmsguru",
			ContestStatus: "Finished",
			URL:           "testurl",
		}
		testParseProblem(t, "problemsets_acmsguru_problem_99999_553.html", want)
	})
}
