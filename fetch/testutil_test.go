package fetch

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/PuerkitoBio/goquery"
)

func loadHtmlTestFile(filename string) (*goquery.Document, error) {
	f, err := os.Open("testdata/" + filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return goquery.NewDocumentFromReader(f)
}

func pageFetcherFor(filename string, wantURL string) func(context.Context, string) (*goquery.Document, error) {
	return func(_ context.Context, url string) (*goquery.Document, error) {
		if url != wantURL {
			return nil, fmt.Errorf("got %v, want %v", url, wantURL)
		}
		return loadHtmlTestFile(filename)
	}
}

func pageFetcherWithClientFor(filename string, wantURL string) func(context.Context, string, *http.Client) (*goquery.Document, error) {
	return func(_ context.Context, url string, _ *http.Client) (*goquery.Document, error) {
		if url != wantURL {
			return nil, fmt.Errorf("got %v, want %v", url, wantURL)
		}
		return loadHtmlTestFile(filename)
	}
}
