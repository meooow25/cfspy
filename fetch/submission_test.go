package fetch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-test/deep"
)

func testParseSubmission(t *testing.T, filename string, want *SubmissionInfo) {
	f := Fetcher{
		FetchPage: pageFetcherFor(filename, "testurl"),
	}
	got, err := f.Submission(context.Background(), "testurl")
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(got, want); diff != nil {
		t.Fatal(diff)
	}
}

func TestParseSubmission(t *testing.T) {
	t.Run("solo", func(t *testing.T) {
		want := &SubmissionInfo{
			Author: &SubmissionInfoAuthor{
				Handle: "AM.EM.U4EAC19012",
				Color:  colorClsMap["user-black"],
			},
			Problem:  "1267B",
			Language: "Python 3",
			Verdict:  "Wrong answer on test 1",
			Type:     "Practice",
			SentTime: time.Date(2019, 12, 12, 13, 23, 43, 0, time.UTC),
			URL:      "testurl",
			Content:  "x=input()\n",
		}
		testParseSubmission(t, "contest_1267_submission_66681791.html", want)
	})

	t.Run("ghost", func(t *testing.T) {
		want := &SubmissionInfo{
			AuthorGhost: "SPb ITMO: Reduce (Korobkov, Ovechkin, Poduremennykh)",
			Problem:     "1267A",
			Language:    "Unknown",
			Verdict:     "Accepted",
			Type:        "Virtual",
			SentTime:    time.Date(2019, 12, 02, 11, 25, 44, 0, time.UTC),
			URL:         "testurl",
		}
		testParseSubmission(t, "contest_1267_submission_66173991.html", want)
	})

	t.Run("team", func(t *testing.T) {
		want := &SubmissionInfo{
			AuthorTeam: &SubmissionInfoTeam{
				Name: "RednBlack Tree Team",
				Authors: []*SubmissionInfoAuthor{
					{Handle: "IZOBRETATEL777", Color: colorClsMap["user-green"]},
					{Handle: "emilprogrammist", Color: colorClsMap["user-green"]},
					{Handle: "Sadykhzadeh", Color: colorClsMap["user-cyan"]},
				},
			},
			Problem:  "1267L",
			Language: "GNU C++17",
			Verdict:  "Wrong answer on test 1",
			Type:     "Contestant",
			SentTime: time.Date(2019, 12, 01, 9, 18, 50, 0, time.UTC),
			URL:      "testurl",
			Content: `#include <bits/stdc++.h>

using namespace std;
string s;
long long n, l ,k;

int main()
{
  cin >> n >> l >> k >> s; k = 0;
  sort(s.begin(), s.end());
  for (int i = 0; i < n; i++)
  {
  	for (int j = 0; j < 3; j++, k++)
  	  cout << s[k];
  	cout << endl;
  }
}
`,
		}
		testParseSubmission(t, "contest_1267_submission_66109629.html", want)
	})

	t.Run("noTimeAndMemory", func(t *testing.T) {
		want := &SubmissionInfo{
			Author: &SubmissionInfoAuthor{
				Handle: "frodakcin",
				Color:  colorClsMap["user-red"],
			},
			Problem:  "1386A",
			Language: "GNU C++11",
			Verdict:  "Perfect result: 100 points",
			Type:     "Practice",
			SentTime: time.Date(2020, 8, 22, 6, 8, 49, 0, time.UTC),
			URL:      "testurl",
			Content: `#include <cstdio>
#include <cassert>

using ll = long long;

int T;
ll N, l, r, c;
bool pos;

bool guess(ll v)
{
	printf("? %lld\n", v);
	fflush(stdout);
	int x;
	scanf("%d", &x);
	return x;
}
void solve()
{
	scanf("%lld", &N);
	l=0, r=N;
	c=1;
	for(ll x=0;x+2<N;)
	{
		c -= x+N>>1;
		x=x+N>>1;
		assert(x+1<N);
		c += x+N>>1;
		x=x+N>>1;
	}
	guess(c);
	pos=1;
	for(;r-l>1;)
	{
		ll m=(l+r)/2;
		if(pos)
			c += m;
		else
			c -= m;
		if(guess(c))
			r=m;
		else
			l=m;
		pos ^= 1;
	}
	printf("= %lld\n", r);
	fflush(stdout);
}
int main()
{
	scanf("%d", &T);
	while(T--) solve();
	return 0;
}
`,
		}
		testParseSubmission(t, "contest_1386_submission_90658946.html", want)
	})
}

func TestParseSubmissionLocaleParamStripped(t *testing.T) {
	expected := errors.New("expected")
	f := Fetcher{
		FetchPage: func(ctx context.Context, url string) (*goquery.Document, error) {
			if strings.Contains(url, "locale") {
				t.Fatal(fmt.Errorf("locale param not stripped: %v", url))
			}
			return nil, expected
		},
	}
	_, err := f.Submission(context.Background(), "https://codeforces.com/submission?locale=test")
	if err != expected {
		t.Fatal(err)
	}
}
