package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
	"github.com/meooow25/cfspy/fetch"
)

var languageNameToExt = map[string]string{
	// Currently available in the filter options on the status page of contests.
	"GNU C11":               "c",
	"Clang++17 Diagnostics": "cpp",
	"GNU C++11":             "cpp",
	"GNU C++14":             "cpp",
	"GNU C++17":             "cpp",
	"MS C++":                "cpp",
	"MS C++ 2017":           "cpp",
	"GNU C++17 (64)":        "cpp",
	"Mono C#":               "cs",
	"D":                     "d",
	"Go":                    "go",
	"Haskell":               "hs",
	"Java 11":               "java",
	"Java 8":                "java",
	"Kotlin":                "kt",
	"Ocaml":                 "ml",
	"Delphi":                "pas",
	"FPC":                   "pas",
	"PascalABC.NET":         "pas",
	"Perl":                  "pl",
	"Python 2":              "py",
	"Python 3":              "py",
	"Pypy 2":                "py",
	"PyPy 3":                "py",
	"Ruby":                  "rb",
	"Rust":                  "rs",
	"Scala":                 "sc",
	"JavaScript":            "js",
	"Node.js":               "js",

	// Some old ones. I have no way of getting an exhaustive list of these removed names.
	"GNU C":     "c",
	"GNU C++":   "cpp",
	"GNU C++0x": "cpp",
	"Java 6":    "java",
	"Java 7":    "java",
}

// The number of lines beyond which the snippet is sent as a file instead of message text.
// This is because Discord's new file previews are nice and collapsible, taking up less space
// compared to a wall of text in the message body.
// The value below is chosen because it seems reasonable to me.
const maxSnippetMsgLines = 30

// Installs the submission watcher feature. The bot watches for Codeforces submission links and
// responds with an embed containing info about the submission. If the submission has line numbers,
// responds with the lines.
func installSubmissionFeature(bot *bot.Bot) {
	bot.Client.Logger().Info("Setting up CF submission feature")
	bot.OnMessageCreate(maybeHandleSubmissionURL)
}

func maybeHandleSubmissionURL(ctx *bot.Context, evt *disgord.MessageCreate) {
	go func() {
		submissionURLMatches := fetch.ParseSubmissionURLs(evt.Message.Content)
		if len(submissionURLMatches) == 0 {
			return
		}
		first := submissionURLMatches[0]
		handleSubmissionURL(ctx, first)
	}()
}

// Fetches the submission page and responds on the Discord channel.
func handleSubmissionURL(ctx *bot.Context, match *fetch.SubmissionURLMatch) {
	ctx.Logger.Info("Processing submission URL: ", match.URL)

	submissionInfo, err := fetch.Submission(context.Background(), match.URL)
	if err != nil {
		err = fmt.Errorf("Error fetching submission from %v: %w", match.URL, err)
		ctx.Logger.Error(err)
		respondWithError(ctx, err)
		return
	}

	var content string
	var embed *disgord.Embed
	var file *disgord.CreateMessageFileParams
	if match.LineBegin == 0 {
		embed = makeSubmissionEmbed(submissionInfo, ctx.Logger)
	} else {
		snippet, numLines, err := makeCodeSnippet(
			submissionInfo.Content, match.LineBegin, match.LineEnd)
		if err != nil {
			respondWithError(ctx, err)
			return
		}
		tryContent := makeContent(snippet, submissionInfo.Language)
		if numLines <= maxSnippetMsgLines && !bot.ContentTooLong(tryContent) {
			content = tryContent
		} else {
			// The file size is never expected to be too large as Codeforces source limit is 64KB
			// and Discord limit is 8MB.
			reader, filename := makeReaderAndFilename(
				snippet, submissionInfo.Language, submissionInfo.ID)
			file = &disgord.CreateMessageFileParams{Reader: reader, FileName: filename}
		}
	}

	page := bot.NewPage(content, embed)
	if file != nil {
		err = respondWithOnePagePreview(ctx, page, *file)
	} else {
		err = respondWithOnePagePreview(ctx, page)
	}
	if err != nil {
		ctx.Logger.Error(fmt.Errorf("Error sending problem info: %w", err))
	}
}

func makeSubmissionEmbed(s *fetch.SubmissionInfo, logger disgord.Logger) *disgord.Embed {
	prefix := ""
	if s.Verdict == "Accepted" || strings.HasPrefix(s.Verdict, "Perfect result") {
		prefix = "✅ "
	}
	var author string
	var color int
	switch {
	case s.Author != nil:
		author = s.Author.Handle
		color = s.Author.Color
	case s.AuthorGhost != "":
		author = s.AuthorGhost + " 👻"
		color = 0x999999 // Same color as text on CF
	case s.AuthorTeam != nil:
		var handles []string
		for _, author := range s.AuthorTeam.Authors {
			handles = append(handles, author.Handle)
		}
		author = s.AuthorTeam.Name + ": " + strings.Join(handles, ", ")
		color = 0x666666 // Darker than ghosts
	default:
		logger.Error("No author details in submission info: ", s.URL)
	}
	language := s.Language
	if language == "Unknown" { // Happens for ghosts
		language = "Unknown language"
	}
	return &disgord.Embed{
		Title:       "Submission for " + s.Problem + " by " + author,
		URL:         s.URL,
		Color:       color,
		Description: prefix + s.Verdict + " • " + s.ParticipantType + " • " + language,
		Timestamp:   disgord.Time{Time: s.SentTime},
	}
}

func makeCodeSnippet(code string, begin, end int) (snippet string, numLines int, err error) {
	code = strings.ReplaceAll(code, "\r\n", "\n")
	lines := strings.Split(code, "\n")
	begin, end = clamp(begin, 1, len(lines)), clamp(end, 1, len(lines))
	lines = lines[begin-1 : end]

	// This might not look nice if there are mixed spaces and tabs.
	// But if you write such code, you deserve it.
	minSpaceCount := math.MaxInt32
	for _, line := range lines {
		for i, c := range line {
			if c != ' ' && c != '\t' {
				if i < minSpaceCount {
					minSpaceCount = i
				}
				break
			}
		}
	}
	allEmpty := true
	for i := range lines {
		if len(lines[i]) > minSpaceCount {
			lines[i] = lines[i][minSpaceCount:]
			allEmpty = false
		} else {
			lines[i] = ""
		}
	}
	if allEmpty {
		return "", 0, errors.New("Selected lines are empty")
	}

	return strings.Join(lines, "\n"), len(lines), nil
}

func makeContent(snippet, language string) string {
	return "```" + languageNameToExt[language] + "\n" + snippet + "```"
}

func makeReaderAndFilename(snippet, language, id string) (*strings.Reader, string) {
	var ext string
	if ext = languageNameToExt[language]; ext == "" {
		ext = "txt"
	}
	filename := id + "_snippet." + ext
	return strings.NewReader(snippet), filename
}

func clamp(x, low, high int) int {
	if x < low {
		x = low
	} else if x > high {
		x = high
	}
	return x
}
