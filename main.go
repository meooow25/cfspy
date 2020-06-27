package main

import (
	"context"
	"os"

	"github.com/meooow25/cfspy/bot"
	"github.com/sirupsen/logrus"
)

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
			Desc: "Codeforces Spy watches for Codeforces comment links and shows a preview.\n" +
				"Supported commands:",
			Logger: logger,
		},
	)

	installPingFeature(b)
	installCfCommentFeature(b)
	installCfBlogFeature(b)
	installStatusFeature(b)

	b.Client.StayConnectedUntilInterrupted(context.Background())

	logger.Info("CFSpy stopped")
}
