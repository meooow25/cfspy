package fetch

import (
	"os"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-test/deep"
)

func testParseSubmission(t *testing.T, filename string, want *SubmissionInfo) {
	f, err := os.Open("testdata/" + filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseSubmission("testurl", doc)
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
		testParseSubmission(t, "1267_66681791.html", want)
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
		testParseSubmission(t, "1267_66173991.html", want)
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
		testParseSubmission(t, "1267_66109629.html", want)
	})
}
