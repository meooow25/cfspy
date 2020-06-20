package main

import "github.com/andersfylling/disgord"

var updateStatusPayload = &disgord.UpdateStatusPayload{
	Game: &disgord.Activity{
		Name: "for Codeforces comments | c;help for info",
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

// InstallStatusFeature installs the staus feature, which updates the bot's status on ready and on
// resume.
func InstallStatusFeature(bot *Bot) {
	bot.Client.Logger().Info("Setting up status feature")
	bot.Client.On(disgord.EvtReady, setStatus)
	bot.Client.On(disgord.EvtResumed, setStatus)
}
