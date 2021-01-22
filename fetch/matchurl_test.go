package fetch

import (
	"testing"

	"github.com/go-test/deep"
)

func TestParseBlogURLs(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []*BlogURLMatch
	}{
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
		{"singleSpoilered", "||https://codeforces.com/blog/entry/123||",
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
				"See this suppressed link <https://codeforces.com/blog/entry/131415> and this " +
				"spoilered link ||https://codeforces.com/blog/entry/232425||.",
			[]*BlogURLMatch{
				{URL: "https://codeforces.com/blog/entry/123"},
				{URL: "https://codeforces.com/blog/entry/456"},
				{URL: "https://codeforces.com/blog/entry/789#comment-101112", CommentID: "101112"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkEqual(t, test.expected, ParseBlogURLs(test.text))
		})
	}
}

func TestParseProblemURLs(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []*ProblemURLMatch
	}{
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
		{"singleSpoilered", "||https://codeforces.com/contest/123/problem/B||",
			nil,
		},
		{"singleWithParams", "https://codeforces.com/contest/123/problem/B?locale=ru#key=value",
			[]*ProblemURLMatch{
				{URL: "https://codeforces.com/contest/123/problem/B?locale=ru#key=value"},
			},
		},
		{"multiple",
			"See https://codeforces.com/contest/123/problem/B and https://codeforces.com/gym/123456/problem/C. " +
				"See this suppressed link <https://codeforces.com/problemsets/acmsguru/problem/99999/123> and " +
				"this spoilered link ||https://codeforces.com/problemsets/acmsguru/problem/99999/456||.",
			[]*ProblemURLMatch{
				{URL: "https://codeforces.com/contest/123/problem/B"},
				{URL: "https://codeforces.com/gym/123456/problem/C"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkEqual(t, test.expected, ParseProblemURLs(test.text))
		})
	}
}

func TestParseSubmissionURLs(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []*SubmissionURLMatch
	}{
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
		{"singleSpoilered", "||https://codeforces.com/contest/123/submission/123456||",
			nil,
		},
		{"singleWithParams", "https://codeforces.com/contest/123/submission/123456?locale=ru#key=value",
			[]*SubmissionURLMatch{
				{URL: "https://codeforces.com/contest/123/submission/123456?locale=ru#key=value"},
			},
		},
		{"multiple",
			"See https://codeforces.com/contest/123/submission/123456 and <https://codeforces.com/gym/123456/submission/54321> " +
				"and ||https://codeforces.com/gym/123456/submission/54321||. ",
			[]*SubmissionURLMatch{
				{URL: "https://codeforces.com/contest/123/submission/123456"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkEqual(t, test.expected, ParseSubmissionURLs(test.text))
		})
	}
}

func TestParseProfileURLs(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []*ProfileURLMatch
	}{
		{"helloWorld", "Hello, world!", nil},
		{"homePage", "https://codeforces.com/", nil},
		{"single", "https://codeforces.com/profile/handle1",
			[]*ProfileURLMatch{
				{URL: "https://codeforces.com/profile/handle1"},
			},
		},
		{"singleWithText", "Visit https://codeforces.com/profile/handle2.",
			[]*ProfileURLMatch{
				{URL: "https://codeforces.com/profile/handle2"},
			},
		},
		{"singleSuppressed", "<https://codeforces.com/profile/handle3>",
			nil,
		},
		{"singleSpoilered", "||https://codeforces.com/profile/handle4||",
			nil,
		},
		{"singleWithParams", "https://codeforces.com/profile/handle5?locale=ru#key=value",
			[]*ProfileURLMatch{
				{URL: "https://codeforces.com/profile/handle5?locale=ru#key=value"},
			},
		},
		{"multiple",
			"See https://codeforces.com/profile/handle6 and https://codeforces.com/profile/handle7. " +
				"See this suppressed link <https://codeforces.com/profile/handle8> and " +
				"this spoilered link ||https://codeforces.com/profile/handle9||.",
			[]*ProfileURLMatch{
				{URL: "https://codeforces.com/profile/handle6"},
				{URL: "https://codeforces.com/profile/handle7"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkEqual(t, test.expected, ParseProfileURLs(test.text))
		})
	}
}

func TestRemoveSpoilers(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"no spoiler", "no spoiler"},
		{"||some spoiler||", ""},
		{"partial ||spoiler|| text", "partial  text"},
		{"a || b || c || d || e", "a  c  e"},
		{"complete ||spoiler|| incomplete ||spoiler", "complete  incomplete ||spoiler"},
		{"!||s|p|o|i|l|e|r||!", "!!"},
	}
	for _, test := range tests {
		checkEqual(t, test.want, removeSpoilers(test.input))
	}
}

func checkEqual(t *testing.T, expected, got interface{}) {
	if diff := deep.Equal(expected, got); diff != nil {
		t.Fatal(diff)
	}
}
