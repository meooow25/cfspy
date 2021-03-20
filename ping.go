package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/meooow25/cfspy/bot"
)

const (
	cfHomeURL = "https://codeforces.com"
	timeout   = 5 * time.Second
)

func onPingCf(ctx *bot.Context) {
	go func() {
		if len(ctx.Args) > 1 {
			ctx.SendIncorrectUsageMsg()
			return
		}

		timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		req, err := http.NewRequestWithContext(timeoutCtx, http.MethodHead, cfHomeURL, nil)
		if err != nil {
			ctx.Logger.Error(err) // No reason for new request to fail.
			return
		}

		start := time.Now()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			err := err.(*url.Error)
			if err.Timeout() {
				ctx.Send(fmt.Sprintf("Connecting to <%v> timed out after %v", cfHomeURL, timeout))
			} else {
				ctx.Send(fmt.Sprintf("Error: %v", err))
			}
			return
		}
		lat := time.Since(start).Round(time.Millisecond)
		ctx.Send(fmt.Sprintf("Pinged <%v>: Response %v, Latency %v", cfHomeURL, resp.Status, lat))
	}()
}

func onPing(ctx *bot.Context) {
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

// Installs the pingcf command.
func installPingCfCommand(b *bot.Bot) {
	b.Client.Logger().Info("Setting up pingcf command")
	b.AddCommand(&bot.Command{
		ID:          "pingcf",
		Description: "Checks the latency of codeforces.com",
		Handler:     onPingCf,
	})
}

// Installs the ping command.
func installPingCommand(b *bot.Bot) {
	b.Client.Logger().Info("Setting up ping command")
	b.AddCommand(&bot.Command{
		ID:          "ping",
		Description: "Checks the latency of the Discord REST API",
		Handler:     onPing,
	})
}
