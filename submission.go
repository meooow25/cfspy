package main

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
	"github.com/meooow25/cfspy/fetch"
)

var languageNameToCode = map[string]string{
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
	"Perl":                  "perl",
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

// Installs the submission watcher feature. The bot watches for Codeforces submission links and
// responds with an embed containing info about the submission. If the submission has line numbers,
// responds with the lines.
func installSubmissionFeature(bot *bot.Bot) {
	bot.Client.Logger().Info("Setting up CF submission feature")
	bot.OnMessageCreate(maybeHandleSubmissionURL)
}

func maybeHandleSubmissionURL(ctx bot.Context, evt *disgord.MessageCreate) {
	if evt.Message.Author.Bot {
		return
	}
	go func() {
		submissionURLMatches := fetch.ParseSubmissionURLs(evt.Message.Content)
		if len(submissionURLMatches) == 0 {
			return
		}
		first := submissionURLMatches[0]
		if first.Suppressed {
			return
		}
		handleSubmissionURL(ctx, &first)
	}()
}

// Fetches the submission page and responds on the Discord channel.
func handleSubmissionURL(ctx bot.Context, match *fetch.SubmissionURLMatch) {
	ctx.Logger.Info("Processing submission URL: ", match.URL)

	submissionInfo, err := fetch.Submission(ctx.Ctx, match.URL)
	if err != nil {
		err = fmt.Errorf("Error fetching submission from %v: %w", match.URL, err)
		ctx.Logger.Error(err)
		ctx.SendTimed(timedErrorMsgTTL, ctx.MakeErrorEmbed(err.Error()))
		return
	}

	content := ""
	var embed *disgord.Embed = nil
	if match.LineBegin == 0 {
		embed = makeSubmissionEmbed(submissionInfo)
	} else {
		content, err = makeCodeSnippet(
			submissionInfo.Content, submissionInfo.Language, match.LineBegin, match.LineEnd)
		if err != nil {
			ctx.SendTimed(timedErrorMsgTTL, ctx.MakeErrorEmbed(err.Error()))
			return
		}
	}

	// Allow the author to delete the preview.
	_, err = ctx.SendWithDelBtn(bot.OnePageWithDelParams{
		Content:         content,
		Embed:           embed,
		DeactivateAfter: time.Minute,
		DelCallback: func(evt *disgord.MessageReactionAdd) {
			// This will fail without manage messages permission, that's fine.
			bot.UnsuppressEmbeds(evt.Ctx, ctx.Session, ctx.Message)
		},
		AllowOp: func(evt *disgord.MessageReactionAdd) bool {
			return evt.UserID == ctx.Message.Author.ID
		},
	})
	if err != nil {
		ctx.Logger.Error(fmt.Errorf("Error sending problem info: %w", err))
		return
	}

	// This will fail without manage messages permission, that's fine.
	bot.SuppressEmbeds(ctx.Ctx, ctx.Session, ctx.Message)
}

func makeSubmissionEmbed(s *fetch.SubmissionInfo) *disgord.Embed {
	prefix := ""
	if s.Verdict == "Accepted" {
		prefix = "✅ "
	}
	return &disgord.Embed{
		Title:       "Submission for " + s.Problem + " by " + s.AuthorHandle,
		URL:         s.URL,
		Color:       s.AuthorColor,
		Description: prefix + s.Verdict + " • " + s.Type + " • " + s.Language,
		Timestamp:   disgord.Time{Time: s.SentTime},
	}
}

func makeCodeSnippet(code, language string, begin, end int) (string, error) {
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
		return "", errors.New("Selected lines are empty")
	}

	return "```" + languageNameToCode[language] + "\n" + strings.Join(lines, "\n") + "```", nil
}

func clamp(x, low, high int) int {
	if x < low {
		x = low
	} else if x > high {
		x = high
	}
	return x
}