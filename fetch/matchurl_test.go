package fetch

import (
	"reflect"
	"testing"
)

type blogTest struct {
	name     string
	text     string
	expected []BlogURLMatch
}

type problemTest struct {
	name     string
	text     string
	expected []ProblemURLMatch
}

var blogTests = []blogTest{
	{"helloWorld", "Hello, world!", nil},
	{"homePage", "https://codeforces.com/", nil},
	{"single", "https://codeforces.com/blog/entry/123",
		[]BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123"},
		},
	},
	{"singleWithText", "Visit https://codeforces.com/blog/entry/123.",
		[]BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123"},
		},
	},
	{"singleSuppressed", "<https://codeforces.com/blog/entry/123>",
		[]BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123", Suppressed: true},
		},
	},
	{"singleWithParams", "https://codeforces.com/blog/entry/123?locale=ru#comment-456&key=value",
		[]BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123?locale=ru#comment-456&key=value", CommentID: "456"},
		},
	},
	{"multiple",
		"See https://codeforces.com/blog/entry/123 and https://codeforces.com/blog/entry/456. " +
			"Also see https://codeforces.com/blog/entry/789#comment-101112 this comment.\n" +
			"See this suppressed link <https://codeforces.com/blog/entry/131415>.",
		[]BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123"},
			{URL: "https://codeforces.com/blog/entry/456"},
			{URL: "https://codeforces.com/blog/entry/789#comment-101112", CommentID: "101112"},
			{URL: "https://codeforces.com/blog/entry/131415", Suppressed: true},
		},
	},
}

var problemTests = []problemTest{
	{"helloWorld", "Hello, world!", nil},
	{"homePage", "https://codeforces.com/", nil},
	{"singleContest", "https://codeforces.com/contest/123/problem/B",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B"},
		},
	},
	{"singleGym", "https://codeforces.com/gym/123456/problem/C",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/gym/123456/problem/C"},
		},
	},
	{"singleAcmsguru", "https://codeforces.com/problemsets/acmsguru/problem/99999/123",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/problemsets/acmsguru/problem/99999/123"},
		},
	},
	{"singleWithText", "Visit https://codeforces.com/contest/123/problem/B.",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B"},
		},
	},
	{"singleSuppressed", "<https://codeforces.com/contest/123/problem/B>",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B", Suppressed: true},
		},
	},
	{"singleWithParams", "https://codeforces.com/contest/123/problem/B?locale=ru#key=value",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B?locale=ru#key=value"},
		},
	},
	{"multiple",
		"See https://codeforces.com/contest/123/problem/B and https://codeforces.com/gym/123456/problem/C. " +
			"See this suppressed link <https://codeforces.com/problemsets/acmsguru/problem/99999/123>.",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B"},
			{URL: "https://codeforces.com/gym/123456/problem/C"},
			{URL: "https://codeforces.com/problemsets/acmsguru/problem/99999/123", Suppressed: true},
		},
	},
}

func TestParseBlogURLs(t *testing.T) {
	for _, test := range blogTests {
		t.Run(test.name, func(t *testing.T) {
			m := ParseBlogURLs(test.text)
			if !reflect.DeepEqual(m, test.expected) {
				t.Errorf("Expected %v, found %v", test.expected, m)
			}
		})
	}
}

func TestParseProblemURLs(t *testing.T) {
	for _, test := range problemTests {
		t.Run(test.name, func(t *testing.T) {
			m := ParseProblemURLs(test.text)
			if !reflect.DeepEqual(m, test.expected) {
				t.Errorf("Expected %v, found %v", test.expected, m)
			}
		})
	}
}
