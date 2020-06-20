package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const cfHomeURL = "https://codeforces.com"

var pingCfClient = http.Client{Timeout: 5 * time.Second}

func onPingCf(ctx BotContext) {
	go func() {
		if len(ctx.Args) > 1 {
			ctx.SendIncorrectUsageMsg()
			return
		}

		start := time.Now()
		resp, err := pingCfClient.Head(cfHomeURL)
		if err != nil {
			err := err.(*url.Error)
			if err.Timeout() {
				ctx.Send(fmt.Sprintf(
					"Connecting to <%v> timed out after %v", cfHomeURL, pingCfClient.Timeout))
			} else {
				ctx.Send(fmt.Sprintf("Error: %v", err))
			}
			return
		}
		lat := time.Since(start).Round(time.Millisecond)
		ctx.Send(fmt.Sprintf("Pinged <%v>: Response %v, Latency %v", cfHomeURL, resp.Status, lat))
	}()
}

func onPing(ctx BotContext) {
	go func() {
		if len(ctx.Args) > 1 {
			ctx.SendIncorrectUsageMsg()
			return
		}
		start := time.Now()
		pongMsg, err := ctx.Send("pong!")
		if err != nil {
			return
		}
		lat := time.Since(start).Round(time.Millisecond)
		ctx.EditMsg(pongMsg, fmt.Sprintf("Latency %v", lat))
	}()
}

// InstallPingFeature installs the ping and cfping commands.
func InstallPingFeature(bot *Bot) {
	bot.Client.Logger().Info("Setting up ping feature")
	bot.AddCommand(&Command{
		ID:      "ping",
		Desc:    "Checks the latency of the Discord REST API",
		Handler: onPing,
	})
	bot.AddCommand(&Command{
		ID:      "pingcf",
		Desc:    "Checks the latency of codeforces.com",
		Handler: onPingCf,
	})
}
