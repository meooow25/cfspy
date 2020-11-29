package main

import (
	"flag"
	"os"

	"github.com/andersfylling/disgord"
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
	serverCountFeature := flag.Bool("scf", false, "install the server count feature")
	flag.Parse()

	logger.Info("------------ CFSpy starting ------------")
	defer logger.Info("------------ CFSpy stopped ------------")

	b := bot.New(
		bot.Info{
			Config: disgord.Config{
				BotToken: token,
				Logger:   logger,

				// Ideally we would just set config.Intents, but somewhat misleadingly Disgord
				// doesn't use intents for anything except to *add* DM message support. The reject
				// events list can be used instead, which gets translated to intents.
				// https://github.com/andersfylling/disgord/blob/5c80ec9176ee57789c5018848aa894d1175065eb/internal/gateway/eventclient.go#L37-L51
				RejectEvents: disgord.AllEventsExcept(relevantEvents()...),
			},
			Name:       "CFSpy",
			Prefix:     "c;",
			Desc:       "Codeforces Spy watches for Codeforces links and shows a preview.",
			SupportURL: supportURL,
		},
	)

	installPingCfCommand(b)
	installFeatureInfoCommand(b)
	installPingCommand(b)

	installStatusFeature(b)

	installBlogAndCommentFeature(b)
	installProblemFeature(b)
	installSubmissionFeature(b)

	if *serverCountFeature {
		installServerCountFeature(b)
	}

	b.Client.Gateway().StayConnectedUntilInterrupted()
}

func relevantEvents() []string {
	// https://discord.com/developers/docs/topics/gateway#list-of-intents
	return []string{
		// Guilds intent
		disgord.EvtGuildCreate,
		disgord.EvtGuildUpdate,
		disgord.EvtGuildDelete,
		disgord.EvtGuildRoleCreate,
		disgord.EvtGuildRoleUpdate,
		disgord.EvtGuildRoleDelete,
		disgord.EvtChannelCreate,
		disgord.EvtChannelUpdate,
		disgord.EvtChannelDelete,
		disgord.EvtChannelPinsUpdate,
		// Guild messages intent
		disgord.EvtMessageCreate,
		disgord.EvtMessageUpdate,
		disgord.EvtMessageDelete,
		disgord.EvtMessageDeleteBulk,
		// Guild message reactions intent
		disgord.EvtMessageReactionAdd,
		disgord.EvtMessageReactionRemove,
		disgord.EvtMessageReactionRemoveAll,
		disgord.EvtMessageReactionRemoveEmoji,
	}
}
