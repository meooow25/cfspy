package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	urlpkg "net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Utility to download, clean, and save a Codeforces page
func main() {
	url := flag.String("url", "", "Codeforces URL")
	flag.Parse()

	resp, err := http.Get(*url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Remove the insane amount of inline scripts, plus other unnecessary elements
	doc.Find("script").Remove()
	doc.Find("style").Remove()
	doc.Find("link").Remove()
	doc.Find("meta").Remove()

	html, err := goquery.OuterHtml(doc.Selection)
	if err != nil {
		log.Fatal(err)
	}

	// Remove blank lines
	lines := strings.Split(html, "\n")
	i := 0
	insideSource := false
	for _, line := range lines {
		if strings.Contains(line, "program-source-text") {
			insideSource = true
		}
		if insideSource || strings.TrimSpace(line) != "" {
			lines[i] = line
			i++
		}
		if insideSource && strings.Contains(line, "</pre>") {
			insideSource = false
		}
	}
	html = strings.Join(lines[:i], "\n")

	u, _ := urlpkg.Parse(*url)
	filename := strings.ReplaceAll(u.Path[1:], "/", "_") + ".html"

	ioutil.WriteFile(filename, []byte(html), 0644)
}
