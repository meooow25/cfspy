package main

import (
	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
)

const featureInfo = "CFSpy watches for Codeforces links and shows helpful previews.\n" +
	"Supported links include\n" +
	"- _Blogs_: Shows some information about the blog.\n" +
	"- _Comments_: Shows the comment information and content.\n" +
	"- _Problems_: Shows some information about the problem.\n" +
	"- _Profiles_: Shows some information about the user profile.\n" +
	"- _Submissions_: Shows some information about the submission.\n" +
	"- _Submissions with line numbers_: Shows a snippet from the submission containing the " +
	"specified lines. For line selection and highlighting support in your browser you may want " +
	"to install this [userscript](https://greasyfork.org/en/scripts/403747-cf-linemaster)."

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
		ID:          "features",
		Description: "Shows information about automatic features",
		Handler:     onFeatureInfo,
	})
}
