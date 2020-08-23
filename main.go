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
			Name:   "CFSpy",
			Token:  token,
			Prefix: "c;",
			Desc: "Codeforces Spy watches for Codeforces comment, blog and problem links and " +
				"shows a preview.\n" +
				"Supported commands:",
			SupportURL: supportURL,
			Logger:     logger,
		},
	)

	installPingFeature(b)
	installBlogAndCommentFeature(b)
	installProblemFeature(b)
	installStatusFeature(b)

	b.Client.StayConnectedUntilInterrupted(context.Background())

	logger.Info("CFSpy stopped")
}
