package main

import (
	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
)

var updateStatusPayload = &disgord.UpdateStatusPayload{
	Game: &disgord.Activity{
		Name: "for Codeforces links | c;help for info",
		Type: 3, // Watching activity type
	},
}

func setStatus(s disgord.Session) {
	go func() {
		err := s.UpdateStatus(updateStatusPayload)
		if err != nil {
			s.Logger().Error("Error setting status: ", err)
		}
	}()
}

// Installs the staus feature, which updates the bot's status on ready and on
// resume.
func installStatusFeature(b *bot.Bot) {
	b.Client.Logger().Info("Setting up status feature")
	b.Client.Gateway().BotReady(func() { setStatus(b.Client) })
	b.Client.Gateway().Resumed(func(_ disgord.Session, _ *disgord.Resumed) { setStatus(b.Client) })
}
