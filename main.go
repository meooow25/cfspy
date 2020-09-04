package main

import (
	"context"
	"os"

	"github.com/meooow25/cfspy/bot"
	"github.com/sirupsen/logrus"
)

const supportURL = "https://github.com/meooow25/cfspy/issues"

var token = os.Getenv("TOKEN")
var logger = logrus.New()

func init() {
	logger.Formatter.(*logrus.TextFormatter).FullTimestamp = true
	if token == "" {
		logger.Fatal("TOKEN env var missing")
	}
}

func main() {
	logger.Info("CFSpy starting")

	b := bot.New(
		bot.Info{
			Name:       "CFSpy",
			Token:      token,
			Prefix:     "c;",
			Desc:       "Codeforces Spy watches for Codeforces links and shows a preview.",
			SupportURL: supportURL,
			Logger:     logger,
		},
	)

	installPingCfCommand(b)
	installFeatureInfoCommand(b)
	installPingCommand(b)

	installStatusFeature(b)

	installBlogAndCommentFeature(b)
	installProblemFeature(b)
	installSubmissionFeature(b)

	b.Client.StayConnectedUntilInterrupted(context.Background())

	logger.Info("CFSpy stopped")
}
