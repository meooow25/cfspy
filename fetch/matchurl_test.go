package fetch

import (
	"testing"

	"github.com/go-test/deep"
)

type blogTest struct {
	name     string
	text     string
	expected []*BlogURLMatch
}

type problemTest struct {
	name     string
	text     string
	expected []*ProblemURLMatch
}

type submissionTest struct {
	name     string
	text     string
	expected []*SubmissionURLMatch
}

var blogTests = []blogTest{
	{"helloWorld", "Hello, world!", nil},
	{"homePage", "https://codeforces.com/", nil},
	{"single", "https://codeforces.com/blog/entry/123",
		[]*BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123"},
		},
	},
	{"singleWithText", "Visit https://codeforces.com/blog/entry/123.",
		[]*BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123"},
		},
	},
	{"singleSuppressed", "<https://codeforces.com/blog/entry/123>",
		nil,
	},
	{"singleWithParams", "https://codeforces.com/blog/entry/123?locale=ru#comment-456&key=value",
		[]*BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123?locale=ru#comment-456&key=value", CommentID: "456"},
		},
	},
	{"multiple",
		"See https://codeforces.com/blog/entry/123 and https://codeforces.com/blog/entry/456. " +
			"Also see https://codeforces.com/blog/entry/789#comment-101112 this comment.\n" +
			"See this suppressed link <https://codeforces.com/blog/entry/131415>.",
		[]*BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123"},
			{URL: "https://codeforces.com/blog/entry/456"},
			{URL: "https://codeforces.com/blog/entry/789#comment-101112", CommentID: "101112"},
		},
	},
}

var problemTests = []problemTest{
	{"helloWorld", "Hello, world!", nil},
	{"homePage", "https://codeforces.com/", nil},
	{"singleContest", "https://codeforces.com/contest/123/problem/B",
		[]*ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B"},
		},
	},
	{"singleGym", "https://codeforces.com/gym/123456/problem/C",
		[]*ProblemURLMatch{
			{URL: "https://codeforces.com/gym/123456/problem/C"},
		},
	},
	{"singleAcmsguru", "https://codeforces.com/problemsets/acmsguru/problem/99999/123",
		[]*ProblemURLMatch{
			{URL: "https://codeforces.com/problemsets/acmsguru/problem/99999/123"},
		},
	},
	{"singleWithText", "Visit https://codeforces.com/contest/123/problem/B.",
		[]*ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B"},
		},
	},
	{"singleSuppressed", "<https://codeforces.com/contest/123/problem/B>",
		nil,
	},
	{"singleWithParams", "https://codeforces.com/contest/123/problem/B?locale=ru#key=value",
		[]*ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B?locale=ru#key=value"},
		},
	},
	{"multiple",
		"See https://codeforces.com/contest/123/problem/B and https://codeforces.com/gym/123456/problem/C. " +
			"See this suppressed link <https://codeforces.com/problemsets/acmsguru/problem/99999/123>.",
		[]*ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B"},
			{URL: "https://codeforces.com/gym/123456/problem/C"},
		},
	},
}

var submissionTests = []submissionTest{
	{"helloWorld", "Hello, world!", nil},
	{"homePage", "https://codeforces.com/", nil},
	{"singleSubmission", "https://codeforces.com/contest/123/submission/123456",
		[]*SubmissionURLMatch{
			{URL: "https://codeforces.com/contest/123/submission/123456"},
		},
	},
	{"singleGym", "https://codeforces.com/gym/123456/submission/54321",
		[]*SubmissionURLMatch{
			{URL: "https://codeforces.com/gym/123456/submission/54321"},
		},
	},
	{"singleWithText", "Visit https://codeforces.com/contest/123/submission/123456.",
		[]*SubmissionURLMatch{
			{URL: "https://codeforces.com/contest/123/submission/123456"},
		},
	},
	{"singleSuppressed", "<https://codeforces.com/contest/123/submission/123456>",
		nil,
	},
	{"singleWithParams", "https://codeforces.com/contest/123/submission/123456?locale=ru#key=value",
		[]*SubmissionURLMatch{
			{URL: "https://codeforces.com/contest/123/submission/123456?locale=ru#key=value"},
		},
	},
	{"multiple",
		"See https://codeforces.com/contest/123/submission/123456 and <https://codeforces.com/gym/123456/submission/54321>. ",
		[]*SubmissionURLMatch{
			{URL: "https://codeforces.com/contest/123/submission/123456"},
		},
	},
}

func TestParseBlogURLs(t *testing.T) {
	for _, test := range blogTests {
		t.Run(test.name, func(t *testing.T) {
			checkEqual(t, test.expected, ParseBlogURLs(test.text))
		})
	}
}

func TestParseProblemURLs(t *testing.T) {
	for _, test := range problemTests {
		t.Run(test.name, func(t *testing.T) {
			checkEqual(t, test.expected, ParseProblemURLs(test.text))
		})
	}
}

func TestParseSubmissionURLs(t *testing.T) {
	for _, test := range submissionTests {
		t.Run(test.name, func(t *testing.T) {
			checkEqual(t, test.expected, ParseSubmissionURLs(test.text))
		})
	}
}

func checkEqual(t *testing.T, expected, got interface{}) {
	if diff := deep.Equal(expected, got); diff != nil {
		t.Fatal(diff)
	}
}
