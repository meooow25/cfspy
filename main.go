package main

import (
	"context"
	"os"

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

	bot := NewBot(
		BotInfo{
			Name:   "CFSpy",
			Token:  token,
			Prefix: "c;",
			Desc: "Codeforces Spy watches for Codeforces comment links and shows a preview.\n" +
				"Supported commands:",
			Logger: logger,
		},
	)

	InstallPingFeature(bot)
	InstallCfCommentFeature(bot)
	InstallStatusFeature(bot)

	bot.Client.StayConnectedUntilInterrupted(context.Background())

	logger.Info("CFSpy stopped")
}
