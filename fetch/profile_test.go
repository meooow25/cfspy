package fetch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-test/deep"
)

func testParseProfile(t *testing.T, filename string, want *ProfileInfo) {
	f := Fetcher{
		FetchPage: pageFetcherFor(filename, "testurl"),
	}
	got, err := f.Profile(context.Background(), "testurl")
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(got, want); diff != nil {
		t.Fatal(diff)
	}
}

func TestParseProfile(t *testing.T) {
	t.Run("rated", func(t *testing.T) {
		want := &ProfileInfo{
			Handle:    "rainboy",
			Rating:    2042,
			MaxRating: 2396,
			Rank:      "Candidate Master",
			Color:     colorClsMap["user-violet"],
			Avatar:    "https://userpic.codeforces.com/408096/title/2ebfef1265c2f1e6.jpg",
			URL:       "testurl",
		}
		testParseProfile(t, "profile_rainboy.html", want)
	})
	t.Run("unrated", func(t *testing.T) {
		want := &ProfileInfo{
			Handle:    "LanceTheDragonTrainer",
			Rating:    0,
			MaxRating: 0,
			Rank:      "Unrated",
			Color:     colorClsMap["user-black"],
			Avatar:    "https://userpic.codeforces.com/no-title.jpg",
			URL:       "testurl",
		}
		testParseProfile(t, "profile_LanceTheDragonTrainer.html", want)
	})
	t.Run("hqRated", func(t *testing.T) {
		want := &ProfileInfo{
			Handle:    "geranazavr555",
			Rating:    1772,
			MaxRating: 1864,
			Rank:      "Headquarters",
			Color:     colorClsMap["user-admin"],
			Avatar:    "https://userpic.codeforces.com/226065/title/c40b38db239bbdab.jpg",
			URL:       "testurl",
		}
		testParseProfile(t, "profile_geranazavr555.html", want)
	})
	t.Run("hqUnrated", func(t *testing.T) {
		want := &ProfileInfo{
			Handle:    "MikeMirzayanov",
			Rating:    0,
			MaxRating: 0,
			Rank:      "Headquarters",
			Color:     colorClsMap["user-admin"],
			Avatar:    "https://userpic.codeforces.com/11/title/c7fb4051127c29e4.jpg",
			URL:       "testurl",
		}
		testParseProfile(t, "profile_MikeMirzayanov.html", want)
	})
}

func TestParseProfileLocaleParamStripped(t *testing.T) {
	expected := errors.New("expected")
	f := Fetcher{
		FetchPage: func(ctx context.Context, url string) (*goquery.Document, error) {
			if strings.Contains(url, "locale") {
				t.Fatal(fmt.Errorf("locale param not stripped: %v", url))
			}
			return nil, expected
		},
	}
	_, err := f.Profile(context.Background(), "https://codeforces.com/profile?locale=test")
	if err != expected {
		t.Fatal(err)
	}
}
