package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/go-test/deep"
	"github.com/meooow25/cfspy/fetch"
)

var (
	testTime   = time.Now()
	testAuthor = &fetch.SubmissionInfoAuthor{
		Handle: "author",
		Color:  0x123456,
	}
	testAuthorTeam = &fetch.SubmissionInfoTeam{
		Name: "worst team ever",
		Authors: []*fetch.SubmissionInfoAuthor{
			{Handle: "member1", Color: 0x010203},
			{Handle: "member2", Color: 0x030201},
		},
	}
	testContent         string
	testContentLongLine = strings.Repeat("a", 5000)
)

func init() {
	var contentBuilder strings.Builder
	for i := 0; i < 200; i++ {
		contentBuilder.WriteString(fmt.Sprintf("Line %v\n", i+1))
	}
	contentBuilder.WriteString("\n") // 200th line empty
	for i := 200; i < 400; i++ {
		contentBuilder.WriteString(fmt.Sprintf("Line %v\n", i+1))
	}
	testContent = contentBuilder.String()
}

func newSubmissionInfo(opts ...func(*fetch.SubmissionInfo)) *fetch.SubmissionInfo {
	// defaults that can be overriden: author, language, verdict, content
	info := &fetch.SubmissionInfo{
		ID:              "998244353",
		Author:          testAuthor,
		Problem:         "4321Z",
		Language:        "Go",
		Verdict:         "Verdict",
		ParticipantType: "Contestant",
		SentTime:        testTime,
		Content:         testContent,
		URL:             "testUrl",
	}
	for _, opt := range opts {
		opt(info)
	}
	return info
}

func verdict(v string) func(*fetch.SubmissionInfo) {
	return func(info *fetch.SubmissionInfo) {
		info.Verdict = v
	}
}

func authorTeam(a *fetch.SubmissionInfoTeam) func(*fetch.SubmissionInfo) {
	return func(info *fetch.SubmissionInfo) {
		info.Author = nil
		info.AuthorTeam = a
		info.AuthorGhost = ""
	}
}

func authorGhost(a string) func(*fetch.SubmissionInfo) {
	return func(info *fetch.SubmissionInfo) {
		info.Author = nil
		info.AuthorTeam = nil
		info.AuthorGhost = a
	}
}

func language(l string) func(*fetch.SubmissionInfo) {
	return func(info *fetch.SubmissionInfo) {
		info.Language = l
	}
}

func content(c string) func(*fetch.SubmissionInfo) {
	return func(info *fetch.SubmissionInfo) {
		info.Content = c
	}
}

func newWantEmbed(title, description string, color int) *disgord.Embed {
	return &disgord.Embed{
		Title:       title,
		Description: description,
		URL:         "testUrl",
		Timestamp:   disgord.Time{Time: testTime},
		Color:       color,
	}
}

func testContentLines(start, end int) string {
	return strings.Join(strings.Split(testContent, "\n")[start-1:end], "\n")
}

func TestMakeSubmissionResponse(t *testing.T) {
	type file struct{ name, content string }
	tests := []struct {
		name string

		info      *fetch.SubmissionInfo
		lineBegin int
		lineEnd   int

		wantContent string
		wantEmbed   *disgord.Embed
		wantFile    *file
		wantErr     error
	}{
		{
			name: "summaryAccepted",
			info: newSubmissionInfo(verdict("Accepted")),
			wantEmbed: newWantEmbed(
				"Submission for 4321Z by author",
				"✅ Accepted • Contestant • Go",
				testAuthor.Color),
		},
		{
			name: "summaryPerfectResult",
			info: newSubmissionInfo(verdict("Perfect result: 100 points")),
			wantEmbed: newWantEmbed(
				"Submission for 4321Z by author",
				"✅ Perfect result: 100 points • Contestant • Go",
				testAuthor.Color,
			),
		},
		{
			name: "summaryNotAccepted",
			info: newSubmissionInfo(verdict("Any verdict excepted accepted")),
			wantEmbed: newWantEmbed(
				"Submission for 4321Z by author",
				"Any verdict excepted accepted • Contestant • Go",
				testAuthor.Color,
			),
		},
		{
			name: "summaryAuthorTeam",
			info: newSubmissionInfo(authorTeam(testAuthorTeam)),
			wantEmbed: newWantEmbed(
				"Submission for 4321Z by worst team ever: member1, member2",
				"Verdict • Contestant • Go",
				teamColor,
			),
		},
		{
			name: "summaryAuthorGhost",
			info: newSubmissionInfo(authorGhost("wooooo"), language("Unknown")),
			wantEmbed: newWantEmbed(
				"Submission for 4321Z by wooooo 👻",
				"Verdict • Contestant • Unknown language",
				ghostColor,
			),
		},
		{
			name:        "snippetShort",
			info:        newSubmissionInfo(),
			lineBegin:   12,
			lineEnd:     15,
			wantContent: "```go\n" + testContentLines(12, 15) + "```",
		},
		{
			name:        "snippetShortLineNumsClamped",
			info:        newSubmissionInfo(content("code")),
			lineBegin:   -100,
			lineEnd:     100,
			wantContent: "```go\ncode```",
		},
		{
			name:        "snippetShortNoExtMapped",
			info:        newSubmissionInfo(language("HolyC")),
			lineBegin:   12,
			lineEnd:     15,
			wantContent: "```\n" + testContentLines(12, 15) + "```",
		},
		{
			name:        "snippetShortMaxLines",
			info:        newSubmissionInfo(),
			lineBegin:   12,
			lineEnd:     12 + maxSnippetMsgLines - 1,
			wantContent: "```go\n" + testContentLines(12, 12+maxSnippetMsgLines-1) + "```",
		},
		{
			name:      "snippetLongMinLines",
			info:      newSubmissionInfo(),
			lineBegin: 12,
			lineEnd:   12 + maxSnippetMsgLines,
			wantFile: &file{
				name:    "snippet_998244353.go",
				content: testContentLines(12, 12+maxSnippetMsgLines),
			},
		},
		{
			name:      "snippetLong",
			info:      newSubmissionInfo(),
			lineBegin: 12,
			lineEnd:   365,
			wantFile: &file{
				name:    "snippet_998244353.go",
				content: testContentLines(12, 365),
			},
		},
		{
			name:      "snippetLongNoExtMapped",
			info:      newSubmissionInfo(language("HolyC")),
			lineBegin: 12,
			lineEnd:   365,
			wantFile: &file{
				name:    "snippet_998244353.txt",
				content: testContentLines(12, 365),
			},
		},
		{
			name:      "snippetLongOneLine",
			info:      newSubmissionInfo(content(testContentLongLine)),
			lineBegin: 1,
			lineEnd:   1,
			wantFile: &file{
				name:    "snippet_998244353.go",
				content: testContentLongLine,
			},
		},
		{
			name:      "snippetEmpty",
			info:      newSubmissionInfo(),
			lineBegin: 201,
			lineEnd:   201,
			wantErr:   errSelectionEmpty,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content, embed, file, err :=
				makeSubmissionResponse(test.info, test.lineBegin, test.lineEnd)

			eq := func(a, b interface{}) {
				if diff := deep.Equal(a, b); diff != nil {
					t.Fatal(diff)
				}
			}

			eq(err, test.wantErr)
			eq(content, test.wantContent)
			eq(embed, test.wantEmbed)
			if test.wantFile != nil {
				eq(file.FileName, test.wantFile.name)
				fileContent, err := ioutil.ReadAll(file.Reader)
				if err != nil {
					t.Fatal(err)
				}
				eq(string(fileContent), test.wantFile.content)
			} else if file != nil {
				t.Fatal("file not expected")
			}
		})
	}
}
