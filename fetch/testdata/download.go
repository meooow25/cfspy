package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Utility to download, clean, and save a submission page
func main() {
	subURL := flag.String("url", "", "submission URL")
	flag.Parse()

	// https://codeforces.com/contest/<cid>/submission/<sid>
	pieces := strings.Split(*subURL, "/")
	contestID := pieces[4]
	submissionID := pieces[6]

	resp, err := http.Get(*subURL)
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

	ioutil.WriteFile(contestID+"_"+submissionID+".html", []byte(html), 0644)
}
