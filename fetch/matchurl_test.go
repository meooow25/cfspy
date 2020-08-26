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
	{"Hello, world!", "Hello, world!", nil},
	{"Home page", "https://codeforces.com/", nil},
	{"Single", "https://codeforces.com/blog/entry/123",
		[]BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123", BlogID: "123"},
		},
	},
	{"Single with text", "Visit https://codeforces.com/blog/entry/123.",
		[]BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123", BlogID: "123"},
		},
	},
	{"Single suppressed", "<https://codeforces.com/blog/entry/123>",
		[]BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123", BlogID: "123", Suppressed: true},
		},
	},
	{"Single with params", "https://codeforces.com/blog/entry/123?locale=ru#comment-456&key=value",
		[]BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123?locale=ru#comment-456&key=value", BlogID: "123", CommentID: "456"},
		},
	},
	{"Multiple",
		"See https://codeforces.com/blog/entry/123 and https://codeforces.com/blog/entry/456. " +
			"Also see https://codeforces.com/blog/entry/789#comment-101112 this comment.\n" +
			"See this suppressed link <https://codeforces.com/blog/entry/131415>.",
		[]BlogURLMatch{
			{URL: "https://codeforces.com/blog/entry/123", BlogID: "123"},
			{URL: "https://codeforces.com/blog/entry/456", BlogID: "456"},
			{URL: "https://codeforces.com/blog/entry/789#comment-101112", BlogID: "789", CommentID: "101112"},
			{URL: "https://codeforces.com/blog/entry/131415", BlogID: "131415", Suppressed: true},
		},
	},
}

var problemTests = []problemTest{
	{"Hello, world!", "Hello, world!", nil},
	{"Home page", "https://codeforces.com/", nil},
	{"Single contest", "https://codeforces.com/contest/123/problem/B",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B", ContestID: "123", ProblemID: "B"},
		},
	},
	{"Single gym", "https://codeforces.com/gym/123456/problem/C",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/gym/123456/problem/C", ContestID: "123456", ProblemID: "C"},
		},
	},
	{"Single acmsguru", "https://codeforces.com/problemsets/acmsguru/problem/99999/123",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/problemsets/acmsguru/problem/99999/123", ContestID: "99999", ProblemID: "123"},
		},
	},
	{"Single with text", "Visit https://codeforces.com/contest/123/problem/B.",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B", ContestID: "123", ProblemID: "B"},
		},
	},
	{"Single suppressed", "<https://codeforces.com/contest/123/problem/B>",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B", ContestID: "123", ProblemID: "B", Suppressed: true},
		},
	},
	{"Single with params", "https://codeforces.com/contest/123/problem/B?locale=ru#key=value",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B?locale=ru#key=value", ContestID: "123", ProblemID: "B"},
		},
	},
	{"Multiple",
		"See https://codeforces.com/contest/123/problem/B and https://codeforces.com/gym/123456/problem/C. " +
			"See this suppressed link <https://codeforces.com/problemsets/acmsguru/problem/99999/123>.",
		[]ProblemURLMatch{
			{URL: "https://codeforces.com/contest/123/problem/B", ContestID: "123", ProblemID: "B"},
			{URL: "https://codeforces.com/gym/123456/problem/C", ContestID: "123456", ProblemID: "C"},
			{URL: "https://codeforces.com/problemsets/acmsguru/problem/99999/123", ContestID: "99999", ProblemID: "123", Suppressed: true},
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
