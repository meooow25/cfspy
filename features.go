package main

import (
	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
)

const featureInfo = "CFSpy watches for Codeforces links. It can\n" +
	"- Watch for blog links and show some basic information about the blog.\n" +
	"- Watch for comment links and show the comment.\n" +
	"- Watch for problem links and show some basic information about the problem.\n" +
	"- Watch for submission links and show some basic information about the submission or show a " +
	"snippet from the submission. Showing a snippet requires line numbers, for which you may " +
	"install this [userscript](https://greasyfork.org/en/scripts/403747-cf-linemaster)."

func onFeatureInfo(ctx bot.Context) {
	embed := disgord.Embed{
		Author:      &disgord.EmbedAuthor{Name: "Features"},
		Description: featureInfo,
	}
	ctx.Send(embed)
}

// Installs the features command.
func installFeatureInfoCommand(b *bot.Bot) {
	b.Client.Logger().Info("Setting up features command")
	b.AddCommand(&bot.Command{
		ID:      "features",
		Desc:    "Shows information about automatic features",
		Handler: onFeatureInfo,
	})
}
