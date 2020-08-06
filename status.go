package main

import (
	"github.com/andersfylling/disgord"
	"github.com/meooow25/cfspy/bot"
)

var updateStatusPayload = &disgord.UpdateStatusPayload{
	Game: &disgord.Activity{
		Name: "for Codeforces comment and blog links | c;help for info",
		Type: 3, // Watching activity type
	},
}

func setStatus(s disgord.Session) {
	go func() {
		s.Logger().Info("Updating status")
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
	b.Client.On(disgord.EvtReady, setStatus)
	b.Client.On(disgord.EvtResumed, setStatus)
}
