package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/google/go-github/v32/github"
	"github.com/meooow25/cfspy/bot"
	"golang.org/x/oauth2"
)

var (
	githubToken = os.Getenv("GITHUB_TOKEN")
	gistID      = os.Getenv("GIST_ID")
)

const (
	gistFilename = "server-count.json"
	gistJSONFmt  = `{"serverCount": %v}`
)

const refreshInterval = 15 * time.Minute

func updateServerCount(s disgord.Session, gc *github.Client) {
	ctx := context.Background()

	gist, _, err := gc.Gists.Get(ctx, gistID)
	if err != nil {
		s.Logger().Error("Error getting gist: ", err)
		return
	}
	curContent := *gist.Files[gistFilename].Content

	guilds, err := s.GetCurrentUserGuilds(ctx, nil)
	if err != nil {
		s.Logger().Error("Error fetching guilds: ", err)
		return
	}
	newContent := fmt.Sprintf(gistJSONFmt, len(guilds))
	if newContent == curContent {
		return
	}

	s.Logger().Info("Updating server count: ", newContent)
	_, _, err = gc.Gists.Edit(ctx, gistID, &github.Gist{
		Files: map[github.GistFilename]github.GistFile{
			gistFilename: {Content: &newContent},
		},
	})
	if err != nil {
		s.Logger().Error("Error updating gist: ", err)
	}
}

func startServerCountTask(s disgord.Session) {
	go func() {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
		tc := oauth2.NewClient(context.Background(), ts)
		gc := github.NewClient(tc)
		for {
			updateServerCount(s, gc)
			time.Sleep(refreshInterval)
		}
	}()
}

// Installs the server count feature, which updates a Github Gist with the bot's server count at
// regular intervals.
func installServerCountFeature(b *bot.Bot) {
	b.Client.Logger().Info("Setting up server count feature")

	if githubToken == "" {
		panic("GITHUB_TOKEN env var missing")
	}
	if gistID == "" {
		panic("GIST_ID env var missing")
	}

	b.Client.On(disgord.EvtReady, startServerCountTask)
}
